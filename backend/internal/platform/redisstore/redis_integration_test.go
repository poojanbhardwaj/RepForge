//go:build integration

package redisstore

import (
	"context"
	"testing"
	"time"

	"repforge.local/backend/internal/platform/config"
)

func TestRedisPing(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("refusing integration Redis operation: %v", err)
	}
	client, err := New(cfg.RedisURL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("redis ping: %v", err)
	}
}
