package iam

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tevoworks/corekit/backend/internal/authverify"
	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
	"github.com/tevoworks/corekit/backend/internal/modules/queue"
	"github.com/tevoworks/corekit/backend/internal/redisstore"
	"github.com/tevoworks/corekit/backend/pkg/event"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, email, password, fullName string, isSuperAdmin bool) (*User, error)
	Login(ctx context.Context, email, password, ipAddress, userAgent string) (string, *User, error)
	GetProfile(ctx context.Context, id int64) (*User, error)
	UpdateProfile(ctx context.Context, id int64, email, fullName, avatarURL string, password, oldPassword *string, actorID int64) (*User, error)
	PromoteToSuperAdmin(ctx context.Context, email string) error
	ListGlobalUsers(ctx context.Context, limit int, cursor int64) ([]User, error)
	UpdateUserRole(ctx context.Context, userID int64, roleID *int64, actorID int64) error
	Impersonate(ctx context.Context, impersonatorID, targetUserID int64) (string, error)
	RevokeSession(ctx context.Context, tokenID string, revokedBy *int64) error
	RevokeAllSessions(ctx context.Context, userID int64, exceptTokenID string, revokedBy *int64) error
	ListSessions(ctx context.Context, userID int64) ([]Session, error)
	ListGlobalSessions(ctx context.Context, limit int, cursor int64) ([]Session, error)
	GetSessionByUserAndToken(ctx context.Context, userID int64, tokenID string) (*Session, error)

	// Identity
	VerifyEmail(ctx context.Context, token string) (*User, error)
	ResendVerification(ctx context.Context, email string) (*UserVerification, error)
	CreateOAuthUser(ctx context.Context, email, fullName, provider, providerUserID string) (*User, error)
	LinkOAuthIdentity(ctx context.Context, userID int64, provider, providerUserID string) error
	GetIdentityByProvider(ctx context.Context, provider, providerUserID string) (*UserIdentity, error)

	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateSession(ctx context.Context, s *Session) error
	GetNotificationPreferences(ctx context.Context, userID int64) ([]NotificationPreference, error)
	UpdateNotificationPreference(ctx context.Context, userID int64, notificationType string, enabled bool) error

	// Notifications
	GetNotifications(ctx context.Context, userID int64, limit int, cursor *time.Time, unreadOnly bool) ([]Notification, error)
	GetUnreadCount(ctx context.Context, userID int64) (int, error)
	MarkAsRead(ctx context.Context, userID, notifID int64) error
	MarkAllAsRead(ctx context.Context, userID int64) error
	DeleteNotification(ctx context.Context, userID, notifID int64) error
	CreateNotification(ctx context.Context, n *Notification) error

	// User Preferences
	UpsertPreference(ctx context.Context, userID int64, key, value string) error
	ListPreferences(ctx context.Context, userID int64) ([]UserPreference, error)

	// Admin Actions
	AdminCreateUser(ctx context.Context, email, fullName string, isSuperAdmin bool, actorID int64) (*User, error)
	AdminUpdateUser(ctx context.Context, targetUserID int64, email, fullName string, actorID int64) (*User, error)
	AdminDeleteUser(ctx context.Context, targetUserID int64, actorID int64) error
	AdminChangeUserStatus(ctx context.Context, targetUserID int64, status string, actorID int64) error
	ForceResetPassword(ctx context.Context, targetUserID int64, actorID int64) error

	// Data Export & Account Deletion
	ExportMyData(ctx context.Context, userID int64) (*UserDataExport, error)
	DeleteMyAccount(ctx context.Context, userID int64) error
	VerifyPassword(ctx context.Context, userID int64, password string) error

	// Linked OAuth Accounts
	GetLinkedAccounts(ctx context.Context, userID int64) ([]UserIdentity, error)
	UnlinkAccount(ctx context.Context, userID, identityID int64) error

	// Forgot / Reset Password
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error

	// Session Refresh
	RefreshSession(ctx context.Context, tokenString, ipAddress, userAgent string) (string, *User, error)
}

type service struct {
	db              *sql.DB
	repo            Repository
	jwtSecret       string
	auditService    audit.Service
	revStore        *redisstore.RevocationStore
	queueRepo       queue.Repository
	eventDispatcher *event.EventDispatcher
	cache           *authverify.IntrospectionCache
	frontendURL     string
}

func NewService(
	db *sql.DB,
	repo Repository,
	jwtSecret string,
	auditService audit.Service,
	revStore *redisstore.RevocationStore,
	queueRepo queue.Repository,
	eventDispatcher *event.EventDispatcher,
	cache *authverify.IntrospectionCache,
	frontendURL string,
) Service {
	return &service{
		db:              db,
		repo:            repo,
		jwtSecret:       jwtSecret,
		auditService:    auditService,
		revStore:        revStore,
		queueRepo:       queueRepo,
		eventDispatcher: eventDispatcher,
		cache:           cache,
		frontendURL:     frontendURL,
	}
}

func generateSecureToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(bytes)
	h := sha256.New()
	h.Write([]byte(token))
	tokenHash := hex.EncodeToString(h.Sum(nil))
	return token, tokenHash, nil
}

