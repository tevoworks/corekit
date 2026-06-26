package permregistry

import "time"

// RegistryEntry is a system-wide known permission, registered by a domain service.
type RegistryEntry struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"` // e.g. "payment:create"
	Description string    `json:"description"`
	Domain      string    `json:"domain"` // e.g. "payment"
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateRegistryEntryRequest is the payload to register a new permission.
type CreateRegistryEntryRequest struct {
	Name        string `json:"name" validate:"required,nohtml"`
	Description string `json:"description" validate:"nohtml"`
	Domain      string `json:"domain" validate:"required"`
}

// UpdateRegistryEntryRequest is the payload to update a registry entry.
type UpdateRegistryEntryRequest struct {
	Description string `json:"description" validate:"nohtml"`
	Domain      string `json:"domain"`
	IsActive    bool   `json:"is_active"`
}

// GlobalTemplate is a system-level template managed by super-admins.
type GlobalTemplate struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	Category    string    `json:"category"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateGlobalTemplateRequest is the payload to create a global template.
type CreateGlobalTemplateRequest struct {
	Name        string   `json:"name" validate:"required,nohtml"`
	Description string   `json:"description" validate:"nohtml"`
	Permissions []string `json:"permissions" validate:"required"`
	Category    string   `json:"category" validate:"required"`
}

// UpdateGlobalTemplateRequest is the payload to update a global template.
type UpdateGlobalTemplateRequest struct {
	Name        string   `json:"name" validate:"required,nohtml"`
	Description string   `json:"description" validate:"nohtml"`
	Permissions []string `json:"permissions"`
	Category    string   `json:"category"`
	IsActive    bool     `json:"is_active"`
}

// ByDomain groups registry entries by their domain for display.
type ByDomain struct {
	Domain      string          `json:"domain"`
	Permissions []RegistryEntry `json:"permissions"`
}
