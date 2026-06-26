package event

import (
	"context"
	"database/sql"
	"encoding/json"
)

type EventRepository interface {
	Enqueue(ctx context.Context, tx *sql.Tx, eventType string, payload []byte, idempotencyKey *string) error
}

type EventDispatcher struct {
	repo EventRepository
}

func NewEventDispatcher(repo EventRepository) *EventDispatcher {
	return &EventDispatcher{repo: repo}
}

func (ed *EventDispatcher) Dispatch(ctx context.Context, tx *sql.Tx, eventType string, payload map[string]any) error {
	if ed.repo == nil {
		return nil
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return ed.repo.Enqueue(ctx, tx, eventType, payloadBytes, nil)
}