func (s *service) Register(ctx context.Context, email, password, fullName string, isSuperAdmin bool) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	u := &User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		FullName:     fullName,
		Status:       "ACTIVE",
	}

	actx := database.WithAuditAction(ctx, "REGISTER_USER")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		// Serialize first-user registration to prevent TOCTOU race
		// where two concurrent registrations both see count=0
		if _, err := database.GetQueryer(txCtx, s.db).ExecContext(txCtx, `SELECT pg_advisory_xact_lock(-7049161932602029030)`); err != nil {
			return fmt.Errorf("failed to acquire registration lock: %w", err)
		}
		var count int
		if err := database.GetQueryer(txCtx, s.db).QueryRowContext(txCtx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
			return fmt.Errorf("failed to check existing users: %w", err)
		}
		isFirst := count == 0
		u.IsSuperAdmin = isFirst

		if !isFirst {
			var customerRoleID *int64
			database.GetQueryer(txCtx, s.db).QueryRowContext(txCtx,
				`SELECT id FROM roles WHERE name = 'customer' LIMIT 1`).Scan(&customerRoleID)
			if customerRoleID != nil {
				u.RoleID = customerRoleID
			}
		}

		if err := s.repo.CreateUser(txCtx, u); err != nil {
			return err
		}

		if isFirst {
			return nil
		}

		token, tokenHash, err := generateSecureToken()
		if err != nil {
			return err
		}

		v := &UserVerification{
			UserID:    u.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		if err := s.repo.CreateVerificationToken(txCtx, v); err != nil {
			return err
		}

		emailPayload := EmailSendPayload{
			To:      u.Email,
			Subject: "Confirm Your Email - CoreKit",
			Body:    fmt.Sprintf("Please verify your email: %s/verify?token=%s", s.frontendURL, token),
		}
		payloadBytes, err := json.Marshal(emailPayload)
		if err != nil {
			return err
		}
		idempotencyKey := fmt.Sprintf("verify_email_user_%d_token_%s", u.ID, tokenHash)
		if s.queueRepo != nil {
			if err := s.queueRepo.Enqueue(txCtx, database.GetTx(txCtx), queue.JobTypeEmailSend, payloadBytes, &idempotencyKey); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if s.eventDispatcher != nil {
		if err := s.eventDispatcher.Dispatch(ctx, nil, "user.registered", map[string]any{
			"user_id":    u.ID,
			"email_hash": sha256Hex(u.Email),
		}); err != nil {
			slog.Error("failed to dispatch event", "event", "user.registered", "error", err)
		}
	}

	return u, nil
}

func (s *service) Login(ctx context.Context, email, password, ipAddress, userAgent string) (string, *User, error) {
	var u *User
	var err error

	u, err = s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", nil, err
	}

	passwordHash := ""
	userID := int64(0)

	if u != nil && u.DeletedAt == nil {
		passwordHash = u.PasswordHash
		userID = u.ID
	} else {
		passwordHash = string(dummyHash)
	}

	passwordMatch := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil

	if userID == 0 || !passwordMatch {
		if userID > 0 && u.Status == "ACTIVE" {
			_ = database.RunInTransaction(ctx, s.db, func(txCtx context.Context) error {
				var currentAttempts int
				err := database.GetQueryer(txCtx, s.db).QueryRowContext(txCtx,
					`UPDATE users SET failed_login_attempts = failed_login_attempts + 1 WHERE id = $1 RETURNING failed_login_attempts`, userID).
					Scan(&currentAttempts)
				if err != nil {
					return nil
				}
				if currentAttempts >= maxFailedLoginAttempts {
					s.repo.LockAccount(txCtx, userID, time.Now().Add(accountLockoutDuration))
				}
				return nil
			})
		}
		if s.eventDispatcher != nil {
			if err := s.eventDispatcher.Dispatch(ctx, nil, "user.login.failed", map[string]any{
				"email_hash": sha256Hex(email),
				"error":      "invalid email or password",
			}); err != nil {
				slog.Error("failed to dispatch event", "event", "user.login.failed", "error", err)
			}
		}
		return "", nil, errors.New("invalid email or password")
	}

	if u.Status != "ACTIVE" {
		return "", nil, errors.New("invalid email or password")
	}

	tokenIDBytes := make([]byte, 32)
	if _, err := rand.Read(tokenIDBytes); err != nil {
		return "", nil, fmt.Errorf("generate token id: %w", err)
	}
	tokenID := hex.EncodeToString(tokenIDBytes)

	role := ""
	if u.RoleID != nil && u.RoleName != nil {
		role = *u.RoleName
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":        u.ID,
		"role":           role,
		"is_super_admin": u.IsSuperAdmin,
		"token_id":       tokenID,
		"iat":            time.Now().Unix(),
		"jti":            tokenID,
		"exp":            time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", nil, err
	}

	sess := &Session{
		UserID:            u.ID,
		TokenID:           tokenID,
		IPAddress:         ipAddress,
		UserAgent:         userAgent,
		ExpiresAt:         time.Now().Add(24 * time.Hour),
		AbsoluteExpiresAt: time.Now().Add(absoluteSessionTTL),
	}

	actx := database.WithAuditCtx(ctx, u.ID, "SESSION_LOGIN")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		if err := s.repo.CreateSession(txCtx, sess); err != nil {
			return err
		}
		if _, err := database.GetQueryer(txCtx, s.db).ExecContext(txCtx,
			`UPDATE users SET failed_login_attempts = 0, locked_until = NULL, last_login_at = CURRENT_TIMESTAMP WHERE id = $1`, u.ID); err != nil {
			slog.Error("failed to reset login attempts after successful login", "user_id", u.ID, "error", err)
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}

	if s.eventDispatcher != nil {
		if err := s.eventDispatcher.Dispatch(ctx, nil, "user.login.success", map[string]any{
			"user_id":    u.ID,
			"email_hash": sha256Hex(u.Email),
		}); err != nil {
			slog.Error("failed to dispatch event", "event", "user.login.success", "error", err)
		}
	}

	return tokenString, u, nil
}

