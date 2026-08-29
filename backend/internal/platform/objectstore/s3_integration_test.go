//go:build integration

package objectstore

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"repforge.local/backend/internal/platform/config"
)

func TestS3PutGetDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("refusing integration object operation: %v", err)
	}
	store, err := NewS3(ctx, Options{
		Endpoint: cfg.S3Endpoint, Region: cfg.S3Region,
		AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey,
		Bucket: cfg.S3Bucket, ForcePathStyle: cfg.S3ForcePathStyle,
	})
	if err != nil {
		t.Fatal(err)
	}
	key := "integration/" + uuid.NewString()
	if err := store.Put(ctx, key, strings.NewReader("synthetic-object")); err != nil {
		t.Fatalf("put: %v", err)
	}
	body, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	value, err := io.ReadAll(body)
	body.Close()
	if err != nil || string(value) != "synthetic-object" {
		t.Fatalf("read: value=%q error=%v", value, err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
