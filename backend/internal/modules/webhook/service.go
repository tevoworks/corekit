package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/url"
	"time"

	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
	"github.com/tevoworks/corekit/backend/internal/modules/queue"
)

var privateCIDRs []*net.IPNet

func init() {
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
		"169.254.0.0/16",
		"100.64.0.0/10",
		"198.18.0.0/15",
	}
	for _, cidr := range privateBlocks {
		_, block, err := net.ParseCIDR(cidr)
		if err == nil {
			privateCIDRs = append(privateCIDRs, block)
		}
	}
}

func validateWebhookURL(rawURL string) ([]string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.New("invalid webhook URL")
	}

	if u.Scheme != "https" {
		return nil, errors.New("webhook URL must use HTTPS")
	}

	host := u.Hostname()
	if host == "" {
		return nil, errors.New("webhook URL must have a host")
	}

	ips, lookupErr := net.LookupHost(host)
	if lookupErr != nil {
		slog.Warn("webhook URL host DNS resolution failed (allowing creation, SSRF check at dispatch time)", "host", host, "error", lookupErr)
		return nil, nil
	}

	var resolved []string
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		for _, block := range privateCIDRs {
			if block.Contains(ip) {
				return nil, errors.New("webhook URL must not point to a private or internal IP address")
			}
		}
		resolved = append(resolved, ipStr)
	}

	return resolved, nil
}

type Service interface {
	Create(ctx context.Context, actorID int64, name, url string, events []string, secret string, active bool) (*Webhook, error)
	Update(ctx context.Context, id, actorID int64, name, url string, events []string, secret string, active bool) (*Webhook, error)
	GetByID(ctx context.Context, id int64) (*Webhook, error)
	List(ctx context.Context, limit int, cursor int64) ([]Webhook, error)
	Delete(ctx context.Context, id, actorID int64) error

	ListDeliveries(ctx context.Context, webhookID int64, limit int, cursor int64) ([]WebhookDelivery, error)
	GetDeliveryByID(ctx context.Context, webhookID, id int64) (*WebhookDelivery, error)
	RetryDelivery(ctx context.Context, webhookID, deliveryID, actorID int64) error
	TestWebhook(ctx context.Context, id, actorID int64) error
}

type service struct {
	db                 *sql.DB
	repo               Repository
	queueRepo          queue.Repository
	auditService       audit.Service
	encryptionKey      []byte
}

func NewService(db *sql.DB, repo Repository, queueRepo queue.Repository, auditService audit.Service, encryptionKey string) Service {
	var key []byte
	if encryptionKey != "" {
		k, err := hex.DecodeString(encryptionKey)
		if err == nil && len(k) == 32 {
			key = k
		} else {
			slog.Warn("invalid webhook encryption key, falling back to plaintext", "error", err)
		}
	}
	return &service{
		db:                 db,
		repo:               repo,
		queueRepo:          queueRepo,
		auditService:       auditService,
		encryptionKey:      key,
	}
}

func generateSecret() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *service) Create(ctx context.Context, actorID int64, name, url string, events []string, secret string, active bool) (*Webhook, error) {
	resolvedIPs, err := validateWebhookURL(url)
	if err != nil {
		return nil, err
	}
	if secret == "" {
		secret = generateSecret()
	}
	encryptedSecret, err := encryptAESGCM(s.encryptionKey, secret)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []string{}
	}
	wh := &Webhook{
		Name:        name,
		URL:         url,
		Events:      events,
		Secret:      encryptedSecret,
		Active:      active,
		CreatedBy:   actorID,
		ResolvedIPs: resolvedIPs,
	}

	actx := database.WithAuditCtx(ctx, actorID, "CREATE_WEBHOOK")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Create(txCtx, wh)
	})
	if err != nil {
		return nil, err
	}
	wh.RawSecret = secret
	wh.MaskSecret()
	wh.Secret = secret
	return wh, nil
}

func (s *service) decryptSecret(encrypted string) string {
	decrypted, err := decryptAESGCM(s.encryptionKey, encrypted)
	if err != nil {
		slog.Error("failed to decrypt webhook secret", "error", err)
		return encrypted
	}
	return decrypted
}

func (s *service) Update(ctx context.Context, id, actorID int64, name, url string, events []string, secret string, active bool) (*Webhook, error) {
	resolvedIPs, err := validateWebhookURL(url)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []string{}
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("webhook not found")
	}

	if secret == "" {
		secret = s.decryptSecret(existing.Secret)
	}
	encryptedSecret, err := encryptAESGCM(s.encryptionKey, secret)
	if err != nil {
		return nil, err
	}

	wh := &Webhook{
		ID:          id,
		Name:        name,
		URL:         url,
		Events:      events,
		Secret:      encryptedSecret,
		Active:      active,
		ResolvedIPs: resolvedIPs,
	}

	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_WEBHOOK")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Update(txCtx, wh)
	})
	if err != nil {
		return nil, err
	}
	wh.Secret = secret
	wh.MaskSecret()
	return wh, nil
}

