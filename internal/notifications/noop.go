package notifications

import (
	"context"

	"github.com/rmewborne/jellygate/internal/domain"
)

// NoopNotifier satisfies domain.Notifier and silently discards all events.
// Used when DISCORD_WEBHOOK_URL is unset.
type NoopNotifier struct{}

func (n *NoopNotifier) InviteCreated(_ context.Context, _ domain.Invite) error       { return nil }
func (n *NoopNotifier) InviteUsed(_ context.Context, _ domain.Invite, _ string) error { return nil }