func (s *service) GetProfile(ctx context.Context, id int64) (*User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *service) PromoteToSuperAdmin(ctx context.Context, email string) error {
	actx := database.WithAuditAction(ctx, "PROMOTE_SUPERADMIN")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.UpdateSuperAdminStatus(txCtx, email, true)
	})
}

func (s *service) ListGlobalUsers(ctx context.Context, limit int, cursor int64) ([]User, error) {
	return s.repo.ListGlobal(ctx, limit, cursor)
}

func (s *service) UpdateUserRole(ctx context.Context, userID int64, roleID *int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_USER_ROLE")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.UpdateUserRole(txCtx, userID, roleID)
	})
}

func (s *service) Impersonate(ctx context.Context, impersonatorID, targetUserID int64) (string, error) {
	var tokenString string

	actx := database.WithAuditCtx(ctx, impersonatorID, "IMPERSONATE")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		target, err := s.repo.GetUserByID(txCtx, targetUserID)
		if err != nil || target == nil {
			return errors.New("target user not found")
		}
		if target.Status != "ACTIVE" || target.DeletedAt != nil {
			return errors.New("cannot impersonate: target user is not active")
		}

		tokenIDBytes := make([]byte, 32)
		if _, err := rand.Read(tokenIDBytes); err != nil {
			return fmt.Errorf("generate token id: %w", err)
		}
		tokenID := hex.EncodeToString(tokenIDBytes)

		role := ""
		if target.RoleID != nil && target.RoleName != nil {
			role = *target.RoleName
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":         targetUserID,
			"role":            role,
			"is_super_admin":  target.IsSuperAdmin,
			"token_id":        tokenID,
			"impersonator_id": impersonatorID,
			"iat":             time.Now().Unix(),
			"jti":             tokenID,
			"exp":             time.Now().Add(2 * time.Hour).Unix(),
		})

		tokenString, err = token.SignedString([]byte(s.jwtSecret))
		if err != nil {
			return err
		}

		sess := &Session{
			UserID:            targetUserID,
			TokenID:           tokenID,
			ExpiresAt:         time.Now().Add(2 * time.Hour),
			AbsoluteExpiresAt: time.Now().Add(absoluteSessionTTL),
		}
		return s.repo.CreateSession(txCtx, sess)
	})

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *service) RefreshSession(ctx context.Context, tokenString, ipAddress, userAgent string) (string, *User, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", nil, fmt.Errorf("invalid token: %w", err)
	}

	userID, ok := claims["user_id"].(float64)
	if !ok || userID == 0 {
		return "", nil, errors.New("invalid token claims")
	}
	tokenID, ok := claims["token_id"].(string)
	if !ok || tokenID == "" {
		return "", nil, errors.New("invalid token claims")
	}
	role, _ := claims["role"].(string)
	isSuperAdmin, _ := claims["is_super_admin"].(bool)

	var impersonatorID *int64
	if imp, ok := claims["impersonator_id"].(float64); ok && imp > 0 {
		id := int64(imp)
		impersonatorID = &id
	}

	uid := int64(userID)

	// Check session validity
	sess, err := s.repo.GetSessionByTokenID(ctx, tokenID)
	if err != nil {
		return "", nil, fmt.Errorf("session lookup: %w", err)
	}
	if sess == nil {
		return "", nil, errors.New("session not found")
	}
	if sess.UserID != uid {
		return "", nil, errors.New("session user mismatch")
	}
	if sess.RevokedAt != nil {
		return "", nil, errors.New("session has been revoked")
	}
	if time.Now().After(sess.ExpiresAt.Add(1 * time.Hour)) {
		return "", nil, errors.New("session expired too long ago")
	}
	if time.Now().After(sess.AbsoluteExpiresAt) {
		return "", nil, errors.New("session has exceeded its absolute lifetime")
	}

	// Check user status
	user, err := s.repo.GetUserByID(ctx, uid)
	if err != nil {
		return "", nil, fmt.Errorf("user lookup: %w", err)
	}
	if user == nil {
		return "", nil, errors.New("user not found")
	}
	if user.Status != "ACTIVE" && user.Status != "FORCE_PASSWORD_RESET" {
		return "", nil, errors.New("account is not active")
	}
	if user.DeletedAt != nil {
		return "", nil, errors.New("account has been deleted")
	}
	if user.IsLocked() {
		return "", nil, errors.New("account is temporarily locked")
	}

	// Generate new token ID and session
	newTokenIDBytes := make([]byte, 32)
	if _, err := rand.Read(newTokenIDBytes); err != nil {
		return "", nil, fmt.Errorf("generate token id: %w", err)
	}
	newTokenID := hex.EncodeToString(newTokenIDBytes)

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":        uid,
		"role":           role,
		"is_super_admin": isSuperAdmin,
		"token_id":       newTokenID,
		"iat":            time.Now().Unix(),
		"jti":            newTokenID,
		"exp":            time.Now().Add(24 * time.Hour).Unix(),
	})
	if impersonatorID != nil {
		if claims, ok := newToken.Claims.(jwt.MapClaims); ok {
			claims["impersonator_id"] = *impersonatorID
		}
	}

	newTokenString, err := newToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", nil, fmt.Errorf("sign token: %w", err)
	}

	newSess := &Session{
		UserID:            uid,
		TokenID:           newTokenID,
		IPAddress:         ipAddress,
		UserAgent:         userAgent,
		ExpiresAt:         time.Now().Add(24 * time.Hour),
		AbsoluteExpiresAt: sess.AbsoluteExpiresAt,
	}

	actx := database.WithAuditCtx(ctx, uid, "SESSION_REFRESH")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		if err := s.repo.RevokeSession(txCtx, tokenID, &uid); err != nil {
			return err
		}
		return s.repo.CreateSession(txCtx, newSess)
	})
	if err != nil {
		return "", nil, fmt.Errorf("session refresh: %w", err)
	}

	if s.revStore != nil && s.revStore.IsEnabled() {
		s.revStore.MarkRevoked(tokenID, sess.ExpiresAt)
	}
	if s.cache != nil {
		s.cache.Invalidate(tokenID)
	}

	return newTokenString, user, nil
}

