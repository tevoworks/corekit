package iam

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/tevoworks/corekit/backend/internal/middleware"
	"github.com/tevoworks/corekit/backend/internal/validation"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

type oauthCode struct {
	UserID    int64
	TokenID   string
	ExpiresAt time.Time
}

var (
	oauthCodes       = make(map[string]oauthCode)
	oauthCodesMu     sync.Mutex
	oauthCleanupOnce sync.Once
	oauthInitLog     sync.Once
)

func (h *Handler) generateOAuthCode(userID int64, tokenID string) string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	code := hex.EncodeToString(b)

	oc := oauthCode{
		UserID:    userID,
		TokenID:   tokenID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if h.oauthRedis != nil {
		data, _ := json.Marshal(oc)
		if err := h.oauthRedis.Set(context.Background(), "oauth:"+code, data, 5*time.Minute).Err(); err != nil {
			slog.Error("Failed to store OAuth code in Redis", "error", err)
		}
		return code
	}

	oauthInitLog.Do(func() {
		slog.Warn("OAuth code storage falling back to in-memory map — not shared across instances, use Redis in multi-replica deployments")
	})
	oauthCodesMu.Lock()
	oauthCodes[code] = oc
	oauthCodesMu.Unlock()

	oauthCleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-h.ctx.Done():
					return
				case <-ticker.C:
					oauthCodesMu.Lock()
					for k, v := range oauthCodes {
						if time.Now().After(v.ExpiresAt) {
							delete(oauthCodes, k)
						}
					}
					oauthCodesMu.Unlock()
				}
			}
		}()
	})

	return code
}

func (h *Handler) consumeOAuthCode(code string) (oauthCode, bool) {
	if h.oauthRedis != nil {
		data, err := h.oauthRedis.GetDel(context.Background(), "oauth:"+code).Bytes()
		if err != nil {
			return oauthCode{}, false
		}
		var oc oauthCode
		if err := json.Unmarshal(data, &oc); err != nil {
			return oauthCode{}, false
		}
		if time.Now().After(oc.ExpiresAt) {
			return oauthCode{}, false
		}
		return oc, true
	}

	oauthCodesMu.Lock()
	defer oauthCodesMu.Unlock()

	oc, ok := oauthCodes[code]
	if !ok || time.Now().After(oc.ExpiresAt) {
		return oauthCode{}, false
	}
	delete(oauthCodes, code)
	return oc, true
}

type Handler struct {
	service            Service
	rbacVerifier       middleware.RBACVerifier
	db                 *sql.DB
	jwtSecret          string
	appEnv             string
	googleClientID     string
	googleClientSecret string
	googleRedirectURL  string
	frontendURL        string
	oauthRedis         *redis.Client
	httpClient         *http.Client
	ctx                context.Context
	cancel             context.CancelFunc
}

