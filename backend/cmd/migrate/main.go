package main

import (
	"fmt"
	"os"
	"path/filepath"

	"repforge.local/backend/internal/platform/config"
	"repforge.local/backend/internal/platform/migrations"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/migrate <up|down|status>")
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "repforge-migrate: configuration:", err)
		os.Exit(1)
	}
	directory, err := filepath.Abs("migrations")
	if err != nil {
		fmt.Fprintln(os.Stderr, "repforge-migrate: migration path:", err)
		os.Exit(1)
	}
	if err := migrations.Run(cfg.DatabaseURL, directory, os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "repforge-migrate:", err)
		os.Exit(1)
	}
}
