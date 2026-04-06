package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/internal/store"
)

func TestInviteStore_CreateAndGetByToken(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewInviteStore(pool)
	ctx := context.Background()

	inv := domain.Invite{
		ID:        uuid.NewString(),
		Token:     "tok-abc123",
		Label:     "test invite",
		CreatedBy: "admin",
	}

	if err := s.Create(ctx, inv); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.GetByToken(ctx, inv.Token)
	if err != nil {
		t.Fatalf("GetByToken: %v", err)
	}
	if got.Label != inv.Label {
		t.Errorf("label: want %q, got %q", inv.Label, got.Label)
	}
	if got.UseCount != 0 {
		t.Errorf("use_count: want 0, got %d", got.UseCount)
	}
}

func TestInviteStore_GetByToken_NotFound(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewInviteStore(pool)
	ctx := context.Background()

	_, err := s.GetByToken(ctx, "nonexistent")
	if err != domain.ErrInviteNotFound {
		t.Errorf("want ErrInviteNotFound, got %v", err)
	}
}

func TestInviteStore_List(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewInviteStore(pool)
	ctx := context.Background()

	for i, label := range []string{"first", "second"} {
		_ = s.Create(ctx, domain.Invite{
			ID:        uuid.NewString(),
			Token:     "tok-list-" + label,
			Label:     label,
			CreatedBy: "admin",
			// stagger created_at ordering
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	invites, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(invites) != 2 {
		t.Fatalf("want 2 invites, got %d", len(invites))
	}
}

func TestInviteStore_Revoke(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewInviteStore(pool)
	ctx := context.Background()

	id := uuid.NewString()
	_ = s.Create(ctx, domain.Invite{ID: id, Token: "tok-revoke", Label: "r", CreatedBy: "admin"})

	if err := s.Revoke(ctx, id); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	got, err := s.GetByToken(ctx, "tok-revoke")
	if err != nil {
		t.Fatalf("GetByToken after revoke: %v", err)
	}
	if !got.Revoked {
		t.Error("want Revoked=true")
	}
}

func TestInviteStore_Revoke_NotFound(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewInviteStore(pool)
	ctx := context.Background()

	err := s.Revoke(ctx, uuid.NewString())
	if err != domain.ErrInviteNotFound {
		t.Errorf("want ErrInviteNotFound, got %v", err)
	}
}

func TestInviteStore_IncrementUse(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewInviteStore(pool)
	ctx := context.Background()

	id := uuid.NewString()
	_ = s.Create(ctx, domain.Invite{ID: id, Token: "tok-inc", Label: "i", CreatedBy: "admin"})

	if err := s.IncrementUse(ctx, id); err != nil {
		t.Fatalf("IncrementUse: %v", err)
	}
	if err := s.IncrementUse(ctx, id); err != nil {
		t.Fatalf("IncrementUse (2nd): %v", err)
	}

	got, _ := s.GetByToken(ctx, "tok-inc")
	if got.UseCount != 2 {
		t.Errorf("want UseCount=2, got %d", got.UseCount)
	}
}

func TestInviteStore_WithLibraryIDs(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewInviteStore(pool)
	ctx := context.Background()

	libs := []string{"lib-movies", "lib-tv"}
	id := uuid.NewString()
	_ = s.Create(ctx, domain.Invite{
		ID: id, Token: "tok-libs", Label: "l", CreatedBy: "admin",
		LibraryIDs: libs,
	})

	got, _ := s.GetByToken(ctx, "tok-libs")
	if len(got.LibraryIDs) != 2 || got.LibraryIDs[0] != libs[0] {
		t.Errorf("library_ids round-trip failed: got %v", got.LibraryIDs)
	}
}

func TestInviteStore_GroupLibraries(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewInviteStore(pool)
	ctx := context.Background()

	id := uuid.NewString()
	_ = s.Create(ctx, domain.Invite{
		ID: id, Token: "tok-grp", Label: "g", CreatedBy: "admin",
		LibraryIDs:     []string{"lib-movies", "lib-tv"},
		GroupLibraries: true,
	})

	got, err := s.GetByToken(ctx, "tok-grp")
	if err != nil {
		t.Fatalf("GetByToken: %v", err)
	}
	if !got.GroupLibraries {
		t.Error("want GroupLibraries=true, got false")
	}

	// Verify the default is false when not set.
	id2 := uuid.NewString()
	_ = s.Create(ctx, domain.Invite{
		ID: id2, Token: "tok-nogrp", Label: "ng", CreatedBy: "admin",
	})
	got2, err := s.GetByToken(ctx, "tok-nogrp")
	if err != nil {
		t.Fatalf("GetByToken: %v", err)
	}
	if got2.GroupLibraries {
		t.Error("want GroupLibraries=false by default, got true")
	}
}
