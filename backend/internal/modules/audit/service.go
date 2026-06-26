package audit

import (
	"context"
)

type Service interface {
	GetLogs(ctx context.Context, limit int, cursor int64) ([]AuditLog, error)
	GetLogsFiltered(ctx context.Context, actorID *int64, action, dateFrom, dateTo string, limit int, cursor int64) ([]AuditLog, error)
	AnonymizeAuditPII(ctx context.Context, userID int64, email string) error
	PruneAuditLogs(ctx context.Context, retentionDays int) (int, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetLogs(ctx context.Context, limit int, cursor int64) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.List(ctx, limit, cursor)
}

func (s *service) AnonymizeAuditPII(ctx context.Context, userID int64, email string) error {
	return s.repo.AnonymizeAuditPII(ctx, userID, email)
}

func (s *service) GetLogsFiltered(ctx context.Context, actorID *int64, action, dateFrom, dateTo string, limit int, cursor int64) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListFiltered(ctx, actorID, action, dateFrom, dateTo, limit, cursor)
}

func (s *service) PruneAuditLogs(ctx context.Context, retentionDays int) (int, error) {
	return s.repo.PruneAuditLogs(ctx, retentionDays)
}
