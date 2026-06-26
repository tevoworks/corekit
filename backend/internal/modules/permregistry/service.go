package permregistry

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
	"gopkg.in/yaml.v3"
)

type Service interface {
	// Registry
	ListRegistry(ctx context.Context) ([]RegistryEntry, error)
	ListRegistryByDomain(ctx context.Context) ([]ByDomain, error)
	GetRegistryEntry(ctx context.Context, id int64) (*RegistryEntry, error)
	RegisterPermission(ctx context.Context, req CreateRegistryEntryRequest, actorID int64) (*RegistryEntry, error)
	UpdateRegistryEntry(ctx context.Context, id int64, req UpdateRegistryEntryRequest, actorID int64) (*RegistryEntry, error)
	DeleteRegistryEntry(ctx context.Context, id int64, actorID int64) error

	// Global templates
	ListGlobalTemplates(ctx context.Context) ([]GlobalTemplate, error)
	GetGlobalTemplate(ctx context.Context, id int64) (*GlobalTemplate, error)
	CreateGlobalTemplate(ctx context.Context, req CreateGlobalTemplateRequest, actorID int64) (*GlobalTemplate, error)
	UpdateGlobalTemplate(ctx context.Context, id int64, req UpdateGlobalTemplateRequest, actorID int64) (*GlobalTemplate, error)
	DeleteGlobalTemplate(ctx context.Context, id int64, actorID int64) error

	// Auto-discovery: load permissions.yaml and upsert all entries
	SyncFromYAML(ctx context.Context, yamlPath string) error

	// Export all registered permissions back to YAML format
	ExportToYAML(ctx context.Context) ([]byte, error)
}

type service struct {
	db           *sql.DB
	repo         Repository
	auditService audit.Service
}

func NewService(db *sql.DB, repo Repository, auditService audit.Service) Service {
	return &service{db: db, repo: repo, auditService: auditService}
}

// ── Registry ──────────────────────────────────────────────────────────────────

func (s *service) ListRegistry(ctx context.Context) ([]RegistryEntry, error) {
	return s.repo.ListRegistry(ctx)
}

func (s *service) ListRegistryByDomain(ctx context.Context) ([]ByDomain, error) {
	return s.repo.ListRegistryByDomain(ctx)
}

func (s *service) GetRegistryEntry(ctx context.Context, id int64) (*RegistryEntry, error) {
	e, err := s.repo.GetRegistryEntry(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, fmt.Errorf("registry entry not found")
	}
	return e, nil
}

func (s *service) RegisterPermission(ctx context.Context, req CreateRegistryEntryRequest, actorID int64) (*RegistryEntry, error) {
	if req.Name == "" || req.Domain == "" {
		return nil, fmt.Errorf("name and domain are required")
	}
	e := &RegistryEntry{
		Name:        req.Name,
		Description: req.Description,
		Domain:      req.Domain,
		IsActive:    true,
	}
	actx := database.WithAuditCtx(ctx, actorID, "REGISTER_PERMISSION")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.CreateRegistryEntry(txCtx, e)
	})
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (s *service) UpdateRegistryEntry(ctx context.Context, id int64, req UpdateRegistryEntryRequest, actorID int64) (*RegistryEntry, error) {
	e := &RegistryEntry{
		ID:          id,
		Description: req.Description,
		Domain:      req.Domain,
		IsActive:    req.IsActive,
	}
	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_REGISTRY_ENTRY")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.UpdateRegistryEntry(txCtx, e)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.GetRegistryEntry(ctx, id)
}

func (s *service) DeleteRegistryEntry(ctx context.Context, id int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_REGISTRY_ENTRY")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.DeleteRegistryEntry(txCtx, id)
	})
}

// ── Global Templates ──────────────────────────────────────────────────────────

func (s *service) ListGlobalTemplates(ctx context.Context) ([]GlobalTemplate, error) {
	return s.repo.ListGlobalTemplates(ctx)
}

func (s *service) GetGlobalTemplate(ctx context.Context, id int64) (*GlobalTemplate, error) {
	t, err := s.repo.GetGlobalTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("global template not found")
	}
	return t, nil
}

func (s *service) CreateGlobalTemplate(ctx context.Context, req CreateGlobalTemplateRequest, actorID int64) (*GlobalTemplate, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("template name is required")
	}
	if req.Permissions == nil {
		req.Permissions = []string{}
	}
	if req.Category == "" {
		req.Category = "portal_access"
	}
	t := &GlobalTemplate{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
		Category:    req.Category,
		IsActive:    true,
	}
	actx := database.WithAuditCtx(ctx, actorID, "CREATE_GLOBAL_TEMPLATE")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.CreateGlobalTemplate(txCtx, t)
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *service) UpdateGlobalTemplate(ctx context.Context, id int64, req UpdateGlobalTemplateRequest, actorID int64) (*GlobalTemplate, error) {
	if req.Permissions == nil {
		req.Permissions = []string{}
	}
	if req.Category == "" {
		req.Category = "portal_access"
	}
	t := &GlobalTemplate{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
		Category:    req.Category,
		IsActive:    req.IsActive,
	}
	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_GLOBAL_TEMPLATE")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.UpdateGlobalTemplate(txCtx, t)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.GetGlobalTemplate(ctx, id)
}

func (s *service) DeleteGlobalTemplate(ctx context.Context, id int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_GLOBAL_TEMPLATE")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.DeleteGlobalTemplate(txCtx, id)
	})
}

// ── Auto-discovery ────────────────────────────────────────────────────────────

type yamlSchema map[string][]struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func (s *service) SyncFromYAML(ctx context.Context, yamlPath string) error {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("read permissions.yaml: %w", err)
	}
	var schema yamlSchema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("parse permissions.yaml: %w", err)
	}
	count := 0
	for domain, perms := range schema {
		for _, p := range perms {
			if err := s.repo.UpsertRegistryEntry(ctx, p.Name, p.Description, domain); err != nil {
				slog.Warn("upsert registry entry failed", "name", p.Name, "error", err)
				continue
			}
			count++
		}
	}
	slog.Info("synced permissions from yaml", "count", count, "path", yamlPath)
	return nil
}

func (s *service) ExportToYAML(ctx context.Context) ([]byte, error) {
	entries, err := s.repo.ListRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("list registry: %w", err)
	}

	schema := make(yamlSchema)
	for _, e := range entries {
		if !e.IsActive {
			continue
		}
		schema[e.Domain] = append(schema[e.Domain], struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		}{
			Name:        e.Name,
			Description: e.Description,
		})
	}

	out, err := yaml.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshal yaml: %w", err)
	}
	return out, nil
}