func (h *Handler) setTokenCookie(c echo.Context, token string) {
	c.SetCookie(&http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   h.appEnv == "production",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearTokenCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   h.appEnv == "production",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
}

func NewHandler(s Service, rbacVerifier middleware.RBACVerifier, db *sql.DB, jwtSecret string, appEnv string, googleClientID, googleClientSecret, googleRedirectURL, frontendURL string, redisURL string) *Handler {
	var rdb *redis.Client
	if redisURL != "" {
		if opts, err := redis.ParseURL(redisURL); err == nil {
			rdb = redis.NewClient(opts)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Handler{
		service:            s,
		rbacVerifier:       rbacVerifier,
		db:                 db,
		jwtSecret:          jwtSecret,
		appEnv:             appEnv,
		googleClientID:     googleClientID,
		googleClientSecret: googleClientSecret,
		googleRedirectURL:  googleRedirectURL,
		frontendURL:        frontendURL,
		oauthRedis:         rdb,
		httpClient:         &http.Client{Timeout: 10 * time.Second},
		ctx:                ctx,
		cancel:             cancel,
	}
}

func (h *Handler) Stop() {
	h.cancel()
}

func (h *Handler) RegisterRoutes(e *echo.Echo, globalGroup *echo.Group, authMW echo.MiddlewareFunc) {
	// Public routes
	e.POST("/api/auth/register", h.Register, middleware.LimitIP(3))
	e.POST("/api/auth/login", h.Login, middleware.LimitIP(10), middleware.LimitEmail(5))
	e.GET("/api/auth/verify", h.VerifyEmail, middleware.LimitIP(10))
	e.POST("/api/auth/verify", h.VerifyEmail, middleware.LimitIP(10))
	e.POST("/api/auth/verify-resend", h.ResendVerification, middleware.LimitIP(3))
	e.GET("/api/auth/oauth/google", h.GoogleOAuthLogin, middleware.LimitIP(10))
	e.GET("/api/auth/oauth/google/callback", h.GoogleOAuthCallback, middleware.LimitIP(10))
	e.POST("/api/auth/exchange-code", h.ExchangeCode, middleware.LimitIP(5))
	e.POST("/api/auth/refresh", h.RefreshToken, middleware.LimitIP(5))
	e.POST("/api/auth/forgot-password", h.ForgotPassword, middleware.LimitIP(5), middleware.LimitEmail(3))
	e.POST("/api/auth/reset-password", h.ResetPassword, middleware.LimitIP(5), middleware.LimitEmail(3))

	// Authenticated user routes
	globalGroup.GET("/me", h.Me)
	globalGroup.POST("/logout", h.Logout)
	globalGroup.POST("/logout-all", h.LogoutAll)
	globalGroup.PATCH("/profile", h.UpdateProfile)
	globalGroup.GET("/sessions", h.ListMySessions)
	globalGroup.DELETE("/sessions/:token_id", h.RevokeMySession)
	globalGroup.GET("/preferences", h.ListUserPreferences)
	globalGroup.PUT("/preferences/:key", h.UpsertUserPreference)
	globalGroup.GET("/identities", h.ListLinkedAccounts)
	globalGroup.DELETE("/identities/:id", h.UnlinkAccount)

	// Notification Preferences
	globalGroup.GET("/notifications/preferences", h.GetNotificationPreferences)
	globalGroup.PUT("/notifications/preferences/:type", h.UpdateNotificationPreference)

	// Notifications
	globalGroup.GET("/notifications", h.ListNotifications)
	globalGroup.GET("/notifications/unread-count", h.GetUnreadCount)
	globalGroup.PATCH("/notifications/:id/read", h.MarkNotificationRead)
	globalGroup.POST("/notifications/read-all", h.MarkAllNotificationsRead)
	globalGroup.DELETE("/notifications/:id", h.DeleteNotification)

	// Data Export & Account Deletion
	globalGroup.POST("/export-data", h.ExportMyData)
	globalGroup.DELETE("/account", h.DeleteMyAccount)

	// Admin routes (require RBAC manage:users or super_admin)
	globalGroup.GET("/users", h.ListGlobalUsers, middleware.LimitIP(60), middleware.RBACMiddleware(h.rbacVerifier, "read:users"))
	globalGroup.POST("/users", h.AdminCreateUser, middleware.LimitIP(20), middleware.RBACMiddleware(h.rbacVerifier, "manage:users"))
	globalGroup.PUT("/users/:id", h.AdminUpdateUser, middleware.LimitIP(20), middleware.RBACMiddleware(h.rbacVerifier, "manage:users"))
	globalGroup.PATCH("/users/:id/status", h.AdminChangeUserStatus, middleware.LimitIP(20), middleware.RBACMiddleware(h.rbacVerifier, "manage:users"))
	globalGroup.DELETE("/users/:id", h.AdminDeleteUser, middleware.LimitIP(10), middleware.RBACMiddleware(h.rbacVerifier, "manage:users"))
	globalGroup.POST("/users/:id/force-reset", h.AdminForceResetPassword, middleware.LimitIP(10), middleware.RBACMiddleware(h.rbacVerifier, "manage:users"))
	globalGroup.PUT("/users/:id/role", h.UpdateUserRole, middleware.LimitIP(20), middleware.RBACMiddleware(h.rbacVerifier, "manage:users"))
	globalGroup.POST("/impersonate", h.Impersonate, middleware.LimitIP(10), middleware.RBACMiddleware(h.rbacVerifier, "manage:users"))
	globalGroup.POST("/users/promote", h.PromoteToSuperAdmin, middleware.LimitIP(10), middleware.RBACMiddleware(h.rbacVerifier, "manage:users"))
	globalGroup.GET("/sessions/all", h.ListGlobalSessions, middleware.LimitIP(30), middleware.RBACMiddleware(h.rbacVerifier, "read:sessions"))
	globalGroup.DELETE("/sessions/all/:token_id", h.RevokeSession, middleware.LimitIP(20), middleware.RBACMiddleware(h.rbacVerifier, "manage:users"))
	globalGroup.GET("/users/:id/sessions", h.ListUserSessions, middleware.LimitIP(30), middleware.RBACMiddleware(h.rbacVerifier, "read:sessions"))
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,emailfmt"`
	Password string `json:"password" validate:"required,password"`
	FullName string `json:"full_name" validate:"required,nohtml"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,emailfmt"`
	Password string `json:"password" validate:"required,password"`
}

func (h *Handler) Register(c echo.Context) error {
	ctx := c.Request().Context()

	// Only allow registration when no users exist (first-time setup)
	var count int
	if err := h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		slog.Error("failed to check existing users count", "error", err)
		return httputil.InternalError(c)
	}
	if count > 0 {
		return httputil.Forbidden(c, "Registration is closed")
	}

	var req RegisterRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	u, err := h.service.Register(ctx, email, req.Password, req.FullName, false)
	if err != nil {
		slog.Error("registration failed", "error", err.Error())
		return httputil.InternalError(c)
	}

	// First user is auto-activated — return token immediately
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()
	token, _, err := h.service.Login(ctx, email, req.Password, ipAddress, userAgent)
	if err == nil {
		h.setTokenCookie(c, token)
		return httputil.Created(c, map[string]interface{}{
			"token": token,
			"user":  u,
		})
	}

	return httputil.Created(c, map[string]string{"message": "Verification email sent. Please check your inbox."})
}

func (h *Handler) Login(c echo.Context) error {
	ctx := c.Request().Context()
	var req LoginRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	token, u, err := h.service.Login(ctx, email, req.Password, ipAddress, userAgent)
	if err != nil {
		return httputil.Unauthorized(c, "Invalid email or password")
	}

	h.setTokenCookie(c, token)

	return httputil.OK(c, map[string]interface{}{
		"user": u,
	})
}

func (h *Handler) Me(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	u, err := h.service.GetProfile(ctx, userID)
	if err != nil {
		return httputil.InternalError(c)
	}

	if u == nil {
		return httputil.NotFound(c, "User not found")
	}

	return httputil.OK(c, u)
}

func (h *Handler) Logout(c echo.Context) error {
	ctx := c.Request().Context()
	tokenID := middleware.GetTokenID(c)
	actorID := middleware.GetUserID(c)
	var actorPtr *int64
	if actorID > 0 {
		actorPtr = &actorID
	}

	if tokenID != "" {
		err := h.service.RevokeSession(ctx, tokenID, actorPtr)
		if err != nil {
			return httputil.InternalError(c)
		}
	}

	h.clearTokenCookie(c)

	return httputil.Message(c, "Logged out successfully")
}

func (h *Handler) LogoutAll(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)
	var actorPtr *int64
	if actorID > 0 {
		actorPtr = &actorID
	}

	tokenID := middleware.GetTokenID(c)

	if actorID > 0 {
		err := h.service.RevokeAllSessions(ctx, actorID, tokenID, actorPtr)
		if err != nil {
			return httputil.InternalError(c)
		}
	}

	return httputil.Message(c, "Logged out from all sessions successfully")
}

func (h *Handler) ListGlobalSessions(c echo.Context) error {
	if !middleware.IsSuperAdmin(c) {
		return httputil.Forbidden(c, "Only super administrators can list all sessions")
	}
	ctx := c.Request().Context()
	limitVal := 50
	cursorVal := int64(0)
	if l := c.QueryParam("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limitVal = val
		}
	}
	if limitVal > 100 {
		limitVal = 100
	}
	if limitVal < 1 {
		limitVal = 50
	}
	if cs := c.QueryParam("cursor"); cs != "" {
		if val, err := strconv.ParseInt(cs, 10, 64); err == nil {
			cursorVal = val
		}
	}

	list, err := h.service.ListGlobalSessions(ctx, limitVal, cursorVal)
	if err != nil {
		return httputil.InternalError(c)
	}

	for i := range list {
		list[i].MaskIP()
	}

	nextCursor := int64(0)
	if len(list) > 0 {
		nextCursor = list[len(list)-1].ID
	}

	return httputil.OKWithMeta(c, list, map[string]interface{}{
		"limit":       limitVal,
		"cursor":      cursorVal,
		"next_cursor": nextCursor,
		"count":       len(list),
	})
}

