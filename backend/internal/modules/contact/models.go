package contact

import (
	"encoding/json"
	"time"
)

type Contact struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	Subject    string    `json:"subject"`
	Message    string    `json:"message"`
	Source     string    `json:"source"`
	Status     string    `json:"status"`
	AssignedTo *int64    `json:"assigned_to,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type NewsletterSubscriber struct {
	ID             int64           `json:"id"`
	Email          string          `json:"email"`
	Name           string          `json:"name"`
	Source         string          `json:"source"`
	Metadata       json.RawMessage `json:"metadata"`
	SubscribedAt   time.Time       `json:"subscribed_at"`
	UnsubscribedAt *time.Time      `json:"unsubscribed_at,omitempty"`
}
