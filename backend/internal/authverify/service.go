// Package authverify implements the Hybrid Authentication Verification engine.
// It serves as the canonical source of truth for external services.
package authverify

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tevoworks/corekit/backend/internal/redisstore"
)

// ─── Action Type & Policy Decision (moved from policy.go) ──────────────────────

// ActionType classifies a request for the policy decision matrix.
type ActionType string

const (
	ActionREAD     ActionType = "READ"
	ActionWRITE    ActionType = "WRITE"
	ActionCRITICAL ActionType = "CRITICAL"
)

// criticalPathPrefixes are URL path prefixes that unconditionally classify as CRITICAL.
var criticalPathPrefixes = []string{
	"/api/auth/logout",
	"/api/auth/logout-all",
	"/api/auth/introspect",
	"/api/sessions",
	"/api/users/promote",
	"/api/impersonate",
}

// criticalPathSegments are path segments that classify any method as CRITICAL.
var criticalPathSegments = []string{
	"/rbac/",
	"/invite",
	"/roles",
	"/permissions",
	"/impersonate",
	"/promote",
	"/revoke",
	"/sessions",
	"/security",
}

// PolicyDecision is the output of the policy engine.
type PolicyDecision struct {
	ActionType               ActionType
	RequiresIntrospection    bool
	AllowCachedIntrospection bool
	Reason                   string
}

// ClassifyRequest determines the ActionType for a given HTTP method and URL path.
func ClassifyRequest(method, path string) ActionType {
	method = strings.ToUpper(method)
	lpath := strings.ToLower(path)

	isSafe := method == "GET" || method == "HEAD" || method == "OPTIONS"

	for _, prefix := range criticalPathPrefixes {
		if strings.HasPrefix(lpath, strings.ToLower(prefix)) {
			return ActionCRITICAL
		}
	}

	for _, seg := range criticalPathSegments {
		if strings.Contains(lpath, strings.ToLower(seg)) {
			return ActionCRITICAL
		}
	}

	if isSafe {
		return ActionREAD
	}

	return ActionWRITE
}

// Decide applies the hybrid decision matrix.
func Decide(redisEnabled bool, actionType ActionType) PolicyDecision {
	switch {
	case redisEnabled && actionType == ActionREAD:
		return PolicyDecision{
			ActionType:               ActionREAD,
			RequiresIntrospection:    false,
			AllowCachedIntrospection: false,
			Reason:                   "Mode A: READ — Redis revocation check sufficient",
		}

	case redisEnabled && actionType == ActionWRITE:
		return PolicyDecision{
			ActionType:               ActionWRITE,
			RequiresIntrospection:    false,
			AllowCachedIntrospection: false,
			Reason:                   "Mode A: WRITE — introspection optional (cache TTL 30s)",
		}

	case redisEnabled && actionType == ActionCRITICAL:
		return PolicyDecision{
			ActionType:               ActionCRITICAL,
			RequiresIntrospection:    true,
			AllowCachedIntrospection: false,
			Reason:                   "Mode A: CRITICAL — mandatory fresh introspection",
		}

	case !redisEnabled && actionType == ActionREAD:
		return PolicyDecision{
			ActionType:               ActionREAD,
			RequiresIntrospection:    false,
			AllowCachedIntrospection: true,
			Reason:                   "Mode B: READ — cached introspection acceptable (TTL 60s)",
		}

	case !redisEnabled && actionType == ActionWRITE:
		return PolicyDecision{
			ActionType:               ActionWRITE,
			RequiresIntrospection:    true,
			AllowCachedIntrospection: true,
			Reason:                   "Mode B: WRITE — mandatory introspection (cache TTL 30s)",
		}

	case !redisEnabled && actionType == ActionCRITICAL:
		return PolicyDecision{
			ActionType:               ActionCRITICAL,
			RequiresIntrospection:    true,
			AllowCachedIntrospection: false,
			Reason:                   "Mode B: CRITICAL — mandatory fresh introspection",
		}

	default:
		return PolicyDecision{
			ActionType:               actionType,
			RequiresIntrospection:    true,
			AllowCachedIntrospection: false,
			Reason:                   "unknown state — fail closed",
		}
	}
}

// ─── Public contract types ────────────────────────────────────────────────────

// IntrospectRequest is the input from an external service.
type IntrospectRequest struct {
	Token      string     `json:"token"`
	ActionType ActionType `json:"action_type"`          // READ | WRITE | CRITICAL
	Permission string     `json:"permission,omitempty"` // optional RBAC check
}

