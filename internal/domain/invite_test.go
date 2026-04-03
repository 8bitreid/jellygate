package domain_test

import (
	"testing"
	"time"

	"github.com/rmewborne/jellygate/internal/domain"
)

func TestInvite_IsValid_Fresh(t *testing.T) {
	inv := domain.Invite{Label: "test"}
	if err := inv.IsValid(); err != nil {
		t.Errorf("fresh invite should be valid, got: %v", err)
	}
}

func TestInvite_IsValid_Revoked(t *testing.T) {
	inv := domain.Invite{Revoked: true}
	if err := inv.IsValid(); err != domain.ErrInviteRevoked {
		t.Errorf("expected ErrInviteRevoked, got: %v", err)
	}
}

func TestInvite_IsValid_Expired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	inv := domain.Invite{ExpiresAt: &past}
	if err := inv.IsValid(); err != domain.ErrInviteExpired {
		t.Errorf("expected ErrInviteExpired, got: %v", err)
	}
}

func TestInvite_IsValid_NotYetExpired(t *testing.T) {
	future := time.Now().Add(time.Hour)
	inv := domain.Invite{ExpiresAt: &future}
	if err := inv.IsValid(); err != nil {
		t.Errorf("invite with future expiry should be valid, got: %v", err)
	}
}

func TestInvite_IsValid_Exhausted(t *testing.T) {
	max := 3
	inv := domain.Invite{MaxUses: &max, UseCount: 3}
	if err := inv.IsValid(); err != domain.ErrInviteExhausted {
		t.Errorf("expected ErrInviteExhausted, got: %v", err)
	}
}

func TestInvite_IsValid_NotYetExhausted(t *testing.T) {
	max := 3
	inv := domain.Invite{MaxUses: &max, UseCount: 2}
	if err := inv.IsValid(); err != nil {
		t.Errorf("invite with uses remaining should be valid, got: %v", err)
	}
}

func TestSession_IsExpired(t *testing.T) {
	s := domain.Session{ExpiresAt: time.Now().Add(-time.Second)}
	if !s.IsExpired() {
		t.Error("expected session to be expired")
	}
}

func TestSession_IsNotExpired(t *testing.T) {
	s := domain.Session{ExpiresAt: time.Now().Add(time.Hour)}
	if s.IsExpired() {
		t.Error("expected session to not be expired")
	}
}
