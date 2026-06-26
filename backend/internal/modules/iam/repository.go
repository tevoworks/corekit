package iam

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/tevoworks/corekit/backend/internal/database"
)

type Repository interface {
	CreateUser(ctx context.Context, u *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	UpdateUser(ctx context.Context, u *User) error

	LockAccount(ctx context.Context, userID int64, lockedUntil time.Time) error
	UpdateSuperAdminStatus(ctx context.Context, email string, isSuper bool) error
	ListGlobal(ctx context.Context, limit int, cursor int64) ([]User, error)
	UpdateUserRole(ctx context.Context, userID int64, roleID *int64) error

	CreateSession(ctx context.Context, s *Session) error
	GetSessionByTokenID(ctx context.Context, tokenID string) (*Session, error)
	GetSessionByUserAndToken(ctx context.Context, userID int64, tokenID string) (*Session, error)
	RevokeSession(ctx context.Context, tokenID string, revokedBy *int64) error
	RevokeAllSessions(ctx context.Context, userID int64, exceptTokenID string, revokedBy *int64) error
	ListSessions(ctx context.Context, userID int64) ([]Session, error)
	ListGlobalSessions(ctx context.Context, limit int, cursor int64) ([]Session, error)

	// Notification Preferences
	GetNotificationPreferences(ctx context.Context, userID int64) ([]NotificationPreference, error)
	UpsertNotificationPreference(ctx context.Context, userID int64, notificationType string, enabled bool) error

	// Notifications
	ListNotifications(ctx context.Context, userID int64, limit int, cursor *time.Time, unreadOnly bool) ([]Notification, error)
	GetUnreadCount(ctx context.Context, userID int64) (int, error)
	MarkAsRead(ctx context.Context, userID, notifID int64) error
	MarkAllAsRead(ctx context.Context, userID int64) error
	DeleteNotification(ctx context.Context, userID, notifID int64) error
	CreateNotification(ctx context.Context, n *Notification) error

	// User Preferences
	UpsertPreference(ctx context.Context, userID int64, key, value string) error
	ListPreferences(ctx context.Context, userID int64) ([]UserPreference, error)

	SoftDeleteUser(ctx context.Context, id int64) error
	DeleteAllNotifications(ctx context.Context, userID int64) error
	DeleteAllPreferences(ctx context.Context, userID int64) error
	DeleteIdentitiesByUserID(ctx context.Context, userID int64) error

	// Identity methods
	CreateIdentity(ctx context.Context, ident *UserIdentity) error
	GetIdentityByProvider(ctx context.Context, provider, providerUserID string) (*UserIdentity, error)
	ListIdentitiesByUserID(ctx context.Context, userID int64) ([]UserIdentity, error)
	DeleteIdentity(ctx context.Context, identityID int64) error
	HasPassword(ctx context.Context, userID int64) (bool, error)
	CreateVerificationToken(ctx context.Context, v *UserVerification) error
	GetVerificationByTokenHash(ctx context.Context, tokenHash string) (*UserVerification, error)
	DeleteVerificationToken(ctx context.Context, id int64) error

	UpdateUserStatus(ctx context.Context, userID int64, status string) error
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) CreateUser(ctx context.Context, u *User) error {
	query := `
		INSERT INTO users (email, password_hash, full_name, is_super_admin, role_id, avatar_url, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, u.Email, u.PasswordHash, u.FullName, u.IsSuperAdmin, u.RoleID, u.AvatarURL, u.Status).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	return err
}

func (r *pgRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.full_name, u.is_super_admin, u.role_id, r.name, u.avatar_url, u.status,
		       u.failed_login_attempts, u.locked_until, u.created_at, u.updated_at, u.deleted_at
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.email = $1 AND u.deleted_at IS NULL`

	var u User
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.IsSuperAdmin, &u.RoleID, &u.RoleName, &u.AvatarURL, &u.Status, &u.FailedLoginAttempts, &u.LockedUntil, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *pgRepository) GetUserByID(ctx context.Context, id int64) (*User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.full_name, u.is_super_admin, u.role_id, r.name, u.avatar_url, u.status,
		       u.failed_login_attempts, u.locked_until, u.created_at, u.updated_at, u.deleted_at
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1 AND u.deleted_at IS NULL`

	var u User
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, id).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.IsSuperAdmin, &u.RoleID, &u.RoleName, &u.AvatarURL, &u.Status, &u.FailedLoginAttempts, &u.LockedUntil, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *pgRepository) UpdateUser(ctx context.Context, u *User) error {
	query := `
		UPDATE users
		SET email = $1, password_hash = $2, full_name = $3, is_super_admin = $4, role_id = $5, avatar_url = $6, status = $7,
		    failed_login_attempts = $8, locked_until = $9, updated_at = CURRENT_TIMESTAMP
		WHERE id = $10`
	q := database.GetQueryer(ctx, r.db)
	result, err := q.ExecContext(ctx, query, u.Email, u.PasswordHash, u.FullName, u.IsSuperAdmin, u.RoleID, u.AvatarURL, u.Status, u.FailedLoginAttempts, u.LockedUntil, u.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (r *pgRepository) LockAccount(ctx context.Context, userID int64, lockedUntil time.Time) error {
	query := `
		UPDATE users
		SET locked_until = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, lockedUntil, userID)
	return err
}