func (s *service) RevokeSession(ctx context.Context, tokenID string, revokedBy *int64) error {
	sess, err := s.repo.GetSessionByTokenID(ctx, tokenID)
	if err != nil {
		return err
	}
	if sess == nil {
		return errors.New("session not found")
	}

	actx := ctx
	if revokedBy != nil {
		actx = database.WithAuditCtx(ctx, *revokedBy, "SESSION_REVOKE")
	} else {
		actx = database.WithAuditAction(ctx, "SESSION_REVOKE")
	}
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.RevokeSession(txCtx, tokenID, revokedBy)
	})
	if err != nil {
		return err
	}

	if s.revStore != nil && s.revStore.IsEnabled() {
		s.revStore.MarkRevoked(tokenID, sess.ExpiresAt)
	}
	if s.cache != nil {
		s.cache.Invalidate(tokenID)
	}

	return nil
}

func (s *service) RevokeAllSessions(ctx context.Context, userID int64, exceptTokenID string, revokedBy *int64) error {
	var revokedTokens []struct {
		TokenID   string
		ExpiresAt time.Time
	}

	actx := ctx
	if revokedBy != nil {
		actx = database.WithAuditCtx(ctx, *revokedBy, "SESSION_REVOKE_ALL")
	} else {
		actx = database.WithAuditAction(ctx, "SESSION_REVOKE_ALL")
	}
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		rows, err := database.GetQueryer(txCtx, s.db).QueryContext(txCtx,
			`SELECT token_id, expires_at FROM sessions WHERE user_id = $1 AND revoked_at IS NULL AND token_id != $2`,
			userID, exceptTokenID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var t struct {
				TokenID   string
				ExpiresAt time.Time
			}
			if err := rows.Scan(&t.TokenID, &t.ExpiresAt); err != nil {
				return err
			}
			revokedTokens = append(revokedTokens, t)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return s.repo.RevokeAllSessions(txCtx, userID, exceptTokenID, revokedBy)
	})
	if err != nil {
		return err
	}

	s.invalidateTokens(revokedTokens)

	return nil
}

func (s *service) invalidateTokens(tokens []struct {
	TokenID   string
	ExpiresAt time.Time
}) {
	for _, t := range tokens {
		if s.revStore != nil && s.revStore.IsEnabled() {
			s.revStore.MarkRevoked(t.TokenID, t.ExpiresAt)
		}
		if s.cache != nil {
			s.cache.Invalidate(t.TokenID)
		}
	}
}

func (s *service) ListSessions(ctx context.Context, userID int64) ([]Session, error) {
	return s.repo.ListSessions(ctx, userID)
}

func (s *service) ListGlobalSessions(ctx context.Context, limit int, cursor int64) ([]Session, error) {
	return s.repo.ListGlobalSessions(ctx, limit, cursor)
}

func (s *service) GetSessionByUserAndToken(ctx context.Context, userID int64, tokenID string) (*Session, error) {
	return s.repo.GetSessionByUserAndToken(ctx, userID, tokenID)
}

