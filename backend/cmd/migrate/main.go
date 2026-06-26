package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/tevoworks/corekit/backend/internal/config"
	"github.com/tevoworks/corekit/backend/internal/database"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/migrate/ <command> [args]")
		fmt.Println("")
		fmt.Println("Commands:")
		fmt.Println("  up           Run all pending migrations")
		fmt.Println("  down         Roll back the last migration")
		fmt.Println("  down <N>     Roll back N migrations")
		fmt.Println("  reset        Roll back all migrations, then re-apply")
		os.Exit(1)
	}

	cmd := os.Args[1]

	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	migrationsDir := "migrations/"

	switch cmd {
	case "up":
		if err := database.RunMigrations(db, migrationsDir); err != nil {
			slog.Error("Migration up failed", "error", err)
			os.Exit(1)
		}
		slog.Info("All pending migrations applied")

	case "down":
		n := 1
		if len(os.Args) > 2 {
			n, err = strconv.Atoi(os.Args[2])
			if err != nil {
				slog.Error("Invalid number, expected integer", "arg", os.Args[2])
				os.Exit(1)
			}
		}
		if err := database.RunMigrationsDown(db, migrationsDir, n); err != nil {
			slog.Error("Migration down failed", "error", err)
			os.Exit(1)
		}

	case "reset":
		if err := database.RunMigrationsDown(db, migrationsDir, -1); err != nil {
			slog.Error("Migration rollback failed", "error", err)
			os.Exit(1)
		}
		if err := database.RunMigrations(db, migrationsDir); err != nil {
			slog.Error("Migration up failed", "error", err)
			os.Exit(1)
		}
		slog.Info("All migrations reset and re-applied")

	default:
		fmt.Printf("Unknown command: %s\n\n", cmd)
		fmt.Println("Commands:")
		fmt.Println("  up           Run all pending migrations")
		fmt.Println("  down         Roll back the last migration")
		fmt.Println("  down <N>     Roll back N migrations")
		fmt.Println("  reset        Roll back all migrations, then re-apply")
		os.Exit(1)
	}
}
