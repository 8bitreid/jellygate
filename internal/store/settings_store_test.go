package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/internal/store"
)

func TestSettingsStore_GetJellyfinAdminToken_NotFound(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewSettingsStore(pool)
	ctx := context.Background()

	_, err := s.GetJellyfinAdminToken(ctx)
	if !errors.Is(err, domain.ErrSettingNotFound) {
		t.Errorf("want ErrSettingNotFound, got %v", err)
	}
}

func TestSettingsStore_SetAndGetJellyfinAdminToken(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewSettingsStore(pool)
	ctx := context.Background()

	const token = "test-admin-token"
	if err := s.SetJellyfinAdminToken(ctx, token); err != nil {
		t.Fatalf("SetJellyfinAdminToken: %v", err)
	}

	got, err := s.GetJellyfinAdminToken(ctx)
	if err != nil {
		t.Fatalf("GetJellyfinAdminToken: %v", err)
	}
	if got != token {
		t.Errorf("token: want %q, got %q", token, got)
	}
}

func TestSettingsStore_SetJellyfinAdminToken_Upsert(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewSettingsStore(pool)
	ctx := context.Background()

	if err := s.SetJellyfinAdminToken(ctx, "first-token"); err != nil {
		t.Fatalf("SetJellyfinAdminToken (first): %v", err)
	}
	if err := s.SetJellyfinAdminToken(ctx, "second-token"); err != nil {
		t.Fatalf("SetJellyfinAdminToken (second): %v", err)
	}

	got, err := s.GetJellyfinAdminToken(ctx)
	if err != nil {
		t.Fatalf("GetJellyfinAdminToken: %v", err)
	}
	if got != "second-token" {
		t.Errorf("upsert: want %q, got %q", "second-token", got)
	}
}