func (s *service) UpdateProfile(ctx context.Context, id int64, email, fullName, avatarURL string, password, oldPassword *string, actorID int64) (*User, error) {
	var u *User
	var emailChanged bool
	var newEmail string
	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_USER_PROFILE")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		var err error
		u, err = s.repo.GetUserByID(txCtx, id)
		if err != nil {
			return err
		}
		if u == nil {
			return errors.New("user not found")
		}

		if password != nil && *password != "" {
			if u.PasswordHash != "" && u.Status != "FORCE_PASSWORD_RESET" {
				if oldPassword == nil || *oldPassword == "" {
					return errors.New("current password is required to set a new password")
				}
				if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(*oldPassword)); err != nil {
					return errors.New("current password is incorrect")
				}
			}
		}

		if email != "" && email != u.Email {
			existing, err := s.repo.GetUserByEmail(txCtx, email)
			if err != nil {
				return err
			}
			if existing != nil && existing.ID != id {
				return errors.New("email already taken")
			}
			emailChanged = true
			newEmail = email
			u.Email = email
		}

		if fullName != "" {
			u.FullName = fullName
		}

		if avatarURL != "" {
			u.AvatarURL = &avatarURL
		}

		if password != nil && *password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(*password), 12)
			if err != nil {
				return err
			}
			u.PasswordHash = string(hash)
			if u.Status == "FORCE_PASSWORD_RESET" || u.Status == "PENDING_VERIFICATION" {
				u.Status = "ACTIVE"
			}
		}

		err = s.repo.UpdateUser(txCtx, u)
		if err != nil {
			return err
		}

		if emailChanged && s.queueRepo != nil {
			token, tokenHash, err := generateSecureToken()
			if err != nil {
				return err
			}
			v := &UserVerification{
				UserID:    u.ID,
				TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}
			if err := s.repo.CreateVerificationToken(txCtx, v); err != nil {
				return err
			}
			emailPayload := EmailSendPayload{
				To:      newEmail,
				Subject: "Confirm Your Email Change - CoreKit",
				Body:    fmt.Sprintf("Please verify your new email: %s/verify?token=%s", s.frontendURL, token),
			}
			payloadBytes, err := json.Marshal(emailPayload)
			if err != nil {
				return err
			}
			idempotencyKey := fmt.Sprintf("email_change_user_%d_token_%s", u.ID, tokenHash)
			if enqErr := s.queueRepo.Enqueue(txCtx, database.GetTx(txCtx), queue.JobTypeEmailSend, payloadBytes, &idempotencyKey); enqErr != nil {
				slog.Error("failed to enqueue email change verification", "error", enqErr)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	if emailChanged && s.eventDispatcher != nil {
		if err := s.eventDispatcher.Dispatch(ctx, nil, "user.email.changed", map[string]any{
			"user_id":    u.ID,
			"email_hash": sha256Hex(u.Email),
		}); err != nil {
			slog.Error("failed to dispatch event", "event", "user.email.changed", "error", err)
		}
	}
	return u, nil
}

func (s *service) VerifyEmail(ctx context.Context, token string) (*User, error) {
	h := sha256.New()
	h.Write([]byte(token))
	tokenHash := hex.EncodeToString(h.Sum(nil))

	var v *UserVerification
	var u *User
	actx := database.WithAuditAction(ctx, "VERIFY_EMAIL_SUCCESS")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		var err error
		v, err = s.repo.GetVerificationByTokenHash(txCtx, tokenHash)
		if err != nil {
			return err
		}
		if v == nil {
			return errors.New("invalid or expired verification token")
		}

		if time.Now().After(v.ExpiresAt) {
			_ = s.repo.DeleteVerificationToken(txCtx, v.ID)
			return errors.New("verification token has expired")
		}

		u, err = s.repo.GetUserByID(txCtx, v.UserID)
		if err != nil || u == nil {
			return errors.New("user not found")
		}

		err = s.repo.UpdateUserStatus(txCtx, v.UserID, "ACTIVE")
		if err != nil {
			return err
		}
		u.Status = "ACTIVE"

		err = s.repo.DeleteVerificationToken(txCtx, v.ID)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if s.eventDispatcher != nil {
		if err := s.eventDispatcher.Dispatch(ctx, nil, "user.verified", map[string]any{
			"user_id": v.UserID,
		}); err != nil {
			slog.Error("failed to dispatch event", "event", "user.verified", "error", err)
		}
	}

	return u, nil
}

func (s *service) ResendVerification(ctx context.Context, email string) (*UserVerification, error) {
	u, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}

	if u.Status == "ACTIVE" {
		return nil, errors.New("user is already verified and active")
	}

	token, tokenHash, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	v := &UserVerification{
		UserID:    u.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	actx := database.WithAuditCtx(ctx, u.ID, "RESEND_VERIFICATION")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		q := database.GetQueryer(txCtx, s.db)
		if _, err := q.ExecContext(txCtx, "DELETE FROM user_verifications WHERE user_id = $1", u.ID); err != nil {
			return fmt.Errorf("delete existing verifications: %w", err)
		}

		err := s.repo.CreateVerificationToken(txCtx, v)
		if err != nil {
			return err
		}

		emailPayload := EmailSendPayload{
			To:      u.Email,
			Subject: "Confirm Your Email - CoreKit",
			Body:    fmt.Sprintf("Please verify your email: %s/verify?token=%s", s.frontendURL, token),
		}
		payloadBytes, err := json.Marshal(emailPayload)
		if err != nil {
			return err
		}
		idempotencyKey := fmt.Sprintf("verify_email_user_%d_token_%s", u.ID, tokenHash)
		if s.queueRepo != nil {
			err = s.queueRepo.Enqueue(txCtx, database.GetTx(txCtx), queue.JobTypeEmailSend, payloadBytes, &idempotencyKey)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return v, nil
}

func (s *service) CreateOAuthUser(ctx context.Context, email, fullName, provider, providerUserID string) (*User, error) {
	u := &User{
		Email:        email,
		PasswordHash: "",
		FullName:     fullName,
		Status:       "ACTIVE",
	}

	actx := database.WithAuditAction(ctx, "CREATE_OAUTH_USER")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		var count int
		if err := database.GetQueryer(txCtx, s.db).QueryRowContext(txCtx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
			return fmt.Errorf("failed to check existing users: %w", err)
		}
		isFirst := count == 0
		u.IsSuperAdmin = isFirst

		if !isFirst {
			var viewerRoleID *int64
			if err := database.GetQueryer(txCtx, s.db).QueryRowContext(txCtx,
				`SELECT id FROM roles WHERE name = 'viewer' LIMIT 1`).Scan(&viewerRoleID); err == nil && viewerRoleID != nil {
				u.RoleID = viewerRoleID
			}
		}

		if err := s.repo.CreateUser(txCtx, u); err != nil {
			return err
		}

		ident := &UserIdentity{
			UserID:         u.ID,
			Provider:       provider,
			ProviderUserID: providerUserID,
		}
		return s.repo.CreateIdentity(txCtx, ident)
	})

	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s *service) LinkOAuthIdentity(ctx context.Context, userID int64, provider, providerUserID string) error {
	ident := &UserIdentity{
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	}

	actx := database.WithAuditCtx(ctx, userID, "LINK_OAUTH_IDENTITY")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.CreateIdentity(txCtx, ident)
	})
}

