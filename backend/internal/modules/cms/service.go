package cms

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
)

type Service interface {
	Create(ctx context.Context, name string, actorID int64) (*Item, error)
	GetByID(ctx context.Context, id int64) (*Item, error)
	List(ctx context.Context, limit int, cursor int64) ([]Item, error)
	Update(ctx context.Context, id int64, name string, actorID int64) (*Item, error)
	Delete(ctx context.Context, id int64, actorID int64) error
}

type service struct {
	db           *sql.DB
	repo         Repository
	auditService audit.Service
}

func NewService(db *sql.DB, repo Repository, auditService audit.Service) Service {
	return &service{db: db, repo: repo, auditService: auditService}
}

func (s *service) Create(ctx context.Context, name string, actorID int64) (*Item, error) {
	item := &Item{Name: name}

	actx := database.WithAuditCtx(ctx, actorID, "CREATE_CMS")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Create(txCtx, item)
	})
	if err != nil {
		return nil, fmt.Errorf("create cms: %w", err)
	}
	return item, nil
}

func (s *service) GetByID(ctx context.Context, id int64) (*Item, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context, limit int, cursor int64) ([]Item, error) {
	return s.repo.List(ctx, limit, cursor)
}

func (s *service) Update(ctx context.Context, id int64, name string, actorID int64) (*Item, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, database.ErrNotFound
	}
	item.Name = name

	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_CMS")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Update(txCtx, item)
	})
	if err != nil {
		return nil, fmt.Errorf("update cms: %w", err)
	}
	return item, nil
}

func (s *service) Delete(ctx context.Context, id int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_CMS")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Delete(txCtx, id)
	})
}
