package main

import (
	"log/slog"
	"os"

	"car-rental-system/internal/app"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := app.Run(logger); err != nil {
		logger.Error("app_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