func (s *service) GetIdentityByProvider(ctx context.Context, provider, providerUserID string) (*UserIdentity, error) {
	return s.repo.GetIdentityByProvider(ctx, provider, providerUserID)
}

func (s *service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.repo.GetUserByEmail(ctx, email)
}

func (s *service) CreateSession(ctx context.Context, sess *Session) error {
	return s.repo.CreateSession(ctx, sess)
}

func (s *service) GetNotificationPreferences(ctx context.Context, userID int64) ([]NotificationPreference, error) {
	return s.repo.GetNotificationPreferences(ctx, userID)
}

func (s *service) UpdateNotificationPreference(ctx context.Context, userID int64, notificationType string, enabled bool) error {
	return s.repo.UpsertNotificationPreference(ctx, userID, notificationType, enabled)
}

func (s *service) ExportMyData(ctx context.Context, userID int64) (*UserDataExport, error) {
	profile, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, errors.New("user not found")
	}

	sessions, err := s.repo.ListSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	notifications, err := s.repo.ListNotifications(ctx, userID, 1000, nil, false)
	if err != nil {
		return nil, err
	}

	prefs, err := s.repo.ListPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserDataExport{
		Profile:       *profile,
		Sessions:      sessions,
		Notifications: notifications,
		Preferences:   prefs,
	}, nil
}

func (s *service) DeleteMyAccount(ctx context.Context, userID int64) error {
	actx := database.WithAuditCtx(ctx, userID, "DELETE_ACCOUNT")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		u, err := s.repo.GetUserByID(txCtx, userID)
		if err != nil {
			return err
		}
		email := ""
		if u != nil {
			email = u.Email
		}

		sessions, err := s.repo.ListSessions(txCtx, userID)
		if err != nil {
			return err
		}
		for _, sess := range sessions {
			if sess.RevokedAt == nil {
				if err := s.repo.RevokeSession(txCtx, sess.TokenID, &userID); err != nil {
					return err
				}
				if s.revStore != nil {
					s.revStore.MarkRevoked(sess.TokenID, sess.ExpiresAt)
				}
				if s.cache != nil {
					s.cache.Invalidate(sess.TokenID)
				}
			}
		}

		if err := s.repo.SoftDeleteUser(txCtx, userID); err != nil {
			return err
		}

		if delNotifErr := s.repo.DeleteAllNotifications(txCtx, userID); delNotifErr != nil {
			slog.Error("failed to delete notifications on account deletion", "error", delNotifErr)
		}
		if delPrefErr := s.repo.DeleteAllPreferences(txCtx, userID); delPrefErr != nil {
			slog.Error("failed to delete preferences on account deletion", "error", delPrefErr)
		}
		if delIdentErr := s.repo.DeleteIdentitiesByUserID(txCtx, userID); delIdentErr != nil {
			slog.Error("failed to delete identities on account deletion", "error", delIdentErr)
		}

		if err := s.auditService.AnonymizeAuditPII(txCtx, userID, email); err != nil {
			return err
		}

		return nil
	})
}

func (s *service) VerifyPassword(ctx context.Context, userID int64, password string) error {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("user not found")
	}
	if u.PasswordHash == "" {
		return errors.New("password verification not available; re-authenticate via OAuth")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return errors.New("current password is incorrect")
	}
	return nil
}

func (s *service) GetLinkedAccounts(ctx context.Context, userID int64) ([]UserIdentity, error) {
	return s.repo.ListIdentitiesByUserID(ctx, userID)
}

func (s *service) UnlinkAccount(ctx context.Context, userID, identityID int64) error {
	hasPwd, err := s.repo.HasPassword(ctx, userID)
	if err != nil {
		return err
	}
	identities, err := s.repo.ListIdentitiesByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if !hasPwd && len(identities) <= 1 {
		return errors.New("you must set a password before disconnecting your last authentication method")
	}
	return s.repo.DeleteIdentity(ctx, identityID)
}

func (s *service) GetNotifications(ctx context.Context, userID int64, limit int, cursor *time.Time, unreadOnly bool) ([]Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListNotifications(ctx, userID, limit, cursor, unreadOnly)
}

func (s *service) GetUnreadCount(ctx context.Context, userID int64) (int, error) {
	return s.repo.GetUnreadCount(ctx, userID)
}

func (s *service) MarkAsRead(ctx context.Context, userID, notifID int64) error {
	return s.repo.MarkAsRead(ctx, userID, notifID)
}

func (s *service) MarkAllAsRead(ctx context.Context, userID int64) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}

func (s *service) DeleteNotification(ctx context.Context, userID, notifID int64) error {
	return s.repo.DeleteNotification(ctx, userID, notifID)
}

func (s *service) CreateNotification(ctx context.Context, n *Notification) error {
	return s.repo.CreateNotification(ctx, n)
}

func (s *service) UpsertPreference(ctx context.Context, userID int64, key, value string) error {
	return s.repo.UpsertPreference(ctx, userID, key, value)
}

