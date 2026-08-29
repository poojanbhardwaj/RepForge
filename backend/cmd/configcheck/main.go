package main

import (
	"fmt"
	"os"

	"repforge.local/backend/internal/platform/config"
)

func main() {
	if _, err := config.Load(); err != nil {
		fmt.Fprintln(os.Stderr, "repforge-configcheck:", err)
		os.Exit(1)
	}
	fmt.Println("Local dependency endpoints and HTTP bind are loopback-only.")
}
