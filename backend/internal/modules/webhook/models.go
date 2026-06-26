package webhook

import (
	"time"
)

const (
	DeliveryStatusPending  = "pending"
	DeliveryStatusSuccess  = "success"
	DeliveryStatusFailed   = "failed"
	DeliveryStatusRetrying = "retrying"
)

type Webhook struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret_masked"`
	RawSecret string    `json:"raw_secret,omitempty"`
	Active    bool      `json:"active"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ResolvedIPs []string `json:"-"`
}

func (w *Webhook) MaskSecret() {
	if len(w.Secret) > 8 {
		w.Secret = w.Secret[:8] + "..." + w.Secret[len(w.Secret)-4:]
	} else if w.Secret != "" {
		w.Secret = "****"
	}
}

type WebhookDelivery struct {
	ID           int64     `json:"id"`
	WebhookID    int64     `json:"webhook_id"`
	Event        string    `json:"event"`
	Status       string    `json:"status"`
	RequestBody  *string   `json:"request_body,omitempty"`
	ResponseBody *string   `json:"response_body,omitempty"`
	ResponseCode *int      `json:"response_code,omitempty"`
	DurationMs   *int      `json:"duration_ms,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