func (s *service) ListPreferences(ctx context.Context, userID int64) ([]UserPreference, error) {
	return s.repo.ListPreferences(ctx, userID)
}

func (s *service) AdminCreateUser(ctx context.Context, email, fullName string, isSuperAdmin bool, actorID int64) (*User, error) {
	u := &User{
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: "",
		FullName:     fullName,
		IsSuperAdmin: isSuperAdmin,
		Status:       "PENDING_VERIFICATION",
	}

	actx := database.WithAuditCtx(ctx, actorID, "ADMIN_CREATE_USER")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		if !isSuperAdmin {
			var viewerRoleID *int64
			if err := database.GetQueryer(txCtx, s.db).QueryRowContext(txCtx,
				`SELECT id FROM roles WHERE name = 'viewer' LIMIT 1`).Scan(&viewerRoleID); err == nil {
				u.RoleID = viewerRoleID
			}
		}

		if err := s.repo.CreateUser(txCtx, u); err != nil {
			return err
		}

		token, tokenHash, err := generateSecureToken()
		if err != nil {
			return err
		}

		v := &UserVerification{
			UserID:    u.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err = s.repo.CreateVerificationToken(txCtx, v)
		if err != nil {
			return err
		}

		emailPayload := EmailSendPayload{
			To:      u.Email,
			Subject: "Invitation to CoreKit",
			Body:    fmt.Sprintf("You have been invited to CoreKit. Click the link to complete your registration and set your password: %s/verify?token=%s&invite=true", s.frontendURL, token),
		}
		payloadBytes, err := json.Marshal(emailPayload)
		if err != nil {
			return err
		}
		idempotencyKey := fmt.Sprintf("invite_user_%d_token_%s", u.ID, tokenHash)
		if s.queueRepo != nil {
			if enqErr := s.queueRepo.Enqueue(txCtx, database.GetTx(txCtx), queue.JobTypeEmailSend, payloadBytes, &idempotencyKey); enqErr != nil {
				slog.Error("failed to enqueue invitation email", "error", enqErr)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s *service) AdminUpdateUser(ctx context.Context, targetUserID int64, email, fullName string, actorID int64) (*User, error) {
	var u *User
	actx := database.WithAuditCtx(ctx, actorID, "ADMIN_UPDATE_USER")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		var err error
		u, err = s.repo.GetUserByID(txCtx, targetUserID)
		if err != nil || u == nil {
			return errors.New("user not found")
		}

		newEmail := strings.ToLower(strings.TrimSpace(email))
		if newEmail != u.Email {
			existing, err := s.repo.GetUserByEmail(txCtx, newEmail)
			if err != nil {
				return err
			}
			if existing != nil {
				return errors.New("email already taken")
			}
		}

		u.Email = newEmail
		u.FullName = fullName
		u.UpdatedAt = time.Now()

		return s.repo.UpdateUser(txCtx, u)
	})

	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *service) AdminChangeUserStatus(ctx context.Context, targetUserID int64, status string, actorID int64) error {
	if targetUserID == actorID {
		return errors.New("cannot change your own status")
	}
	actx := database.WithAuditCtx(ctx, actorID, "ADMIN_CHANGE_USER_STATUS")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		u, err := s.repo.GetUserByID(txCtx, targetUserID)
		if err != nil || u == nil {
			return errors.New("user not found")
		}

		if u.DeletedAt != nil {
			return errors.New("cannot change status of a deleted user")
		}

		validTransitions := map[string][]string{
			"ACTIVE":               {"SUSPENDED", "HALTED"},
			"SUSPENDED":            {"ACTIVE"},
			"HALTED":               {"ACTIVE"},
			"LOCKED":               {"ACTIVE"},
			"PENDING_VERIFICATION": {"ACTIVE", "SUSPENDED", "HALTED"},
		}

		allowed, ok := validTransitions[u.Status]
		if !ok {
			return fmt.Errorf("cannot change status from current state: %s", u.Status)
		}
		valid := false
		for _, s := range allowed {
			if s == status {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid status transition from %s to %s", u.Status, status)
		}

		err = s.repo.UpdateUserStatus(txCtx, targetUserID, status)
		if err != nil {
			return err
		}

		if status != "ACTIVE" {
			if revErr := s.repo.RevokeAllSessions(txCtx, targetUserID, "", &actorID); revErr != nil {
				slog.Error("failed to revoke sessions on status change", "error", revErr)
			}
		}

		u.Status = status
		return nil
	})
}

func (s *service) AdminDeleteUser(ctx context.Context, targetUserID int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "ADMIN_DELETE_USER")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		u, err := s.repo.GetUserByID(txCtx, targetUserID)
		if err != nil {
			return err
		}
		if u == nil || u.DeletedAt != nil {
			return errors.New("user not found")
		}
		if u.IsSuperAdmin {
			return errors.New("cannot delete a super admin user")
		}
		email := u.Email

		sessions, err := s.repo.ListSessions(txCtx, targetUserID)
		if err != nil {
			return err
		}
		for _, sess := range sessions {
			if sess.RevokedAt == nil {
				if err := s.repo.RevokeSession(txCtx, sess.TokenID, &actorID); err != nil {
					return err
				}
				if s.revStore != nil {
					s.revStore.MarkRevoked(sess.TokenID, sess.ExpiresAt)
				}
				if s.cache != nil {
					s.cache.Invalidate(sess.TokenID)
				}
			}
		}

		if err := s.repo.SoftDeleteUser(txCtx, targetUserID); err != nil {
			return err
		}

		if delNotifErr := s.repo.DeleteAllNotifications(txCtx, targetUserID); delNotifErr != nil {
			slog.Error("failed to delete notifications on admin delete", "error", delNotifErr)
		}
		if delPrefErr := s.repo.DeleteAllPreferences(txCtx, targetUserID); delPrefErr != nil {
			slog.Error("failed to delete preferences on admin delete", "error", delPrefErr)
		}
		if delIdentErr := s.repo.DeleteIdentitiesByUserID(txCtx, targetUserID); delIdentErr != nil {
			slog.Error("failed to delete identities on admin delete", "error", delIdentErr)
		}

		if err := s.auditService.AnonymizeAuditPII(txCtx, targetUserID, email); err != nil {
			return err
		}

		return nil
	})
}

