package domain

import "context"

// InviteStore persists and retrieves invites.
type InviteStore interface {
	Create(ctx context.Context, inv Invite) error
	GetByToken(ctx context.Context, token string) (Invite, error)
	List(ctx context.Context) ([]Invite, error)
	Revoke(ctx context.Context, id string) error
	IncrementUse(ctx context.Context, id string) error
}

// SessionStore persists and retrieves admin sessions.
type SessionStore interface {
	Create(ctx context.Context, s Session) error
	Get(ctx context.Context, token string) (Session, error)
	Delete(ctx context.Context, token string) error
	Purge(ctx context.Context) error // removes all expired sessions
}

// JellyfinClient communicates with a Jellyfin server.
type JellyfinClient interface {
	Authenticate(ctx context.Context, username, password string) (accessToken string, err error)
	ListLibraries(ctx context.Context, adminToken string) ([]Library, error)
	CreateUser(ctx context.Context, adminToken, username, password string) (userID string, err error)
	SetLibraryAccess(ctx context.Context, adminToken, userID string, libraryIDs []string) error
}

// Notifier delivers notifications about invite lifecycle events.
type Notifier interface {
	InviteCreated(ctx context.Context, inv Invite) error
	InviteUsed(ctx context.Context, inv Invite, username string) error
}

// RegistrationStore records Jellyfin accounts created via invites.
type RegistrationStore interface {
	Create(ctx context.Context, reg Registration) error
	CountByInviteID(ctx context.Context, inviteID string) (int, error)
}

// SettingsStore persists application-level configuration.
type SettingsStore interface {
	GetJellyfinAdminToken(ctx context.Context) (string, error)
	SetJellyfinAdminToken(ctx context.Context, token string) error
}