func (h *Handler) RevokeSession(c echo.Context) error {
	ctx := c.Request().Context()
	tokenID := c.Param("token_id")
	actorID := middleware.GetUserID(c)
	var actorPtr *int64
	if actorID > 0 {
		actorPtr = &actorID
	}

	err := h.service.RevokeSession(ctx, tokenID, actorPtr)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Session revoked successfully")
}

type UpdateProfileRequest struct {
	Email       string  `json:"email" validate:"required,emailfmt"`
	FullName    string  `json:"full_name" validate:"required,nohtml"`
	AvatarURL   string  `json:"avatar_url" validate:"omitempty,urlstrict"`
	Password    *string `json:"password,omitempty" validate:"omitempty,password"`
	OldPassword *string `json:"old_password,omitempty"`
}

func (h *Handler) UpdateProfile(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return httputil.Unauthorized(c, "User not authenticated")
	}

	var req UpdateProfileRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Password != nil && *req.Password != "" {
		if req.OldPassword == nil || *req.OldPassword == "" {
			return httputil.BadRequest(c, "Current password is required to set a new password")
		}
	}

	u, err := h.service.UpdateProfile(ctx, userID, req.Email, req.FullName, req.AvatarURL, req.Password, req.OldPassword, userID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, u)
}

func (h *Handler) ListMySessions(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return httputil.Unauthorized(c, "User not authenticated")
	}

	list, err := h.service.ListSessions(ctx, userID)
	if err != nil {
		return httputil.InternalError(c)
	}

	for i := range list {
		list[i].MaskIP()
	}

	return httputil.OKWithMeta(c, list, map[string]interface{}{
		"count": len(list),
	})
}

func (h *Handler) RevokeMySession(c echo.Context) error {
	ctx := c.Request().Context()
	tokenID := c.Param("token_id")
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return httputil.Unauthorized(c, "User not authenticated")
	}

	// Verify ownership by loading single session with user_id filter
	sess, err := h.service.GetSessionByUserAndToken(ctx, userID, tokenID)
	if err != nil {
		return httputil.InternalError(c)
	}
	if sess == nil {
		return httputil.Forbidden(c, "You can only revoke your own sessions")
	}

	var actorPtr *int64
	if userID > 0 {
		actorPtr = &userID
	}

	err = h.service.RevokeSession(ctx, tokenID, actorPtr)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Session revoked successfully")
}

