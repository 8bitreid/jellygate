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

// TestInvite_IsValid_ExpiredUTC verifies that an expiry stored as an explicit
// UTC timestamp is correctly identified as expired, regardless of server timezone.
func TestInvite_IsValid_ExpiredUTC(t *testing.T) {
	past := time.Now().UTC().Add(-time.Hour)
	inv := domain.Invite{ExpiresAt: &past}
	if err := inv.IsValid(); err != domain.ErrInviteExpired {
		t.Errorf("expected ErrInviteExpired for past UTC expiry, got: %v", err)
	}
}

// TestInvite_IsValid_NotYetExpiredUTC verifies that a future UTC expiry is valid.
func TestInvite_IsValid_NotYetExpiredUTC(t *testing.T) {
	future := time.Now().UTC().Add(time.Hour)
	inv := domain.Invite{ExpiresAt: &future}
	if err := inv.IsValid(); err != nil {
		t.Errorf("invite with future UTC expiry should be valid, got: %v", err)
	}
}

// TestInvite_IsValid_CrossTimezone simulates the CA-vs-MA scenario: an expiry
// set as a UTC instant must compare correctly against time.Now() regardless of
// what local timezone the server process runs in.
func TestInvite_IsValid_CrossTimezone(t *testing.T) {
	// Simulate an expiry that was 30 minutes ago in UTC.
	expiredUTC := time.Now().UTC().Add(-30 * time.Minute)
	inv := domain.Invite{ExpiresAt: &expiredUTC}
	if err := inv.IsValid(); err != domain.ErrInviteExpired {
		t.Errorf("invite expired 30m ago (UTC) should be expired, got: %v", err)
	}

	// Simulate an expiry 30 minutes from now in UTC.
	validUTC := time.Now().UTC().Add(30 * time.Minute)
	inv2 := domain.Invite{ExpiresAt: &validUTC}
	if err := inv2.IsValid(); err != nil {
		t.Errorf("invite expiring in 30m (UTC) should be valid, got: %v", err)
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
