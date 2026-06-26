package queue

import (
	"context"
	"time"
)

const (
	JobTypeEmailSend          = "EMAIL_SEND"
	JobTypeSecurityEventLog   = "SECURITY_EVENT_LOG"
	JobTypeWebhookDispatch    = "WEBHOOK_DISPATCH"
	JobTypeNotificationCreate = "NOTIFICATION_CREATE"
	StatusPending             = "pending"
	StatusProcessing          = "processing"
	StatusDone                = "done"
	StatusFailed              = "failed"
)

type Job struct {
	ID              int64      `json:"id"`
	Type            string     `json:"type"`
	Payload         []byte     `json:"payload"`
	Status          string     `json:"status"`
	RetryCount      int        `json:"retry_count"`
	MaxRetries      int        `json:"max_retries"`
	RunAfter        time.Time  `json:"run_after"`
	LockedAt        *time.Time `json:"locked_at,omitempty"`
	LockedBy        *string    `json:"locked_by,omitempty"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
	NextRetryAt     *time.Time `json:"next_retry_at,omitempty"`
	IdempotencyKey  *string    `json:"idempotency_key,omitempty"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type JobExecutor interface {
	Execute(ctx context.Context, payload []byte) error
}