// IntrospectResponse is the stable external contract.
type IntrospectResponse struct {
	Active       bool      `json:"active"`
	SessionValid bool      `json:"session_valid"`
	UserID       int64     `json:"user_id,omitempty"`
	TokenID      string    `json:"token_id,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	IsSuperAdmin bool      `json:"is_super_admin"`
}

// inactiveResponse is the canonical fail-closed response.
var inactiveResponse = &IntrospectResponse{
	Active:       false,
	SessionValid: false,
	IsSuperAdmin: false,
}

// ─── DB query interface (kept minimal to avoid circular imports) ──────────────

// sessionRow is the subset of columns we need from the sessions table.
type sessionRow struct {
	UserID    int64
	TokenID   string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// userRow is the subset of columns we need from the users table.
type userRow struct {
	ID           int64
	Status       string
	IsSuperAdmin bool
	DeletedAt    *time.Time
}

// ─── Service ─────────────────────────────────────────────────────────────────

// Service is the introspection engine interface.
type Service interface {
	// Introspect validates a token and returns identity + session state.
	// It is always fail-closed: any error results in active=false.
	Introspect(ctx context.Context, req IntrospectRequest) *IntrospectResponse
}

type service struct {
	db        *sql.DB
	jwtSecret string
	cache     *IntrospectionCache
	revStore  *redisstore.RevocationStore
}

// NewService constructs an IntrospectionService. Both cache and revStore are
// required (use NewIntrospectionCache() and redisstore.NewRevocationStore("")).
func NewService(
	db *sql.DB,
	jwtSecret string,
	cache *IntrospectionCache,
	revStore *redisstore.RevocationStore,
) Service {
	return &service{
		db:        db,
		jwtSecret: jwtSecret,
		cache:     cache,
		revStore:  revStore,
	}
}

// Introspect validates the token through the full pipeline:
//
//  1. Parse + verify JWT signature
//  2. Extract token_id claim (session binding)
//  3. Redis fast-revocation check
//  4. Load session from DB → validate not revoked + not expired
//  5. Cache lookup (if allowed by policy)
//  6. Load user from DB → validate ACTIVE status
//  7. Build response, cache it (unless CRITICAL), return
func (s *service) Introspect(ctx context.Context, req IntrospectRequest) *IntrospectResponse {
	// ── 1. Parse + verify JWT ─────────────────────────────────────────────
	claims, err := s.parseJWT(req.Token)
	if err != nil {
		return inactiveResponse
	}

	tokenID, _ := claims["token_id"].(string)
	if tokenID == "" {
		return inactiveResponse
	}

	userIDFloat, _ := claims["user_id"].(float64)
	userID := int64(userIDFloat)
	if userID == 0 {
		return inactiveResponse
	}

	// ── 2. Normalise action type ──────────────────────────────────────────
	action := req.ActionType
	if action == "" {
		action = ActionREAD
	}

	// ── 3. Redis fast-revocation check (Mode A) ───────────────────────────
	if s.revStore.IsEnabled() {
		revoked, _ := s.revStore.IsRevoked(tokenID)
		if revoked {
			s.cache.Invalidate(tokenID)
			return inactiveResponse
		}
	}

	// ── 4. Decision + cache lookup (before DB) ──────────────────────────
	decision := Decide(s.revStore.IsEnabled(), action)
	if decision.AllowCachedIntrospection {
		if cached, ok := s.cache.Get(tokenID, action); ok {
			return cached
		}
	}

	// ── 5. DB: load session ───────────────────────────────────────────
	sess, err := s.getSession(ctx, tokenID)
	if err != nil || sess == nil {
		s.cache.Invalidate(tokenID)
		return inactiveResponse
	}

	if sess.RevokedAt != nil {
		s.cache.Invalidate(tokenID)
		return inactiveResponse
	}

	if time.Now().After(sess.ExpiresAt) {
		s.cache.Invalidate(tokenID)
		return inactiveResponse
	}

	// ── 6. DB: load + validate user ───────────────────────────────────────
	user, err := s.getUser(ctx, userID)
	if err != nil || user == nil {
		s.cache.Invalidate(tokenID)
		return inactiveResponse
	}

	if user.Status != "ACTIVE" || user.DeletedAt != nil {
		s.cache.Invalidate(tokenID)
		return inactiveResponse
	}

	// ── 7. Build + cache response ─────────────────────────────────────────
	resp := &IntrospectResponse{
		Active:       true,
		SessionValid: true,
		UserID:       userID,
		TokenID:      tokenID,
		ExpiresAt:    sess.ExpiresAt,
		IsSuperAdmin: user.IsSuperAdmin,
	}

	if decision.AllowCachedIntrospection {
		s.cache.Set(tokenID, action, resp)
	}

	return resp
}

// ─── Internal DB helpers ──────────────────────────────────────────────────────

func (s *service) parseJWT(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

func (s *service) getSession(ctx context.Context, tokenID string) (*sessionRow, error) {
	const q = `
		SELECT user_id, token_id, expires_at, revoked_at
		FROM sessions
		WHERE token_id = $1
		LIMIT 1`

	var row sessionRow
	err := s.db.QueryRowContext(ctx, q, tokenID).Scan(
		&row.UserID,
		&row.TokenID,
		&row.ExpiresAt,
		&row.RevokedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *service) getUser(ctx context.Context, userID int64) (*userRow, error) {
	const q = `SELECT id, status, is_super_admin, deleted_at FROM users WHERE id = $1 LIMIT 1`

	var row userRow
	err := s.db.QueryRowContext(ctx, q, userID).Scan(&row.ID, &row.Status, &row.IsSuperAdmin, &row.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