func (s *service) ForceResetPassword(ctx context.Context, targetUserID int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "ADMIN_FORCE_RESET_PASSWORD")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		u, err := s.repo.GetUserByID(txCtx, targetUserID)
		if err != nil || u == nil {
			return errors.New("user not found")
		}

		err = s.repo.UpdateUserStatus(txCtx, targetUserID, "FORCE_PASSWORD_RESET")
		if err != nil {
			return err
		}

		if revErr := s.repo.RevokeAllSessions(txCtx, targetUserID, "", &actorID); revErr != nil {
			slog.Error("failed to revoke sessions on force reset", "error", revErr)
		}

		token, tokenHash, err := generateSecureToken()
		if err != nil {
			return err
		}

		v := &UserVerification{
			UserID:    u.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err = s.repo.CreateVerificationToken(txCtx, v)
		if err != nil {
			return err
		}

		emailPayload := EmailSendPayload{
			To:      u.Email,
			Subject: "Reset Your Password - CoreKit",
			Body:    fmt.Sprintf("Your password has been reset by an administrator. Please click this link to set a new password: %s/verify?token=%s&invite=true", s.frontendURL, token),
		}
		payloadBytes, err := json.Marshal(emailPayload)
		if err != nil {
			return err
		}
		idempotencyKey := fmt.Sprintf("force_reset_user_%d_token_%s", u.ID, tokenHash)
		if s.queueRepo != nil {
			if enqErr := s.queueRepo.Enqueue(txCtx, database.GetTx(txCtx), queue.JobTypeEmailSend, payloadBytes, &idempotencyKey); enqErr != nil {
				slog.Error("failed to enqueue force reset email", "error", enqErr)
			}
		}

		u.Status = "FORCE_PASSWORD_RESET"
		return nil
	})
}

func (s *service) ForgotPassword(ctx context.Context, email string) error {
	u, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if u == nil {
		return nil
	}

	token, tokenHash, err := generateSecureToken()
	if err != nil {
		return err
	}

	v := &UserVerification{
		UserID:    u.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err = database.RunInTransaction(ctx, s.db, func(txCtx context.Context) error {
		if _, err := database.GetQueryer(txCtx, s.db).ExecContext(txCtx,
			"DELETE FROM user_verifications WHERE user_id = $1", u.ID); err != nil {
			return fmt.Errorf("delete existing verifications: %w", err)
		}
		if err := s.repo.CreateVerificationToken(txCtx, v); err != nil {
			return err
		}
		emailPayload := EmailSendPayload{
			To:      u.Email,
			Subject: "Reset Your Password - CoreKit",
			Body:    fmt.Sprintf("To reset your password, click here: %s/reset-password?token=%s", s.frontendURL, token),
		}
		payloadBytes, err := json.Marshal(emailPayload)
		if err != nil {
			return err
		}
		idempotencyKey := fmt.Sprintf("forgot_password_user_%d_token_%s", u.ID, tokenHash)
		if s.queueRepo != nil {
			if enqErr := s.queueRepo.Enqueue(txCtx, database.GetTx(txCtx), queue.JobTypeEmailSend, payloadBytes, &idempotencyKey); enqErr != nil {
				slog.Error("failed to enqueue forgot password email", "error", enqErr)
			}
		}
		return nil
	})
	return err
}

func (s *service) ResetPassword(ctx context.Context, token, newPassword string) error {
	h := sha256.New()
	h.Write([]byte(token))
	tokenHash := hex.EncodeToString(h.Sum(nil))

	return database.RunInTransaction(ctx, s.db, func(txCtx context.Context) error {
		v, err := s.repo.GetVerificationByTokenHash(txCtx, tokenHash)
		if err != nil {
			return err
		}
		if v == nil {
			return errors.New("invalid or expired reset token")
		}
		if time.Now().After(v.ExpiresAt) {
			_ = s.repo.DeleteVerificationToken(txCtx, v.ID)
			return errors.New("reset token has expired")
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
		if err != nil {
			return err
		}

		q := database.GetQueryer(txCtx, s.db)
		_, err = q.ExecContext(txCtx,
			`UPDATE users SET password_hash = $1, status = 'ACTIVE', failed_login_attempts = 0, locked_until = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
			string(hashedPassword), v.UserID)
		if err != nil {
			return err
		}

		if err := s.repo.DeleteVerificationToken(txCtx, v.ID); err != nil {
			return err
		}
		if err := s.repo.RevokeAllSessions(txCtx, v.UserID, "", nil); err != nil {
			slog.Error("failed to revoke sessions on password reset", "error", err)
		}
		return nil
	})
}

func sha256Hex(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
