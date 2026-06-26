package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func RunMigrations(db *sql.DB, migrationsDir string) error {
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations directory %s: %w", migrationsDir, err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, ".down.sql") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	for _, f := range files {
		var applied bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)", f).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", f, err)
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", f, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", f, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (filename) VALUES ($1)", f); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", f, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", f, err)
		}

		slog.Info("Migration applied", "file", f)
	}

	return nil
}

func RunMigrationsDown(db *sql.DB, migrationsDir string, n int) error {
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	rows, err := db.Query("SELECT filename FROM schema_migrations ORDER BY applied_at DESC")
	if err != nil {
		return fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	var applied []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return fmt.Errorf("scan migration: %w", err)
		}
		applied = append(applied, f)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if n > 0 && n < len(applied) {
		applied = applied[:n]
	}

	if len(applied) == 0 {
		slog.Info("No migrations to roll back")
		return nil
	}

	for _, f := range applied {
		downFile := migrationBase(f) + ".down.sql"
		downPath := filepath.Join(migrationsDir, downFile)

		content, err := os.ReadFile(downPath)
		if err != nil {
			return fmt.Errorf("read down migration %s (for %s): %w", downFile, f, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", downFile, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute down migration %s: %w", downFile, err)
		}

		if _, err := tx.Exec("DELETE FROM schema_migrations WHERE filename = $1", f); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record rollback %s: %w", f, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit rollback %s: %w", f, err)
		}

		slog.Info("Migration rolled back", "file", f, "down", downFile)
	}

	return nil
}

func ensureSchemaMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

func migrationBase(name string) string {
	name = strings.TrimSuffix(name, ".sql")
	name = strings.TrimSuffix(name, ".up")
	return name
}
