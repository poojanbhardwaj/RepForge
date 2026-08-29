package database

import (
	"context"
	"testing"

	platformconfig "repforge.local/backend/internal/platform/config"
)

func TestOpenRejectsRemoteConfigurationBeforeConnecting(t *testing.T) {
	if _, err := Open(context.Background(), "postgres://192.0.2.10/repforge?sslmode=disable"); err == nil {
		t.Fatal("expected remote database rejection")
	}
}

func TestOpenConfigRevalidatesCopiedConfiguration(t *testing.T) {
	connectionConfig, err := platformconfig.ParseDatabaseConfig(
		"postgres://127.0.0.1/repforge?sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}
	connectionConfig.Host = "198.51.100.20"
	if _, err := OpenConfig(context.Background(), connectionConfig); err == nil {
		t.Fatal("expected modified remote database configuration rejection")
	}
}