func (h *Handler) VerifyEmail(c echo.Context) error {
	ctx := c.Request().Context()
	token := c.QueryParam("token")
	if token == "" {
		var req struct {
			Token string `json:"token"`
		}
		if err := c.Bind(&req); err == nil && req.Token != "" {
			token = req.Token
		}
	}
	if token == "" {
		return httputil.BadRequest(c, "Verification token is required")
	}

	u, err := h.service.VerifyEmail(ctx, token)
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/#/?verify=invalid")
	}

	isInvite := c.QueryParam("invite") == "true"
	if isInvite {
		tokenIDBytes := make([]byte, 32)
		if _, err := rand.Read(tokenIDBytes); err != nil {
			return c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/#/?verify=error")
		}
		tokenID := hex.EncodeToString(tokenIDBytes)

		sess := &Session{
			UserID:            u.ID,
			TokenID:           tokenID,
			IPAddress:         c.RealIP(),
			UserAgent:         c.Request().UserAgent(),
			ExpiresAt:         time.Now().Add(24 * time.Hour),
			AbsoluteExpiresAt: time.Now().Add(absoluteSessionTTL),
		}
		if err := h.service.CreateSession(ctx, sess); err != nil {
			slog.Error("failed to create session after email verification", "error", err)
		}

		oneTimeCode := h.generateOAuthCode(u.ID, tokenID)
		if oneTimeCode == "" {
			slog.Error("failed to generate OAuth code for invite verification", "user_id", u.ID)
			return c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/#/?verify=error")
		}

		return c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/#/?code="+oneTimeCode+"&force_reset=true")
	}

	return c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/#/?verify=success")
}

type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,emailfmt"`
}

func (h *Handler) ResendVerification(c echo.Context) error {
	ctx := c.Request().Context()
	var req ResendVerificationRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	_, err := h.service.ResendVerification(ctx, email)
	if err != nil {
		slog.Error("resend verification failed", "error", err)
	}

	return httputil.Message(c, "Verification email sent. Please check your inbox.")
}

func generatePKCEPair() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return verifier, challenge, nil
}

func (h *Handler) GoogleOAuthLogin(c echo.Context) error {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate secure OAuth state")
	}
	state := hex.EncodeToString(b)

	codeVerifier, codeChallenge, err := generatePKCEPair()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate PKCE parameters")
	}

	redirectParam := c.QueryParam("redirect")
	if redirectParam == "" || !isAllowedRedirectURL(redirectParam, h.frontendURL) {
		redirectParam = h.frontendURL
	}

	setOAuthCookie := func(name, value string) {
		c.SetCookie(&http.Cookie{
			Name:     name,
			Value:    value,
			Expires:  time.Now().Add(15 * time.Minute),
			HttpOnly: true,
			Secure:   h.appEnv == "production",
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
		})
	}
	setOAuthCookie("oauth_state", state)
	setOAuthCookie("oauth_code_verifier", codeVerifier)
	setOAuthCookie("oauth_redirect", redirectParam)

	googleAuthURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid%%20email%%20profile&state=%s&code_challenge=%s&code_challenge_method=S256&access_type=offline",
		h.googleClientID,
		url.QueryEscape(h.googleRedirectURL),
		state,
		codeChallenge,
	)

	return c.Redirect(http.StatusTemporaryRedirect, googleAuthURL)
}

