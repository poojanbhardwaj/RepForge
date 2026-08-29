package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	platformconfig "repforge.local/backend/internal/platform/config"
)

func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := platformconfig.ParseDatabaseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database configuration: %w", err)
	}
	return OpenConfig(ctx, config)
}

func OpenConfig(ctx context.Context, connectionConfig *pgx.ConnConfig) (*pgxpool.Pool, error) {
	if err := platformconfig.RevalidateDatabaseConfig(connectionConfig); err != nil {
		return nil, fmt.Errorf("validate database configuration: %w", err)
	}
	poolConfig, err := pgxpool.ParseConfig(connectionConfig.ConnString())
	if err != nil {
		return nil, fmt.Errorf("parse database pool configuration: %w", err)
	}
	poolConfig.ConnConfig = connectionConfig.Copy()
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 30 * time.Second
	poolConfig.BeforeConnect = func(_ context.Context, candidate *pgx.ConnConfig) error {
		return platformconfig.RevalidateDatabaseConfig(candidate)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	return pool, nil
}
