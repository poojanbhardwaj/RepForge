package migrations

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	platformconfig "repforge.local/backend/internal/platform/config"
)

func Run(databaseURL, directory, command string) error {
	connectionConfig, err := platformconfig.ParseDatabaseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("validate migration database: %w", err)
	}
	return RunConfig(connectionConfig, directory, command)
}

func RunConfig(connectionConfig *pgx.ConnConfig, directory, command string) error {
	if err := platformconfig.RevalidateDatabaseConfig(connectionConfig); err != nil {
		return fmt.Errorf("validate migration database: %w", err)
	}
	db := stdlib.OpenDB(*connectionConfig.Copy())
	defer db.Close()
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}
	var err error
	switch command {
	case "up":
		err = goose.Up(db, directory)
	case "down":
		err = goose.Down(db, directory)
	case "status":
		err = goose.Status(db, directory)
	default:
		return fmt.Errorf("unsupported migration command %q", command)
	}
	if err != nil {
		return fmt.Errorf("migration %s: %w", command, err)
	}
	return nil
}
