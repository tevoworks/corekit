package settings

import "time"

type Setting struct {
	ID        int64     `json:"id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FeatureFlag struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Key         string    `json:"key"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
