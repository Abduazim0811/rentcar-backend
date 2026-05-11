package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"car-rental-system/internal/auth"
	"car-rental-system/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.Load()
	adminEmail := strings.ToLower(strings.TrimSpace(getEnv("SUPER_ADMIN_EMAIL", "admin@rentcar.local")))
	adminName := strings.TrimSpace(getEnv("SUPER_ADMIN_NAME", "Super Admin"))
	adminPassword := os.Getenv("SUPER_ADMIN_PASSWORD")
	if adminPassword == "" {
		logger.Error("seed_failed", slog.String("error", "SUPER_ADMIN_PASSWORD is required"))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("seed_db_open_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	hash, err := auth.HashPassword(adminPassword)
	if err != nil {
		logger.Error("seed_password_hash_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var id int64
	err = db.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role, email_verified_at)
		VALUES ($1, $2, $3, 'super_admin', NOW())
		ON CONFLICT (email) DO UPDATE
		SET name = EXCLUDED.name,
		    role = 'super_admin',
		    email_verified_at = COALESCE(users.email_verified_at, NOW()),
		    updated_at = NOW()
		RETURNING id
	`, adminName, adminEmail, hash).Scan(&id)
	if err != nil {
		logger.Error("seed_super_admin_failed", slog.String("email", adminEmail), slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("seed_super_admin_completed", slog.Int64("user_id", id), slog.String("email", adminEmail))
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
