package contact

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
)

type Service interface {
	SubmitContact(ctx context.Context, name, email, phone, subject, message, source string) (*Contact, error)
	Subscribe(ctx context.Context, email, name, source string) (*NewsletterSubscriber, error)
	Unsubscribe(ctx context.Context, email string) error

	ListContacts(ctx context.Context, limit, cursor int64, status string) ([]Contact, error)
	GetContact(ctx context.Context, id int64) (*Contact, error)
	UpdateContactStatus(ctx context.Context, id, actorID int64, status string) error
	AssignContact(ctx context.Context, id, userID, actorID int64) error
	DeleteContact(ctx context.Context, id, actorID int64) error

	ListSubscribers(ctx context.Context, limit, cursor int64) ([]NewsletterSubscriber, error)
	DeleteSubscriber(ctx context.Context, id, actorID int64) error
}

type service struct {
	db           *sql.DB
	repo         Repository
	auditService audit.Service
}

func NewService(db *sql.DB, repo Repository, auditService audit.Service) Service {
	return &service{db: db, repo: repo, auditService: auditService}
}

func (s *service) SubmitContact(ctx context.Context, name, email, phone, subject, message, source string) (*Contact, error) {
	c := &Contact{
		Name:    name,
		Email:   email,
		Phone:   phone,
		Subject: subject,
		Message: message,
		Source:  source,
		Status:  "new",
	}
	err := database.RunInTransaction(ctx, s.db, func(txCtx context.Context) error {
		return s.repo.CreateContact(txCtx, c)
	})
	if err != nil {
		return nil, fmt.Errorf("submit contact: %w", err)
	}
	return c, nil
}

func (s *service) Subscribe(ctx context.Context, email, name, source string) (*NewsletterSubscriber, error) {
	sub := &NewsletterSubscriber{
		Email:  email,
		Name:   name,
		Source: source,
	}
	err := database.RunInTransaction(ctx, s.db, func(txCtx context.Context) error {
		return s.repo.CreateSubscriber(txCtx, sub)
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}
	return sub, nil
}

func (s *service) Unsubscribe(ctx context.Context, email string) error {
	return database.RunInTransaction(ctx, s.db, func(txCtx context.Context) error {
		return s.repo.Unsubscribe(txCtx, email)
	})
}

func (s *service) ListContacts(ctx context.Context, limit, cursor int64, status string) ([]Contact, error) {
	return s.repo.ListContacts(ctx, limit, cursor, status)
}

func (s *service) GetContact(ctx context.Context, id int64) (*Contact, error) {
	return s.repo.GetContactByID(ctx, id)
}

func (s *service) UpdateContactStatus(ctx context.Context, id, actorID int64, status string) error {
	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_CONTACT_STATUS")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		existing, err := s.repo.GetContactByID(txCtx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return database.ErrNotFound
		}
		return s.repo.UpdateContactStatus(txCtx, id, status)
	})
}

func (s *service) AssignContact(ctx context.Context, id, userID, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "ASSIGN_CONTACT")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		existing, err := s.repo.GetContactByID(txCtx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return database.ErrNotFound
		}
		return s.repo.AssignContact(txCtx, id, userID)
	})
}

func (s *service) DeleteContact(ctx context.Context, id, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_CONTACT")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.DeleteContact(txCtx, id)
	})
}

func (s *service) ListSubscribers(ctx context.Context, limit, cursor int64) ([]NewsletterSubscriber, error) {
	return s.repo.ListSubscribers(ctx, limit, cursor)
}

func (s *service) DeleteSubscriber(ctx context.Context, id, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_SUBSCRIBER")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.DeleteSubscriber(txCtx, id)
	})
}
