package iam

import (
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	maxFailedLoginAttempts = 10
	accountLockoutDuration = 15 * time.Minute
	absoluteSessionTTL    = 7 * 24 * time.Hour
)

var dummyHash = sync.OnceValue(func() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("timing-attack-mitigation"), bcrypt.DefaultCost)
	if err != nil {
		h, _ = bcrypt.GenerateFromPassword([]byte("fallback"), bcrypt.MinCost)
	}
	return h
})()

type User struct {
	ID                  int64      `json:"id"`
	Email               string     `json:"email"`
	PasswordHash        string     `json:"-"`
	FullName            string     `json:"full_name"`
	IsSuperAdmin        bool       `json:"is_super_admin"`
	RoleID              *int64     `json:"role_id"`
	RoleName            *string    `json:"role_name"`
	AvatarURL           *string    `json:"avatar_url"`
	Status              string     `json:"status"`
	FailedLoginAttempts int        `json:"-"`
	LockedUntil         *time.Time `json:"locked_until,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"`
}

func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

type UserDataExport struct {
	Profile       User             `json:"profile"`
	Sessions      []Session        `json:"sessions"`
	Notifications []Notification   `json:"notifications"`
	Preferences   []UserPreference `json:"preferences"`
}

type UserIdentity struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type UserVerification struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	TokenHash string    `json:"token_hash"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationPreference struct {
	UserID           int64     `json:"user_id"`
	NotificationType string    `json:"notification_type"`
	Channel          string    `json:"channel"`
	Enabled          bool      `json:"enabled"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Notification struct {
	ID        int64           `json:"id"`
	UserID    int64           `json:"user_id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `json:"data,omitempty"`
	IsRead    bool            `json:"is_read"`
	CreatedAt time.Time       `json:"created_at"`
}

type UserPreference struct {
	UserID    int64     `json:"user_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Session struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"user_id"`
	TokenID           string     `json:"token_id"`
	IPAddress         string     `json:"ip_address"`
	UserAgent         string     `json:"user_agent"`
	CreatedAt         time.Time  `json:"created_at"`
	ExpiresAt         time.Time  `json:"expires_at"`
	AbsoluteExpiresAt time.Time  `json:"absolute_expires_at"`
	RevokedAt         *time.Time `json:"revoked_at"`
	RevokedBy         *int64     `json:"revoked_by"`
}

func (s *Session) MaskIP() {
	if s == nil {
		return
	}
	s.IPAddress = maskIP(s.IPAddress)
}

func maskIP(ip string) string {
	if ip == "" {
		return ip
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "0.0.0.0"
	}
	if v4 := parsed.To4(); v4 != nil {
		return v4.String()[:strings.LastIndex(v4.String(), ".")+1] + "0"
	}
	// IPv6: mask last 64 bits, preserve /64 prefix
	v6 := parsed.To16()
	if v6 == nil {
		return "::"
	}
	for i := 8; i < 16; i++ {
		v6[i] = 0
	}
	return net.IP(v6).String()
}
