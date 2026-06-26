package audit

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID             int64           `json:"id"`
	ActorID        *int64          `json:"actor_id"`
	ImpersonatorID *int64          `json:"impersonator_id,omitempty"`
	Action         string          `json:"action"`
	TargetEntity   string          `json:"target_entity"`
	BeforeState    json.RawMessage `json:"before_state,omitempty"`
	AfterState     json.RawMessage `json:"after_state,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	ActorName      *string         `json:"actor_name,omitempty"`
	ActorEmail     *string         `json:"actor_email,omitempty"`
}
