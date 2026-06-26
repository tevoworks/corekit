package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sony/gobreaker"
)

type WebhookPayload struct {
	TargetURL   string         `json:"target_url"`
	Signature   string         `json:"signature,omitempty"`
	Payload     map[string]any `json:"payload"`
	ResolvedIPs []string       `json:"resolved_ips,omitempty"`
}

type WebhookDispatcher struct {
	client   *http.Client
	cbMu     sync.RWMutex
	cbs      map[string]*gobreaker.CircuitBreaker
	stopCh   chan struct{}
	stopOnce sync.Once
}

func NewWebhookDispatcher() *WebhookDispatcher {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	d := &WebhookDispatcher{
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					host, _, err := net.SplitHostPort(addr)
					if err != nil {
						return nil, err
					}
					ips, err := net.LookupIP(host)
					if err != nil {
						return nil, err
					}
					for _, ip := range ips {
						if isRestrictedIP(ip) {
							return nil, errors.New("webhook target resolves to a restricted IP address")
						}
					}
					return dialer.DialContext(ctx, network, addr)
				},
			},
		},
		cbs:    make(map[string]*gobreaker.CircuitBreaker),
		stopCh: make(chan struct{}),
	}
	go d.periodicCleanup()
	return d
}

func (wd *WebhookDispatcher) Stop() {
	wd.stopOnce.Do(func() {
		close(wd.stopCh)
	})
}

func (wd *WebhookDispatcher) periodicCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-wd.stopCh:
			return
		case <-ticker.C:
			wd.cbMu.Lock()
			for url, cb := range wd.cbs {
				counts := cb.Counts()
				if counts.Requests == 0 && counts.TotalSuccesses == 0 && counts.TotalFailures == 0 {
					delete(wd.cbs, url)
				}
			}
			wd.cbMu.Unlock()
		}
	}
}

func (wd *WebhookDispatcher) getCB(targetURL string) *gobreaker.CircuitBreaker {
	wd.cbMu.RLock()
	cb, exists := wd.cbs[targetURL]
	wd.cbMu.RUnlock()
	if exists {
		return cb
	}

	wd.cbMu.Lock()
	defer wd.cbMu.Unlock()

	if cb, exists = wd.cbs[targetURL]; exists {
		return cb
	}

	settings := gobreaker.Settings{
		Name:        "WebhookDispatcher:" + targetURL,
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	}
	cb = gobreaker.NewCircuitBreaker(settings)
	wd.cbs[targetURL] = cb
	return cb
}

func ipsMatch(current, stored []string) bool {
	if len(current) != len(stored) {
		return false
	}
	storedSet := make(map[string]bool, len(stored))
	for _, ip := range stored {
		storedSet[ip] = true
	}
	for _, ip := range current {
		if !storedSet[ip] {
			return false
		}
	}
	return true
}

func isRestrictedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate() || ip.IsUnspecified() {
		return true
	}
	// Carrier-grade NAT (CGNAT) / shared address space
	if ip := ip.To4(); ip != nil {
		if ip[0] == 100 && ip[1] >= 64 && ip[1] <= 127 {
			return true
		}
	}
	return false
}

func (wd *WebhookDispatcher) Execute(ctx context.Context, payload []byte) error {
	var wp WebhookPayload
	if err := json.Unmarshal(payload, &wp); err != nil {
		return err
	}

	if wp.TargetURL == "" {
		return errors.New("webhook target_url is empty")
	}

	u, err := url.Parse(wp.TargetURL)
	if err != nil {
		return err
	}

	if u.Scheme != "https" {
		return errors.New("invalid URL scheme, only https is allowed")
	}

	host := u.Hostname()
	if host == "" {
		return errors.New("invalid or empty host in target URL")
	}

	if len(wp.ResolvedIPs) > 0 {
		currentIPs, err := net.LookupHost(host)
		if err != nil {
			return errors.New("webhook target DNS resolution failed at dispatch time")
		}
		if !ipsMatch(currentIPs, wp.ResolvedIPs) {
			return errors.New("webhook target IP has changed since creation — possible DNS rebinding attack")
		}
	}

	bodyBytes, err := json.Marshal(wp.Payload)
	if err != nil {
		return err
	}

	cb := wd.getCB(wp.TargetURL)
	_, err = cb.Execute(func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, wp.TargetURL, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", uuid.New().String())

		if wp.Signature != "" {
			req.Header.Set("X-Signature-256", wp.Signature)
		}

		resp, err := wd.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, errors.New("webhook responded with non-2xx status code: " + resp.Status)
		}
		return nil, nil
	})

	return err
}