func (r *pgRepository) UpdateSuperAdminStatus(ctx context.Context, email string, isSuper bool) error {
	query := `
		UPDATE users
		SET is_super_admin = $1, updated_at = CURRENT_TIMESTAMP
		WHERE email = $2 AND deleted_at IS NULL`
	q := database.GetQueryer(ctx, r.db)
	result, err := q.ExecContext(ctx, query, isSuper, email)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("user not found")
	}
	return nil
}

func safeLimit(limit int) int {
	if limit < 1 {
		return 1
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

func (r *pgRepository) ListGlobal(ctx context.Context, limit int, cursor int64) ([]User, error) {
	limit = safeLimit(limit)
	query := `
		SELECT u.id, u.email, u.full_name, u.is_super_admin, u.role_id, r.name, u.avatar_url, u.status, u.created_at, u.updated_at
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.deleted_at IS NULL AND ($2 = 0 OR u.id < $2)
		ORDER BY u.id DESC
		LIMIT $1`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, limit, cursor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.IsSuperAdmin, &u.RoleID, &u.RoleName, &u.AvatarURL, &u.Status, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) UpdateUserRole(ctx context.Context, userID int64, roleID *int64) error {
	q := database.GetQueryer(ctx, r.db)
	var result sql.Result
	var err error
	if roleID == nil {
		result, err = q.ExecContext(ctx, `
			UPDATE users
			SET role_id = NULL, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND deleted_at IS NULL`, userID)
	} else {
		result, err = q.ExecContext(ctx, `
			UPDATE users
			SET role_id = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND deleted_at IS NULL
			AND EXISTS (SELECT 1 FROM roles WHERE id = $1)`, *roleID, userID)
	}
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("user or role not found")
	}
	return nil
}

func (r *pgRepository) CreateSession(ctx context.Context, s *Session) error {
	query := `
		INSERT INTO sessions (user_id, token_id, ip_address, user_agent, expires_at, absolute_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, token_id) DO NOTHING
		RETURNING id, created_at`
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, s.UserID, s.TokenID, s.IPAddress, s.UserAgent, s.ExpiresAt, s.AbsoluteExpiresAt).
		Scan(&s.ID, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

func (r *pgRepository) GetSessionByTokenID(ctx context.Context, tokenID string) (*Session, error) {
	query := `
		SELECT s.id, s.user_id, s.token_id, s.ip_address, s.user_agent, s.created_at, s.expires_at, s.absolute_expires_at, s.revoked_at, s.revoked_by
		FROM sessions s
		JOIN users u ON u.id = s.user_id AND u.deleted_at IS NULL
		WHERE s.token_id = $1`
	var s Session
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, tokenID).Scan(
		&s.ID, &s.UserID, &s.TokenID, &s.IPAddress, &s.UserAgent, &s.CreatedAt, &s.ExpiresAt, &s.AbsoluteExpiresAt, &s.RevokedAt, &s.RevokedBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *pgRepository) GetSessionByUserAndToken(ctx context.Context, userID int64, tokenID string) (*Session, error) {
	query := `
		SELECT id, user_id, token_id, ip_address, user_agent, created_at, expires_at, absolute_expires_at, revoked_at, revoked_by
		FROM sessions
		WHERE token_id = $1 AND user_id = $2`
	var s Session
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, tokenID, userID).Scan(
		&s.ID, &s.UserID, &s.TokenID, &s.IPAddress, &s.UserAgent, &s.CreatedAt, &s.ExpiresAt, &s.AbsoluteExpiresAt, &s.RevokedAt, &s.RevokedBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *pgRepository) RevokeSession(ctx context.Context, tokenID string, revokedBy *int64) error {
	query := `
		UPDATE sessions
		SET revoked_at = CURRENT_TIMESTAMP, revoked_by = $1
		WHERE token_id = $2 AND revoked_at IS NULL`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, revokedBy, tokenID)
	return err
}

func (r *pgRepository) RevokeAllSessions(ctx context.Context, userID int64, exceptTokenID string, revokedBy *int64) error {
	query := `
		UPDATE sessions
		SET revoked_at = CURRENT_TIMESTAMP, revoked_by = $1
		WHERE user_id = $2 AND revoked_at IS NULL AND (token_id != $3 OR $3 = '')`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, revokedBy, userID, exceptTokenID)
	return err
}

