package migrations

import (
	"testing"

	platformconfig "repforge.local/backend/internal/platform/config"
)

func TestRunRejectsRemoteConfigurationBeforeMigration(t *testing.T) {
	if err := Run("postgres://192.0.2.10/repforge?sslmode=disable", t.TempDir(), "up"); err == nil {
		t.Fatal("expected remote migration database rejection")
	}
}

func TestRunConfigRevalidatesCopiedConfiguration(t *testing.T) {
	connectionConfig, err := platformconfig.ParseDatabaseConfig(
		"host=127.0.0.1,127.0.0.2 dbname=repforge sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}
	connectionConfig.Fallbacks[0].Host = "203.0.113.20"
	if err := RunConfig(connectionConfig, t.TempDir(), "up"); err == nil {
		t.Fatal("expected modified remote fallback rejection")
	}
}
