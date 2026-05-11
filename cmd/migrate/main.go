package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"os"
	"time"

	"car-rental-system/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	dir := flag.String("dir", "migrations", "migration directory")
	flag.Parse()

	args := flag.Args()
	command := "up"
	arguments := []string{}
	if len(args) > 0 {
		command = args[0]
		arguments = args[1:]
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("migration_db_open_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		logger.Error("migration_db_ping_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		logger.Error("migration_dialect_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if err := goose.RunContext(ctx, command, db, *dir, arguments...); err != nil {
		logger.Error("migration_failed", slog.String("command", command), slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("migration_completed", slog.String("command", command))
}