func (r *pgRepository) ListSessions(ctx context.Context, userID int64) ([]Session, error) {
	query := `
		SELECT id, user_id, token_id, ip_address, user_agent, created_at, expires_at, absolute_expires_at, revoked_at, revoked_by
		FROM sessions
		WHERE user_id = $1
		ORDER BY created_at DESC`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Session
	for rows.Next() {
		var s Session
		err := rows.Scan(
			&s.ID, &s.UserID, &s.TokenID, &s.IPAddress, &s.UserAgent, &s.CreatedAt, &s.ExpiresAt, &s.AbsoluteExpiresAt, &s.RevokedAt, &s.RevokedBy,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) ListGlobalSessions(ctx context.Context, limit int, cursor int64) ([]Session, error) {
	query := `
		SELECT id, user_id, token_id, ip_address, user_agent, created_at, expires_at, absolute_expires_at, revoked_at, revoked_by
		FROM sessions
		WHERE id > $2
		ORDER BY id ASC
		LIMIT $1`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, limit, cursor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Session
	for rows.Next() {
		var s Session
		err := rows.Scan(
			&s.ID, &s.UserID, &s.TokenID, &s.IPAddress, &s.UserAgent, &s.CreatedAt, &s.ExpiresAt, &s.AbsoluteExpiresAt, &s.RevokedAt, &s.RevokedBy,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) CreateIdentity(ctx context.Context, ident *UserIdentity) error {
	query := `
		INSERT INTO user_identities (user_id, provider, provider_user_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx, query, ident.UserID, ident.Provider, ident.ProviderUserID).Scan(&ident.ID, &ident.CreatedAt)
}

func (r *pgRepository) GetIdentityByProvider(ctx context.Context, provider, providerUserID string) (*UserIdentity, error) {
	query := `
		SELECT i.id, i.user_id, i.provider, i.provider_user_id, i.created_at
		FROM user_identities i
		JOIN users u ON u.id = i.user_id AND u.deleted_at IS NULL
		WHERE i.provider = $1 AND i.provider_user_id = $2`
	var ident UserIdentity
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, provider, providerUserID).Scan(
		&ident.ID, &ident.UserID, &ident.Provider, &ident.ProviderUserID, &ident.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &ident, nil
}

func (r *pgRepository) ListIdentitiesByUserID(ctx context.Context, userID int64) ([]UserIdentity, error) {
	query := `SELECT id, user_id, provider, provider_user_id, created_at FROM user_identities WHERE user_id = $1 ORDER BY provider ASC`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []UserIdentity
	for rows.Next() {
		var ident UserIdentity
		if err := rows.Scan(&ident.ID, &ident.UserID, &ident.Provider, &ident.ProviderUserID, &ident.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, ident)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) DeleteIdentity(ctx context.Context, identityID int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, `DELETE FROM user_identities WHERE id = $1`, identityID)
	return err
}

func (r *pgRepository) HasPassword(ctx context.Context, userID int64) (bool, error) {
	q := database.GetQueryer(ctx, r.db)
	var hash string
	err := q.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return hash != "", nil
}

func (r *pgRepository) CreateVerificationToken(ctx context.Context, v *UserVerification) error {
	query := `
		INSERT INTO user_verifications (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx, query, v.UserID, v.TokenHash, v.ExpiresAt).Scan(&v.ID, &v.CreatedAt)
}

func (r *pgRepository) GetVerificationByTokenHash(ctx context.Context, tokenHash string) (*UserVerification, error) {
	query := `
		SELECT v.id, v.user_id, v.token_hash, v.expires_at, v.created_at
		FROM user_verifications v
		JOIN users u ON u.id = v.user_id AND u.deleted_at IS NULL
		WHERE v.token_hash = $1`
	var v UserVerification
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, tokenHash).Scan(
		&v.ID, &v.UserID, &v.TokenHash, &v.ExpiresAt, &v.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

func (r *pgRepository) DeleteVerificationToken(ctx context.Context, id int64) error {
	query := `DELETE FROM user_verifications WHERE id = $1`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, id)
	return err
}

func (r *pgRepository) GetNotificationPreferences(ctx context.Context, userID int64) ([]NotificationPreference, error) {
	query := `
		SELECT user_id, notification_type, channel, enabled, created_at, updated_at
		FROM user_notification_preferences
		WHERE user_id = $1
		ORDER BY notification_type ASC`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []NotificationPreference
	for rows.Next() {
		var p NotificationPreference
		if err := rows.Scan(&p.UserID, &p.NotificationType, &p.Channel, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return prefs, nil
}

func (r *pgRepository) UpsertNotificationPreference(ctx context.Context, userID int64, notificationType string, enabled bool) error {
	query := `
		INSERT INTO user_notification_preferences (user_id, notification_type, channel, enabled)
		VALUES ($1, $2, 'in_app', $3)
		ON CONFLICT (user_id, notification_type, channel)
		DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = CURRENT_TIMESTAMP`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, userID, notificationType, enabled)
	return err
}

func (r *pgRepository) ListNotifications(ctx context.Context, userID int64, limit int, cursor *time.Time, unreadOnly bool) ([]Notification, error) {
	query := `
	SELECT id, user_id, type, title, body, data, is_read, created_at
	FROM notifications
	WHERE user_id = $1`
	args := []interface{}{userID}

	if unreadOnly {
		query += ` AND is_read = FALSE`
	}
	if cursor != nil {
		query += ` AND created_at < $` + strconv.Itoa(len(args)+1)
		args = append(args, *cursor)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Notification
	for rows.Next() {
		var n Notification
		var dataBytes []byte
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &dataBytes, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		n.Data = dataBytes
		list = append(list, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) GetUnreadCount(ctx context.Context, userID int64) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`
	var count int
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *pgRepository) MarkAsRead(ctx context.Context, userID, notifID int64) error {
	query := `UPDATE notifications SET is_read = TRUE WHERE id = $1 AND user_id = $2`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, notifID, userID)
	return err
}

func (r *pgRepository) MarkAllAsRead(ctx context.Context, userID int64) error {
	query := `UPDATE notifications SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, userID)
	return err
}

func (r *pgRepository) DeleteNotification(ctx context.Context, userID, notifID int64) error {
	query := `DELETE FROM notifications WHERE id = $1 AND user_id = $2`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, notifID, userID)
	return err
}

func (r *pgRepository) CreateNotification(ctx context.Context, n *Notification) error {
	query := `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx, query, n.UserID, n.Type, n.Title, n.Body, n.Data).
		Scan(&n.ID, &n.CreatedAt)
}

func (r *pgRepository) SoftDeleteUser(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, `UPDATE users SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

func (r *pgRepository) UpsertPreference(ctx context.Context, userID int64, key, value string) error {
	query := `
		INSERT INTO user_preferences (user_id, key, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, key)
		DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, userID, key, value)
	return err
}

func (r *pgRepository) ListPreferences(ctx context.Context, userID int64) ([]UserPreference, error) {
	query := `SELECT user_id, key, value, updated_at FROM user_preferences WHERE user_id = $1 ORDER BY key ASC`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []UserPreference
	for rows.Next() {
		var p UserPreference
		if err := rows.Scan(&p.UserID, &p.Key, &p.Value, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) DeleteAllNotifications(ctx context.Context, userID int64) error {
	query := `DELETE FROM notifications WHERE user_id = $1`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, userID)
	return err
}

func (r *pgRepository) DeleteAllPreferences(ctx context.Context, userID int64) error {
	query := `DELETE FROM user_preferences WHERE user_id = $1`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, userID)
	return err
}

func (r *pgRepository) DeleteIdentitiesByUserID(ctx context.Context, userID int64) error {
	query := `DELETE FROM user_identities WHERE user_id = $1`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, userID)
	return err
}

func (r *pgRepository) UpdateUserStatus(ctx context.Context, userID int64, status string) error {
	query := `
		UPDATE users
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, status, userID)
	return err
}
