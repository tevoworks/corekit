package iam

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/tevoworks/corekit/backend/internal/authverify"
	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
	"github.com/tevoworks/corekit/backend/internal/modules/queue"
	"github.com/tevoworks/corekit/backend/internal/redisstore"
	"github.com/tevoworks/corekit/backend/pkg/event"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if err := database.RunMigrations(db, "../../../migrations/"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newTestService(t *testing.T) (Service, *sql.DB) {
	t.Helper()
	db := testDB(t)
	repo := NewRepository(db)
	auditSvc := audit.NewService(audit.NewRepository(db))
	queueRepo := queue.NewRepository(db)
	eventDispatcher := event.NewEventDispatcher(queueRepo)
	revStore := redisstore.NewRevocationStore("")
	cache := authverify.NewIntrospectionCache()

	svc := NewService(db, repo, "test-jwt-secret-key-min-32-chars!", auditSvc,
		revStore, queueRepo, eventDispatcher, cache, "http://localhost:5173")
	return svc, db
}

func cleanupUsers(t *testing.T, db *sql.DB) {
	t.Helper()
	_, _ = db.Exec(`DELETE FROM user_verifications`)
	_, _ = db.Exec(`DELETE FROM sessions`)
	_, _ = db.Exec(`DELETE FROM users WHERE email LIKE 'test-%@example.com'`)
}

func TestRegisterAndLogin(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-register-" + t.Name() + "@example.com"
	u, err := svc.Register(ctx, email, "StrongPass1!", "Test User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	token, user, err := svc.Login(ctx, email, "StrongPass1!", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if user == nil || user.ID != u.ID {
		t.Fatal("expected same user")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-dup-" + t.Name() + "@example.com"
	_, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err = svc.Register(ctx, email, "StrongPass2!", "User2", false)
	if err == nil {
		t.Fatal("expected error on duplicate registration")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-wrongpw-" + t.Name() + "@example.com"
	_, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, _, err = svc.Login(ctx, email, "WrongPass1!", "127.0.0.1", "test")
	if err == nil {
		t.Fatal("expected login error for wrong password")
	}
}

func TestLoginNonexistentEmail(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	_, _, err := svc.Login(ctx, "doesnotexist@example.com", "StrongPass1!", "127.0.0.1", "test")
	if err == nil {
		t.Fatal("expected error for nonexistent email")
	}
}

func TestGetProfile(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-profile-" + t.Name() + "@example.com"
	u, err := svc.Register(ctx, email, "StrongPass1!", "Profile User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	profile, err := svc.GetProfile(ctx, u.ID)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile == nil {
		t.Fatal("expected non-nil profile")
	}
	if profile.Email != email {
		t.Fatalf("expected %s, got: %s", email, profile.Email)
	}
}

func TestGetProfileNotFound(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	profile, err := svc.GetProfile(ctx, 999999)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile != nil {
		t.Fatal("expected nil for non-existent user")
	}
}

func TestUpdateProfile(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-upd-" + t.Name() + "@example.com"
	u, err := svc.Register(ctx, email, "StrongPass1!", "Original", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	newEmail := "test-upd-new-" + t.Name() + "@example.com"
	updated, err := svc.UpdateProfile(ctx, u.ID, newEmail, "Updated Name", "", nil, nil, u.ID)
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.FullName != "Updated Name" {
		t.Fatalf("expected Updated Name, got: %s", updated.FullName)
	}
}

func TestUpdateProfileWithPassword(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-pass-upd-" + t.Name() + "@example.com"
	u, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	newPass := "NewStrongPass1!"
	oldPass := "StrongPass1!"
	updated, err := svc.UpdateProfile(ctx, u.ID, email, "User", "", &newPass, &oldPass, u.ID)
	if err != nil {
		t.Fatalf("update profile with password: %v", err)
	}
	if updated == nil {
		t.Fatal("expected non-nil user")
	}
}

func TestSessionLifecycle(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-sess-" + t.Name() + "@example.com"
	_, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, user, err := svc.Login(ctx, email, "StrongPass1!", "10.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	sessions, err := svc.ListSessions(ctx, user.ID)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("expected at least 1 session")
	}

	err = svc.RevokeSession(ctx, sessions[0].TokenID, &user.ID)
	if err != nil {
		t.Fatalf("revoke session: %v", err)
	}

	sessByToken, err := svc.GetSessionByUserAndToken(ctx, user.ID, sessions[0].TokenID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sessByToken != nil && sessByToken.RevokedAt == nil {
		t.Fatal("expected session to be revoked")
	}
}

func TestRevokeAllSessions(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-revoke-all-" + t.Name() + "@example.com"
	_, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, user, err := svc.Login(ctx, email, "StrongPass1!", "10.0.0.1", "agent1")
	if err != nil {
		t.Fatalf("login 1: %v", err)
	}

	err = svc.RevokeAllSessions(ctx, user.ID, "", &user.ID)
	if err != nil {
		t.Fatalf("revoke all: %v", err)
	}

	sessions, err := svc.ListSessions(ctx, user.ID)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	for _, s := range sessions {
		if s.RevokedAt == nil {
			t.Fatal("expected all sessions to be revoked")
		}
	}
}

func TestGetSessionByUserAndToken(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-sess-lookup-" + t.Name() + "@example.com"
	_, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, user, err := svc.Login(ctx, email, "StrongPass1!", "10.0.0.1", "agent")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	sessions, err := svc.ListSessions(ctx, user.ID)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("expected sessions")
	}

	sess, err := svc.GetSessionByUserAndToken(ctx, user.ID, sessions[0].TokenID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sess == nil {
		t.Fatal("expected non-nil session")
	}
}

func TestUserPreferences(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-prefs-" + t.Name() + "@example.com"
	u, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := svc.UpsertPreference(ctx, u.ID, "theme", "dark"); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	prefs, err := svc.ListPreferences(ctx, u.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, p := range prefs {
		if p.Key == "theme" && p.Value == "dark" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected theme=dark preference")
	}
}

func TestGetUserByEmail(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-getbyemail-" + t.Name() + "@example.com"
	_, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	u, err := svc.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if u == nil {
		t.Fatal("expected user")
	}

	missing, err := svc.GetUserByEmail(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("get nonexistent: %v", err)
	}
	if missing != nil {
		t.Fatal("expected nil for non-existent email")
	}
}

func TestVerifyPassword(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-verifypw-" + t.Name() + "@example.com"
	u, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := svc.VerifyPassword(ctx, u.ID, "StrongPass1!"); err != nil {
		t.Fatalf("verify correct password: %v", err)
	}

	if err := svc.VerifyPassword(ctx, u.ID, "WrongPass1!"); err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestAccountLockout(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-lockout-" + t.Name() + "@example.com"
	_, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	for i := 0; i < 12; i++ {
		_, _, _ = svc.Login(ctx, email, "WrongPass1!", "10.0.0.1", "attacker")
	}

	_, _, err = svc.Login(ctx, email, "StrongPass1!", "10.0.0.1", "real-user")
	if err == nil {
		t.Fatal("expected error after lockout")
	}
}

func TestNotificationCRUD(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-notif-" + t.Name() + "@example.com"
	u, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	notif := &Notification{
		UserID: u.ID,
		Type:   "info",
		Title:  "Test",
		Body:   "Hello",
	}
	if err := svc.CreateNotification(ctx, notif); err != nil {
		t.Fatalf("create: %v", err)
	}

	unread, err := svc.GetUnreadCount(ctx, u.ID)
	if err != nil {
		t.Fatalf("unread count: %v", err)
	}
	if unread == 0 {
		t.Fatal("expected at least 1 unread notification")
	}

	if err := svc.MarkAsRead(ctx, u.ID, notif.ID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
}

func TestListSessions(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupUsers(t, db)
	ctx := context.Background()

	email := "test-listsess-" + t.Name() + "@example.com"
	_, err := svc.Register(ctx, email, "StrongPass1!", "User", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, u, err := svc.Login(ctx, email, "StrongPass1!", "1.2.3.4", "browser")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	_, u2, err := svc.Login(ctx, email, "StrongPass1!", "5.6.7.8", "mobile")
	if err != nil {
		t.Fatalf("login 2: %v", err)
	}

	sessions, err := svc.ListSessions(ctx, u.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) < 2 {
		t.Fatalf("expected at least 2 sessions, got: %d", len(sessions))
	}
	_ = u2
}