func (h *Handler) clearOAuthCookies(c echo.Context) {
	for _, name := range []string{"oauth_state", "oauth_code_verifier", "oauth_redirect"} {
		c.SetCookie(&http.Cookie{
			Name:     name,
			Value:    "",
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
			Secure:   h.appEnv == "production",
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func (h *Handler) GoogleOAuthCallback(c echo.Context) error {
	state := c.QueryParam("state")

	cookie, err := c.Cookie("oauth_state")
	if err != nil || cookie == nil || cookie.Value == "" || state == "" || state != cookie.Value {
		slog.Warn("oauth callback: invalid state", slog.String("state", state))
		h.clearOAuthCookies(c)
		return c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/#/?error=oauth_failed")
	}

	redirectURL := h.frontendURL
	if redirectCookie, err := c.Cookie("oauth_redirect"); err == nil && redirectCookie != nil && redirectCookie.Value != "" {
		if isAllowedRedirectURL(redirectCookie.Value, h.frontendURL) {
			redirectURL = redirectCookie.Value
		}
	}

	h.clearOAuthCookies(c)

	code := c.QueryParam("code")
	if code == "" {
		slog.Warn("oauth callback: missing authorization code")
		return c.Redirect(http.StatusTemporaryRedirect, redirectURL+"/#/?error=oauth_failed")
	}

	codeVerifierCookie, _ := c.Cookie("oauth_code_verifier")
	codeVerifier := ""
	if codeVerifierCookie != nil {
		codeVerifier = codeVerifierCookie.Value
	}

	tokenResp, err := h.exchangeGoogleCode(code, codeVerifier)
	if err != nil {
		slog.Warn("oauth callback: token exchange failed", slog.String("error", err.Error()))
		return c.Redirect(http.StatusTemporaryRedirect, redirectURL+"/#/?error=oauth_failed")
	}

	claims, err := h.verifyGoogleIDToken(tokenResp.IDToken)
	var emailVerified bool
	if err == nil && claims != nil {
		switch v := claims.EmailVerified.(type) {
		case bool:
			emailVerified = v
		case string:
			emailVerified = (v == "true")
		}
	}
	if err != nil || claims.Email == "" || !emailVerified {
		slog.Warn("oauth callback: invalid ID token",
			slog.String("email", func() string {
				if claims != nil {
					return claims.Email
				}
				return ""
			}()),
			slog.Bool("verified", emailVerified),
		)
		return c.Redirect(http.StatusTemporaryRedirect, redirectURL+"/#/?error=oauth_failed")
	}

	ctx := c.Request().Context()

	identity, err := h.service.GetIdentityByProvider(ctx, "google", claims.Sub)
	if err != nil {
		slog.Warn("oauth callback: identity lookup failed", slog.String("error", err.Error()))
		return c.Redirect(http.StatusTemporaryRedirect, redirectURL+"/#/?error=oauth_failed")
	}

	var user *User
	if identity != nil {
		user, err = h.service.GetProfile(ctx, identity.UserID)
		if err != nil || user == nil {
			slog.Warn("oauth callback: user not found from identity", slog.Int64("user_id", identity.UserID))
			return c.Redirect(http.StatusTemporaryRedirect, redirectURL+"/#/?error=oauth_failed")
		}
	} else {
		existingUser, err := h.service.GetUserByEmail(ctx, claims.Email)
		if err == nil && existingUser != nil {
			if existingUser.DeletedAt != nil {
				slog.Warn("oauth callback: user account deleted", slog.String("email", claims.Email), slog.Int64("user_id", existingUser.ID))
				return c.Redirect(http.StatusTemporaryRedirect, redirectURL+"/#/?error=oauth_failed")
			}
			err = h.service.LinkOAuthIdentity(ctx, existingUser.ID, "google", claims.Sub)
			if err != nil {
				slog.Warn("oauth callback: identity linking failed", slog.Int64("user_id", existingUser.ID), slog.String("error", err.Error()))
				return c.Redirect(http.StatusTemporaryRedirect, redirectURL+"/#/?error=oauth_failed")
			}
			user = existingUser
		} else {
			user, err = h.service.CreateOAuthUser(ctx, claims.Email, claims.Name, "google", claims.Sub)
			if err != nil {
				slog.Warn("oauth callback: user creation failed", slog.String("email", claims.Email), slog.String("error", err.Error()))
				return c.Redirect(http.StatusTemporaryRedirect, redirectURL+"/#/?error=oauth_failed")
			}
		}
	}

	tokenIDBytes := make([]byte, 32)
	if _, err := rand.Read(tokenIDBytes); err != nil {
		slog.Error("oauth callback: token id generation failed", "error", err)
		return c.Redirect(http.StatusTemporaryRedirect, redirectURL+"/#/?error=oauth_failed")
	}
	tokenID := hex.EncodeToString(tokenIDBytes)

	sess := &Session{
		UserID:            user.ID,
		TokenID:           tokenID,
		IPAddress:         c.RealIP(),
		UserAgent:         c.Request().UserAgent(),
		ExpiresAt:         time.Now().Add(24 * time.Hour),
		AbsoluteExpiresAt: time.Now().Add(absoluteSessionTTL),
	}
	if err := h.service.CreateSession(ctx, sess); err != nil {
		slog.Error("failed to create session after OAuth callback", "error", err)
	}

	oneTimeCode := h.generateOAuthCode(user.ID, tokenID)
	if oneTimeCode == "" {
		slog.Error("failed to generate OAuth code for google callback", "user_id", user.ID)
		return c.Redirect(http.StatusTemporaryRedirect, redirectURL+"/#/?error=oauth_failed")
	}

	return c.Redirect(http.StatusTemporaryRedirect, redirectURL+"/#/?code="+oneTimeCode)
}

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
}

type googleClaims struct {
	Sub           string      `json:"sub"`
	Email         string      `json:"email"`
	Name          string      `json:"name"`
	EmailVerified interface{} `json:"email_verified"`
	Picture       string      `json:"picture"`
	Audience      string      `json:"aud"`
	IssuedAt      int64       `json:"iat"`
	ExpiresAt     int64       `json:"exp"`
	Issuer        string      `json:"iss"`
}

func (c *googleClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.ExpiresAt, 0)), nil
}
func (c *googleClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.IssuedAt, 0)), nil
}
func (c *googleClaims) GetNotBefore() (*jwt.NumericDate, error) { return nil, nil }
func (c *googleClaims) GetIssuer() (string, error)              { return c.Issuer, nil }
func (c *googleClaims) GetSubject() (string, error)             { return c.Sub, nil }
func (c *googleClaims) GetAudience() (jwt.ClaimStrings, error) {
	if c.Audience == "" {
		return nil, nil
	}
	return jwt.ClaimStrings{c.Audience}, nil
}

