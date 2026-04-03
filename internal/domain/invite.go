package domain

import "time"

// Invite is a single-use or multi-use registration link.
type Invite struct {
	ID         string
	Token      string
	Label      string
	CreatedBy  string
	CreatedAt  time.Time
	ExpiresAt  *time.Time // nil = never expires
	MaxUses    *int       // nil = unlimited
	UseCount   int
	LibraryIDs []string
	Revoked    bool
}

// IsValid reports whether the invite can still be used.
func (inv *Invite) IsValid() error {
	if inv.Revoked {
		return ErrInviteRevoked
	}
	if inv.ExpiresAt != nil && time.Now().After(*inv.ExpiresAt) {
		return ErrInviteExpired
	}
	if inv.MaxUses != nil && inv.UseCount >= *inv.MaxUses {
		return ErrInviteExhausted
	}
	return nil
}

// Registration records a user account created via an invite.
type Registration struct {
	ID           string
	InviteID     string
	JellyfinUID  string
	Username     string
	RegisteredAt time.Time
}

// Library is a Jellyfin virtual folder / library.
type Library struct {
	ID   string
	Name string
}
