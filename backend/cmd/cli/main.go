package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/tevoworks/corekit/backend/internal/config"
	"github.com/tevoworks/corekit/backend/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/cli/ <command> [args]")
		fmt.Println("Commands:")
		fmt.Println("  create-admin <email> <password>  - Create a new super admin user")
		fmt.Println("  make-admin <email>                - Promote existing user to super_admin")
		os.Exit(1)
	}

	cmd := os.Args[1]

	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	switch cmd {
	case "create-admin":
		if len(os.Args) < 4 {
			slog.Error("insufficient arguments for create-admin")
			fmt.Println("Usage: create-admin <email> <password>")
			os.Exit(1)
		}
		email := os.Args[2]
		password := os.Args[3]
		createAdmin(db, email, password)
	case "make-admin":
		if len(os.Args) < 3 {
			slog.Error("insufficient arguments for make-admin")
			fmt.Println("Usage: make-admin <email>")
			os.Exit(1)
		}
		email := os.Args[2]
		makeAdmin(db, email)
	default:
		slog.Error("unknown command", "command", cmd)
		os.Exit(1)
	}
}

func createAdmin(db *sql.DB, email, password string) {
	var roleID int64
	err := db.QueryRow(`SELECT id FROM roles WHERE name = 'super_admin'`).Scan(&roleID)
	if err != nil {
		slog.Error("failed to find super_admin role", "error", err)
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		os.Exit(1)
	}

	var id int64
	err = db.QueryRow(`
		INSERT INTO users (email, password_hash, full_name, is_super_admin, role_id, status)
		VALUES ($1, $2, $3, true, $4, 'ACTIVE')
		RETURNING id`,
		email, string(hash), "Admin", roleID,
	).Scan(&id)
	if err != nil {
		slog.Error("failed to create user", "email", email, "error", err)
		os.Exit(1)
	}

	fmt.Printf("Super admin created: %s (id=%d)\n", email, id)
}

func makeAdmin(db *sql.DB, email string) {
	var roleID int64
	err := db.QueryRow(`SELECT id FROM roles WHERE name = 'super_admin'`).Scan(&roleID)
	if err != nil {
		slog.Error("failed to find super_admin role", "error", err)
		os.Exit(1)
	}

	result, err := db.Exec(`UPDATE users SET role_id = $1, is_super_admin = true WHERE email = $2`, roleID, email)
	if err != nil {
		slog.Error("failed to update user role", "email", email, "error", err)
		os.Exit(1)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		slog.Error("user not found", "email", email)
		os.Exit(1)
	}

	fmt.Printf("User %s is now a super_admin\n", email)
}
