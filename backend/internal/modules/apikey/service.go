package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	CreateKey(ctx context.Context, createdBy int64, name string) (*APIKey, string, error)
	ListKeys(ctx context.Context) ([]APIKey, error)
	RevokeKey(ctx context.Context, id, actorID int64) error
	ValidateKey(ctx context.Context, rawKey string) (*APIKey, error)
	RotateKey(ctx context.Context, id, actorID int64) (*APIKey, string, error)
	RotateExpiringKeys(ctx context.Context) (int, error)
}

type service struct {
	repo         Repository
	db           *sql.DB
	auditService audit.Service
}

func NewService(db *sql.DB, repo Repository, auditService audit.Service) Service {
	return &service{
		db:           db,
		repo:         repo,
		auditService: auditService,
	}
}

func generateAPIKey() (string, string, string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", "", "", err
	}
	rawKey := hex.EncodeToString(raw)
	prefixedKey := "ca_" + rawKey

	lookupHash := sha256.Sum256([]byte(prefixedKey))

	hashedKey, err := bcrypt.GenerateFromPassword([]byte(prefixedKey), 12)
	if err != nil {
		return "", "", "", "", err
	}

	prefix := prefixedKey[:10]
	return prefixedKey, string(hashedKey), hex.EncodeToString(lookupHash[:]), prefix, nil
}

func (s *service) CreateKey(ctx context.Context, createdBy int64, name string) (*APIKey, string, error) {
	rawKey, saltedHash, lookupHash, prefix, err := generateAPIKey()
	if err != nil {
		return nil, "", err
	}

	k := &APIKey{
		Name:          name,
		KeyPrefix:     prefix,
		KeyHash:       saltedHash,
		KeyLookupHash: lookupHash,
		CreatedBy:     createdBy,
		ExpiresAt:     time.Now().Add(90 * 24 * time.Hour),
	}

	actx := database.WithAuditCtx(ctx, createdBy, "CREATE_API_KEY")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Create(txCtx, k)
	})
	if err != nil {
		return nil, "", err
	}

	return k, rawKey, nil
}

func (s *service) RotateKey(ctx context.Context, id, actorID int64) (*APIKey, string, error) {
	rawKey, saltedHash, lookupHash, prefix, err := generateAPIKey()
	if err != nil {
		return nil, "", err
	}

	actx := database.WithAuditCtx(ctx, actorID, "ROTATE_API_KEY")
	newExpiry := time.Now().Add(90 * 24 * time.Hour)
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Rotate(txCtx, id, saltedHash, lookupHash, prefix, newExpiry)
	})
	if err != nil {
		return nil, "", err
	}

	k, err := s.repo.GetByHash(ctx, lookupHash)
	if err != nil {
		return nil, "", err
	}
	return k, rawKey, nil
}

func (s *service) RotateExpiringKeys(ctx context.Context) (int, error) {
	keys, err := s.repo.ListExpiring(ctx, 7)
	if err != nil {
		return 0, err
	}
	rotated := 0
	for _, key := range keys {
		_, _, err := s.RotateKey(ctx, key.ID, key.CreatedBy)
		if err != nil {
			slog.Error("failed to rotate expiring key", "key_id", key.ID, "error", err)
			continue
		}
		rotated++
		slog.Info("rotated expiring API key", "key_id", key.ID, "name", key.Name)
	}
	return rotated, nil
}

func (s *service) ListKeys(ctx context.Context) ([]APIKey, error) {
	keys, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	if keys == nil {
		return []APIKey{}, nil
	}
	return keys, nil
}

func (s *service) RevokeKey(ctx context.Context, id, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "REVOKE_API_KEY")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Revoke(txCtx, id)
	})

	return err
}

func (s *service) ValidateKey(ctx context.Context, rawKey string) (*APIKey, error) {
	prefixedKey := "ca_" + rawKey
	lookupHash := sha256.Sum256([]byte(prefixedKey))
	lookupKey := hex.EncodeToString(lookupHash[:])

	k, err := s.repo.GetByHash(ctx, lookupKey)
	if err != nil {
		return nil, err
	}
	if k == nil {
		return nil, fmt.Errorf("invalid or revoked API key")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(prefixedKey)); err != nil {
		return nil, fmt.Errorf("invalid or revoked API key")
	}

	if k.RevokedAt != nil {
		return nil, fmt.Errorf("invalid or revoked API key")
	}

	if time.Now().After(k.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	var userStatus string
	err = s.db.QueryRowContext(ctx, `SELECT status FROM users WHERE id = $1 AND deleted_at IS NULL`, k.CreatedBy).Scan(&userStatus)
	if err != nil || userStatus != "ACTIVE" {
		return nil, fmt.Errorf("invalid or revoked API key")
	}

	_ = s.repo.TouchLastUsed(ctx, k.ID)
	return k, nil
}
