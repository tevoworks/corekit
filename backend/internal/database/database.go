package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
)

var ErrNotFound = errors.New("resource not found")

type txKey struct{}
type auditActionKey struct{}
type ctxUserIDKey struct{}
type impersonatorCtxKey struct{}

func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	slog.Info("Database connection established successfully")
	return db, nil
}

// WithTx returns a context with the transaction.
func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// WithAuditAction attaches a business action name (e.g. "CREATE_ROLE") to the context.
// This is read by RunInTransaction and passed to the audit trigger via set_config().
// If not set, the trigger uses OPERATION_TABLE as fallback (e.g. "INSERT_ROLES").
func WithAuditAction(ctx context.Context, action string) context.Context {
	return context.WithValue(ctx, auditActionKey{}, action)
}

// GetAuditAction extracts the action name from context.
func GetAuditAction(ctx context.Context) string {
	if a, ok := ctx.Value(auditActionKey{}).(string); ok {
		return a
	}
	return ""
}

// WithCtxUserID attaches a user ID to the context. Used by middleware to propagate
// actor_id to the database layer for audit trigger consumption.
// Also accepts nil to handle system/CLI contexts where no user is available.
func WithCtxUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, ctxUserIDKey{}, id)
}

// WithAuditCtx is a convenience helper that sets both the user ID (actor) and
// action name on the context for audit trigger propagation.
// Call this before RunInTransaction to ensure the DB trigger captures rich audit data.
//
// Usage:
//
//	ctx = database.WithAuditCtx(ctx, actorID, "CREATE_ROLE")
//	database.RunInTransaction(ctx, db, func(txCtx) error {
//	    return repo.Create(txCtx, m)
//	})
func WithAuditCtx(ctx context.Context, actorID int64, action string) context.Context {
	return WithAuditAction(WithCtxUserID(ctx, actorID), action)
}

// GetCtxUserID extracts the user ID from context.
func GetCtxUserID(ctx context.Context) *int64 {
	if id, ok := ctx.Value(ctxUserIDKey{}).(int64); ok && id > 0 {
		return &id
	}
	return nil
}

// GetTx extracts the transaction from context.
func GetTx(ctx context.Context) *sql.Tx {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return nil
}

type Queryer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func WithImpersonatorID(ctx context.Context, id *int64) context.Context {
	return context.WithValue(ctx, impersonatorCtxKey{}, id)
}

func GetImpersonatorIDFromCtx(ctx context.Context) *int64 {
	if id, ok := ctx.Value(impersonatorCtxKey{}).(*int64); ok {
		return id
	}
	return nil
}

func GetQueryer(ctx context.Context, db *sql.DB) Queryer {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return db
}

// setAuditConfig runs set_config() on the transaction so the PostgreSQL audit trigger
// can read actor_id, impersonator_id, and action from the session. This is called
// at the start of every RunInTransaction, making audit automatic for all mutations.
func setAuditConfig(ctx context.Context, tx *sql.Tx, actorID *int64, impersonatorID *int64, action string) {
	if actorID != nil {
		_, _ = tx.ExecContext(ctx, "SELECT set_config('app.actor_id', $1, true)", fmt.Sprintf("%d", *actorID))
	}
	if impersonatorID != nil {
		_, _ = tx.ExecContext(ctx, "SELECT set_config('app.impersonator_id', $1, true)", fmt.Sprintf("%d", *impersonatorID))
	}
	if action != "" {
		_, _ = tx.ExecContext(ctx, "SELECT set_config('app.action', $1, true)", action)
	}
}

// RunInTransaction wraps operations in a database transaction and handles rollback/commit.
// It automatically propagates audit context (actor_id, action) to the PostgreSQL session
// so the DB audit trigger captures every mutation.
func RunInTransaction(ctx context.Context, db *sql.DB, fn func(txCtx context.Context) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			slog.Error("panic recovered in transaction", "panic", p)
			err = fmt.Errorf("panic recovered in transaction: %v", p)
		}
	}()

	txCtx := WithTx(ctx, tx)

	// Propagate audit context to the DB session before executing user code.
	// Every mutation is now automatically captured by DB audit triggers.
	// If impersonation is active (impersonatorID present), both actor_id (target user)
	// and impersonator_id (real admin) are set so the audit trigger can record both.
	setAuditConfig(txCtx, tx, GetCtxUserID(ctx), GetImpersonatorIDFromCtx(ctx), GetAuditAction(ctx))

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