func (s *service) GetByID(ctx context.Context, id int64) (*Webhook, error) {
	wh, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if wh != nil {
		wh.Secret = s.decryptSecret(wh.Secret)
		wh.MaskSecret()
	}
	return wh, nil
}

func (s *service) List(ctx context.Context, limit int, cursor int64) ([]Webhook, error) {
	list, err := s.repo.List(ctx, limit, cursor)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Secret = s.decryptSecret(list[i].Secret)
		list[i].MaskSecret()
	}
	if list == nil {
		return []Webhook{}, nil
	}
	return list, nil
}

func (s *service) Delete(ctx context.Context, id, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_WEBHOOK")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Delete(txCtx, id)
	})
}

func (s *service) ListDeliveries(ctx context.Context, webhookID int64, limit int, cursor int64) ([]WebhookDelivery, error) {
	return s.repo.ListDeliveries(ctx, webhookID, limit, cursor)
}

func (s *service) GetDeliveryByID(ctx context.Context, webhookID, id int64) (*WebhookDelivery, error) {
	return s.repo.GetDeliveryByID(ctx, id, webhookID)
}

func (s *service) RetryDelivery(ctx context.Context, webhookID, deliveryID, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "RETRY_WEBHOOK_DELIVERY")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		wh, err := s.repo.GetByID(txCtx, webhookID)
		if err != nil {
			return err
		}
		if wh == nil {
			return errors.New("webhook not found")
		}

		d, err := s.repo.GetDeliveryByID(txCtx, deliveryID, webhookID)
		if err != nil {
			return err
		}
		if d == nil {
			return errors.New("delivery not found")
		}

		if err := s.repo.SetDeliveryRetrying(txCtx, deliveryID); err != nil {
			return err
		}

		var originalPayload map[string]interface{}
		if d.RequestBody != nil {
			_ = json.Unmarshal([]byte(*d.RequestBody), &originalPayload)
		}
		if originalPayload == nil {
			originalPayload = map[string]interface{}{
				"event":       d.Event,
				"webhook_id":  webhookID,
				"delivery_id": deliveryID,
			}
		}

		rawSecret := s.decryptSecret(wh.Secret)
		origPayloadBytes, _ := json.Marshal(originalPayload)
		mac := hmac.New(sha256.New, []byte(rawSecret))
		mac.Write(origPayloadBytes)
		sig := hex.EncodeToString(mac.Sum(nil))

		payload, _ := json.Marshal(map[string]interface{}{
			"target_url":   wh.URL,
			"signature":    sig,
			"payload":      originalPayload,
			"resolved_ips": wh.ResolvedIPs,
		})
		return s.queueRepo.Enqueue(txCtx, database.GetTx(txCtx), queue.JobTypeWebhookDispatch, payload, nil)
	})
}

func (s *service) TestWebhook(ctx context.Context, id, actorID int64) error {
	wh, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if wh == nil {
		return errors.New("webhook not found")
	}

	now := time.Now()
	delivery := &WebhookDelivery{
		WebhookID: id,
		Event:     "test",
		Status:    DeliveryStatusPending,
	}

	err = database.RunInTransaction(ctx, s.db, func(txCtx context.Context) error {
		if err := s.repo.CreateDelivery(txCtx, delivery); err != nil {
			return err
		}

		initialPayload := map[string]interface{}{
			"event":       "test",
			"webhook_id":  id,
			"delivery_id": delivery.ID,
			"test":        true,
			"timestamp":   now,
		}
		payloadBody, _ := json.Marshal(initialPayload)
		bodyStr := string(payloadBody)
		delivery.RequestBody = &bodyStr

		rawSecret := s.decryptSecret(wh.Secret)
		payloadBodyBytes, _ := json.Marshal(initialPayload)
		mac := hmac.New(sha256.New, []byte(rawSecret))
		mac.Write(payloadBodyBytes)
		sig := hex.EncodeToString(mac.Sum(nil))

		payload, _ := json.Marshal(map[string]interface{}{
			"target_url":   wh.URL,
			"signature":    sig,
			"payload":      initialPayload,
			"delivery_id":  delivery.ID,
			"resolved_ips": wh.ResolvedIPs,
		})
		return s.queueRepo.Enqueue(txCtx, database.GetTx(txCtx), queue.JobTypeWebhookDispatch, payload, nil)
	})
	return err
}
