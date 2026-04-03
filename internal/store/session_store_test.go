package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/internal/store"
)

func TestSessionStore_CreateAndGet(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewSessionStore(pool)
	ctx := context.Background()

	sess := domain.Session{
		Token:     "sess-token-1",
		Username:  "admin",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.Create(ctx, sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Get(ctx, sess.Token)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Username != sess.Username {
		t.Errorf("username: want %q, got %q", sess.Username, got.Username)
	}
}

func TestSessionStore_Get_NotFound(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewSessionStore(pool)
	ctx := context.Background()

	_, err := s.Get(ctx, "does-not-exist")
	if err != domain.ErrSessionNotFound {
		t.Errorf("want ErrSessionNotFound, got %v", err)
	}
}

func TestSessionStore_Get_Expired(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewSessionStore(pool)
	ctx := context.Background()

	sess := domain.Session{
		Token:     "sess-expired",
		Username:  "admin",
		ExpiresAt: time.Now().Add(-time.Minute), // already expired
	}
	_ = s.Create(ctx, sess)

	_, err := s.Get(ctx, sess.Token)
	if err != domain.ErrSessionExpired {
		t.Errorf("want ErrSessionExpired, got %v", err)
	}
}

func TestSessionStore_Delete(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewSessionStore(pool)
	ctx := context.Background()

	sess := domain.Session{
		Token:     "sess-delete",
		Username:  "admin",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	_ = s.Create(ctx, sess)

	if err := s.Delete(ctx, sess.Token); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := s.Get(ctx, sess.Token)
	if err != domain.ErrSessionNotFound {
		t.Errorf("want ErrSessionNotFound after delete, got %v", err)
	}
}

func TestSessionStore_Purge(t *testing.T) {
	pool := newTestDB(t)
	s := store.NewSessionStore(pool)
	ctx := context.Background()

	_ = s.Create(ctx, domain.Session{Token: "active", Username: "u", ExpiresAt: time.Now().Add(time.Hour)})
	_ = s.Create(ctx, domain.Session{Token: "expired1", Username: "u", ExpiresAt: time.Now().Add(-time.Hour)})
	_ = s.Create(ctx, domain.Session{Token: "expired2", Username: "u", ExpiresAt: time.Now().Add(-time.Minute)})

	if err := s.Purge(ctx); err != nil {
		t.Fatalf("Purge: %v", err)
	}

	// active session survives
	if _, err := s.Get(ctx, "active"); err != nil {
		t.Errorf("active session should survive purge, got %v", err)
	}
	// expired sessions are gone
	for _, tok := range []string{"expired1", "expired2"} {
		if _, err := s.Get(ctx, tok); err != domain.ErrSessionNotFound {
			t.Errorf("token %q: want ErrSessionNotFound after purge, got %v", tok, err)
		}
	}
}
