package settings

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
)

type Service interface {
	SetSetting(ctx context.Context, key, value string, actorID int64) (*Setting, error)
	GetSetting(ctx context.Context, key string) (*Setting, error)
	ListSettings(ctx context.Context) ([]Setting, error)
	DeleteSetting(ctx context.Context, key string, actorID int64) error
	CreateFlag(ctx context.Context, name, key, description string, enabled bool, actorID int64) (*FeatureFlag, error)
	UpdateFlag(ctx context.Context, id int64, name, key, description string, enabled bool, actorID int64) (*FeatureFlag, error)
	DeleteFlag(ctx context.Context, id int64, actorID int64) error
	ListFlags(ctx context.Context, limit int, cursor int64) ([]FeatureFlag, error)
	LookupFlag(ctx context.Context, key string) (bool, error)
}

type cachedFlags struct {
	flags  []FeatureFlag
	expiry time.Time
}

type service struct {
	db           *sql.DB
	repo         Repository
	auditService audit.Service
	flagsMu      sync.RWMutex
	flagsCache   *cachedFlags
}

func NewService(db *sql.DB, repo Repository, auditService audit.Service) Service {
	return &service{
		db:           db,
		repo:         repo,
		auditService: auditService,
	}
}

func (s *service) getCachedFlags() []FeatureFlag {
	s.flagsMu.RLock()
	c := s.flagsCache
	s.flagsMu.RUnlock()
	if c != nil && time.Now().Before(c.expiry) {
		return c.flags
	}
	return nil
}

func (s *service) setCachedFlags(flags []FeatureFlag) {
	s.flagsMu.Lock()
	s.flagsCache = &cachedFlags{
		flags:  flags,
		expiry: time.Now().Add(60 * time.Second),
	}
	s.flagsMu.Unlock()
}

func (s *service) invalidateFlagsCache() {
	s.flagsMu.Lock()
	s.flagsCache = nil
	s.flagsMu.Unlock()
}

func (s *service) SetSetting(ctx context.Context, key, value string, actorID int64) (*Setting, error) {
	setting := &Setting{
		Key:   key,
		Value: value,
	}

	actx := database.WithAuditCtx(ctx, actorID, "SET_SETTING")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Set(txCtx, setting)
	})

	if err != nil {
		return nil, err
	}

	return setting, nil
}

func (s *service) GetSetting(ctx context.Context, key string) (*Setting, error) {
	return s.repo.Get(ctx, key)
}

func (s *service) ListSettings(ctx context.Context) ([]Setting, error) {
	return s.repo.List(ctx)
}

func (s *service) DeleteSetting(ctx context.Context, key string, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_SETTING")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Delete(txCtx, key)
	})
}

func (s *service) CreateFlag(ctx context.Context, name, key, description string, enabled bool, actorID int64) (*FeatureFlag, error) {
	f := &FeatureFlag{
		Name:        name,
		Key:         key,
		Description: description,
		Enabled:     enabled,
	}

	actx := database.WithAuditCtx(ctx, actorID, "CREATE_FEATURE_FLAG")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.CreateFlag(txCtx, f)
	})
	if err != nil {
		return nil, err
	}
	s.invalidateFlagsCache()
	return f, nil
}

func (s *service) UpdateFlag(ctx context.Context, id int64, name, key, description string, enabled bool, actorID int64) (*FeatureFlag, error) {
	var after *FeatureFlag

	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_FEATURE_FLAG")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		before, err := s.repo.GetFlagByID(txCtx, id)
		if err != nil {
			return err
		}
		if before == nil {
			return database.ErrNotFound
		}

		after = &FeatureFlag{
			ID:          id,
			Name:        name,
			Key:         key,
			Description: description,
			Enabled:     enabled,
		}

		return s.repo.UpdateFlag(txCtx, after)
	})
	if err != nil {
		return nil, err
	}
	s.invalidateFlagsCache()
	return after, nil
}

func (s *service) DeleteFlag(ctx context.Context, id int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_FEATURE_FLAG")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		before, err := s.repo.GetFlagByID(txCtx, id)
		if err != nil {
			return err
		}
		if before == nil {
			return database.ErrNotFound
		}
		return s.repo.DeleteFlag(txCtx, id)
	})
	if err != nil {
		return err
	}
	s.invalidateFlagsCache()
	return nil
}

func (s *service) ListFlags(ctx context.Context, limit int, cursor int64) ([]FeatureFlag, error) {
	if limit == 50 && cursor == 0 {
		if cached := s.getCachedFlags(); cached != nil {
			return cached, nil
		}
	}
	flags, err := s.repo.ListFlags(ctx, limit, cursor)
	if err != nil {
		return nil, err
	}
	if limit == 50 && cursor == 0 {
		s.setCachedFlags(flags)
	}
	return flags, nil
}

func (s *service) LookupFlag(ctx context.Context, key string) (bool, error) {
	if cached := s.getCachedFlags(); cached != nil {
		for _, f := range cached {
			if f.Key == key {
				return f.Enabled, nil
			}
		}
		return false, nil
	}
	f, err := s.repo.GetFlagByKey(ctx, key)
	if err != nil {
		return false, err
	}
	if f == nil {
		return false, nil
	}
	return f.Enabled, nil
}