type jwkKey struct {
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

var (
	googleJWKS     []jwkKey
	googleJWKSMu   sync.RWMutex
	googleJWKSOnce sync.Once
)

func (h *Handler) fetchGoogleJWKS() ([]jwkKey, error) {
	resp, err := h.httpClient.Get("https://www.googleapis.com/oauth2/v3/certs")
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("parse jwks: %w", err)
	}
	return jwks.Keys, nil
}

func (h *Handler) refreshJWKS() {
	keys, err := h.fetchGoogleJWKS()
	if err != nil {
		slog.Error("failed google jwks refresh", "error", err)
		return
	}
	googleJWKSMu.Lock()
	googleJWKS = keys
	googleJWKSMu.Unlock()
	slog.Debug("google jwks refreshed", "key_count", len(keys))
}

// startJWKSRefresher runs an initial fetch then refreshes every hour.
func (h *Handler) startJWKSRefresher() {
	googleJWKSOnce.Do(func() {
		h.refreshJWKS()
		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-h.ctx.Done():
					return
				case <-ticker.C:
					h.refreshJWKS()
				}
			}
		}()
	})
}

func (h *Handler) getJWKS() ([]jwkKey, error) {
	h.startJWKSRefresher()

	googleJWKSMu.RLock()
	defer googleJWKSMu.RUnlock()
	if len(googleJWKS) == 0 {
		return nil, fmt.Errorf("google jwks not yet loaded")
	}
	return googleJWKS, nil
}

func (h *Handler) exchangeGoogleCode(code, codeVerifier string) (*googleTokenResponse, error) {
	data := url.Values{
		"code":          {code},
		"client_id":     {h.googleClientID},
		"client_secret": {h.googleClientSecret},
		"redirect_uri":  {h.googleRedirectURL},
		"grant_type":    {"authorization_code"},
	}
	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	resp, err := h.httpClient.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	var tr googleTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tr.Error != "" {
		return nil, fmt.Errorf("google token error: %s", tr.Error)
	}

	return &tr, nil
}

func (h *Handler) verifyGoogleIDToken(idToken string) (*googleClaims, error) {
	return h.verifyGoogleIDTokenLocal(idToken)
}

func (h *Handler) verifyGoogleIDTokenLocal(idToken string) (*googleClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}
	if header.Kid == "" {
		return nil, fmt.Errorf("missing kid")
	}

	keys, err := h.getJWKS()
	if err != nil {
		return nil, err
	}

	var matchingKey *jwkKey
	for _, k := range keys {
		if k.Kid == header.Kid {
			matchingKey = &k
			break
		}
	}
	if matchingKey == nil {
		return nil, fmt.Errorf("key not found for kid: %s", header.Kid)
	}

	rsaKey, err := matchingKey.rsaPublicKey()
	if err != nil {
		return nil, fmt.Errorf("parse rsa key: %w", err)
	}

	verifiedClaims := &googleClaims{}
	_, err = jwt.ParseWithClaims(idToken, verifiedClaims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return rsaKey, nil
	}, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return nil, fmt.Errorf("jwt verify: %w", err)
	}

	if verifiedClaims.Sub == "" {
		return nil, fmt.Errorf("invalid token: missing subject")
	}
	if verifiedClaims.Audience != h.googleClientID {
		slog.Error("OAuth token audience mismatch", "expected", h.googleClientID, "got", verifiedClaims.Audience)
		return nil, fmt.Errorf("token audience mismatch")
	}

	return verifiedClaims, nil
}

func (k *jwkKey) rsaPublicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)

	var e int
	if len(eBytes) < 8 {
		eBytes = append(make([]byte, 8-len(eBytes)), eBytes...)
	}
	e64 := binary.BigEndian.Uint64(eBytes[len(eBytes)-8:])
	if e64 > uint64(1<<31-1) {
		return nil, fmt.Errorf("public exponent %d exceeds maximum allowed value", e64)
	}
	e = int(e64)

	return &rsa.PublicKey{N: n, E: e}, nil
}

