package apikey

import "time"

type APIKey struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	KeyPrefix     string     `json:"key_prefix"`
	KeyHash       string     `json:"-"`
	KeyLookupHash string     `json:"-"`
	CreatedBy     int64      `json:"created_by"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt     time.Time  `json:"expires_at"`
	RotatedAt     *time.Time `json:"rotated_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}
