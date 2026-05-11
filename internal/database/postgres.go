package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	URL      string
	MaxConns int32
	MinConns int32
}

func Connect(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(config.URL)
	if err != nil {
		return nil, err
	}

	cfg.MaxConns = config.MaxConns
	cfg.MinConns = config.MinConns
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