func isAllowedRedirectURL(rawURL, allowedBase string) bool {
	if rawURL == "" || allowedBase == "" {
		return false
	}
	if strings.HasPrefix(rawURL, "//") {
		return false
	}
	if strings.Contains(rawURL, "..") {
		return false
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.IsAbs() {
		allowed, err := url.Parse(allowedBase)
		if err != nil {
			return false
		}
		if u.User != nil {
			return false
		}
		return u.Scheme == allowed.Scheme && u.Host == allowed.Host
	}
	return false
}

type ExchangeCodeRequest struct {
	Code string `json:"code" validate:"required"`
}

func (h *Handler) ExchangeCode(c echo.Context) error {
	var req ExchangeCodeRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	oc, ok := h.consumeOAuthCode(req.Code)
	if !ok {
		return httputil.Unauthorized(c, "Invalid or expired code")
	}

	ctx := c.Request().Context()

	user, err := h.service.GetProfile(ctx, oc.UserID)
	if err != nil || user == nil {
		return httputil.InternalError(c)
	}

	tokenID := oc.TokenID
	role := ""
	if user.RoleID != nil && user.RoleName != nil {
		role = *user.RoleName
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":        user.ID,
		"role":           role,
		"is_super_admin": user.IsSuperAdmin,
		"token_id":       tokenID,
		"iat":            time.Now().Unix(),
		"jti":            tokenID,
		"exp":            time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		return httputil.InternalError(c)
	}

	h.setTokenCookie(c, tokenString)

	return httputil.OK(c, map[string]interface{}{
		"user_id":    user.ID,
		"expires_in": 86400,
	})
}

func (h *Handler) RefreshToken(c echo.Context) error {
	cookie, err := c.Cookie("token")
	if err != nil {
		return httputil.Unauthorized(c, "Missing authentication token")
	}

	ctx := c.Request().Context()
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	newToken, user, err := h.service.RefreshSession(ctx, cookie.Value, ipAddress, userAgent)
	if err != nil {
		return httputil.Unauthorized(c, "Unable to refresh session. Please log in again.")
	}

	h.setTokenCookie(c, newToken)

	return httputil.OK(c, map[string]interface{}{
		"user": user,
	})
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,emailfmt"`
}

func (h *Handler) ForgotPassword(c echo.Context) error {
	ctx := c.Request().Context()
	var req ForgotPasswordRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if err := h.service.ForgotPassword(ctx, email); err != nil {
		slog.Error("forgot password failed", "error", err)
	}

	return httputil.Message(c, "If the email exists, a password reset link has been sent.")
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,password"`
}

func (h *Handler) ResetPassword(c echo.Context) error {
	ctx := c.Request().Context()
	var req ResetPasswordRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.service.ResetPassword(ctx, req.Token, req.Password); err != nil {
		return httputil.BadRequest(c, "Invalid or expired reset token")
	}

	return httputil.Message(c, "Password has been reset successfully. Please log in with your new password.")
}

type UpdateNotificationPreferenceRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *Handler) GetNotificationPreferences(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	prefs, err := h.service.GetNotificationPreferences(ctx, userID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, prefs)
}

var allowedNotificationTypes = []string{
	"email",
	"push",
	"in_app",
	"security_alert",
	"marketing",
	"system",
}

func (h *Handler) UpdateNotificationPreference(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)
	notificationType := c.Param("type")

	if notificationType == "" {
		return httputil.ValidationError(c, "Notification type is required")
	}
	if len(notificationType) > 50 {
		return httputil.ValidationError(c, "Notification type must be at most 50 characters")
	}

	valid := false
	for _, t := range allowedNotificationTypes {
		if t == notificationType {
			valid = true
			break
		}
	}
	if !valid {
		return httputil.ValidationError(c, "Invalid notification type")
	}

	var req UpdateNotificationPreferenceRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	err := h.service.UpdateNotificationPreference(ctx, userID, notificationType, req.Enabled)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, map[string]interface{}{
		"user_id":           userID,
		"notification_type": notificationType,
		"enabled":           req.Enabled,
	})
}

func (h *Handler) ListNotifications(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	limitVal := 50
	var cursor *time.Time
	unreadOnly := false

	if l := c.QueryParam("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limitVal = val
		}
	}
	if limitVal > 100 {
		limitVal = 100
	}
	if limitVal < 1 {
		limitVal = 50
	}
	if cVal := c.QueryParam("cursor"); cVal != "" {
		if t, err := time.Parse(time.RFC3339, cVal); err == nil {
			cursor = &t
		}
	}
	if c.QueryParam("unread_only") == "true" {
		unreadOnly = true
	}

	list, err := h.service.GetNotifications(ctx, userID, limitVal, cursor, unreadOnly)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := ""
	if len(list) > 0 {
		nextCursor = list[len(list)-1].CreatedAt.Format(time.RFC3339)
	}

	return httputil.OKWithMeta(c, list, map[string]interface{}{
		"count":       len(list),
		"next_cursor": nextCursor,
	})
}

func (h *Handler) GetUnreadCount(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	count, err := h.service.GetUnreadCount(ctx, userID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, map[string]int{"unread_count": count})
}

func (h *Handler) MarkNotificationRead(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid notification ID")
	}

	if err := h.service.MarkAsRead(ctx, userID, id); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Notification marked as read")
}

func (h *Handler) MarkAllNotificationsRead(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	if err := h.service.MarkAllAsRead(ctx, userID); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "All notifications marked as read")
}

func (h *Handler) DeleteNotification(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid notification ID")
	}

	if err := h.service.DeleteNotification(ctx, userID, id); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Notification deleted")
}

type UpsertPreferenceRequest struct {
	Value string `json:"value" validate:"required"`
}

func (h *Handler) ListUserPreferences(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	prefs, err := h.service.ListPreferences(ctx, userID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, prefs)
}

func (h *Handler) UpsertUserPreference(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)
	key := c.Param("key")

	if key == "" {
		return httputil.ValidationError(c, "Key is required")
	}

	var req UpsertPreferenceRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.service.UpsertPreference(ctx, userID, key, req.Value); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Preference saved")
}

func (h *Handler) ExportMyData(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	data, err := h.service.ExportMyData(ctx, userID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, data)
}

type DeleteMyAccountRequest struct {
	Password *string `json:"password,omitempty"`
	Confirm  *bool   `json:"confirm,omitempty"`
}

func (h *Handler) DeleteMyAccount(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	var req DeleteMyAccountRequest
	if err := c.Bind(&req); err != nil {
		return httputil.BadRequest(c, "Invalid request body")
	}

	u, err := h.service.GetProfile(ctx, userID)
	if err != nil || u == nil {
		return httputil.NotFound(c, "User not found")
	}

	if u.PasswordHash != "" {
		if req.Password == nil || *req.Password == "" {
			return httputil.BadRequest(c, "Password is required to delete your account")
		}
		if err := h.service.VerifyPassword(ctx, userID, *req.Password); err != nil {
			return httputil.Unauthorized(c, "Current password is incorrect")
		}
	} else {
		if !isSessionRecent(ctx, h.db, middleware.GetTokenID(c)) {
			return httputil.BadRequest(c, "Please refresh your authentication before deleting your account")
		}
	}

	if err := h.service.DeleteMyAccount(ctx, userID); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Your account has been deleted.")
}

func isSessionRecent(ctx context.Context, db *sql.DB, tokenID string) bool {
	if tokenID == "" {
		return false
	}
	var createdAt time.Time
	err := db.QueryRowContext(ctx,
		`SELECT created_at FROM sessions WHERE token_id = $1`, tokenID,
	).Scan(&createdAt)
	if err != nil {
		return false
	}
	return time.Since(createdAt) < 5*time.Minute
}

func (h *Handler) ListLinkedAccounts(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	idents, err := h.service.GetLinkedAccounts(ctx, userID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, idents)
}

func (h *Handler) UnlinkAccount(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid identity ID")
	}

	if err := h.service.UnlinkAccount(ctx, userID, id); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Account unlinked successfully.")
}

func (h *Handler) ListGlobalUsers(c echo.Context) error {
	ctx := c.Request().Context()

	limitVal := 50
	cursorVal := int64(0)
	if l := c.QueryParam("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limitVal = val
		}
	}
	if limitVal > 100 {
		limitVal = 100
	}
	if limitVal < 1 {
		limitVal = 50
	}

	if cs := c.QueryParam("cursor"); cs != "" {
		if val, err := strconv.ParseInt(cs, 10, 64); err == nil {
			cursorVal = val
		}
	}

	users, err := h.service.ListGlobalUsers(ctx, limitVal, cursorVal)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(users) > 0 {
		nextCursor = users[len(users)-1].ID
	}

	return httputil.OKWithMeta(c, users, map[string]interface{}{
		"count":       len(users),
		"limit":       limitVal,
		"cursor":      cursorVal,
		"next_cursor": nextCursor,
	})
}

type ImpersonateRequest struct {
	TargetUserID int64 `json:"target_user_id" validate:"required,gt=0"`
}

func (h *Handler) Impersonate(c echo.Context) error {
	ctx := c.Request().Context()
	var req ImpersonateRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	impersonatorID := middleware.GetUserID(c)
	if impersonatorID == 0 {
		return httputil.Unauthorized(c, "Unauthorized impersonator")
	}

	tokenStr, err := h.service.Impersonate(ctx, impersonatorID, req.TargetUserID)
	if err != nil {
		return httputil.InternalError(c)
	}

	c.SetCookie(&http.Cookie{
		Name:     "token",
		Value:    tokenStr,
		Expires:  time.Now().Add(2 * time.Hour),
		HttpOnly: true,
		Secure:   h.appEnv == "production",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	return httputil.OK(c, map[string]interface{}{
		"token": tokenStr,
	})
}

type PromoteRequest struct {
	Email string `json:"email" validate:"required,emailfmt"`
}

func (h *Handler) PromoteToSuperAdmin(c echo.Context) error {
	ctx := c.Request().Context()
	var req PromoteRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	err := h.service.PromoteToSuperAdmin(ctx, req.Email)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "User promoted to Super Admin successfully")
}

type UpdateUserRoleRequest struct {
	RoleID *int64 `json:"role_id"`
}

func (h *Handler) UpdateUserRole(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid user ID")
	}

	var req UpdateUserRoleRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.service.UpdateUserRole(ctx, userID, req.RoleID, actorID); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "User role updated successfully")
}

func (h *Handler) AdminDeleteUser(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	targetUserID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	if actorID == targetUserID {
		return httputil.BadRequest(c, "You cannot delete yourself")
	}

	err = h.service.AdminDeleteUser(ctx, targetUserID, actorID)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "user not found"):
			return httputil.NotFound(c, "User not found")
		case strings.Contains(err.Error(), "cannot delete a super admin"):
			return httputil.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		default:
			return httputil.InternalError(c)
		}
	}

	return httputil.Message(c, "User deleted successfully")
}
