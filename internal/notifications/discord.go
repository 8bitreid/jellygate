package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/gtuk/discordwebhook"

	"github.com/rmewborne/jellygate/internal/domain"
)

const botName = "jellygate"

// DiscordNotifier sends invite lifecycle events to a Discord webhook.
//
// Embed color is intentionally omitted: the Discord API requires color as a
// decimal integer but gtuk/discordwebhook models it as *string, which
// JSON-encodes as a quoted string and is silently ignored by Discord.
//
// Wiring: cmd/server/main.go selects DiscordNotifier when DISCORD_WEBHOOK_URL
// is set, and falls back to notifications.NoopNotifier when it is not.
type DiscordNotifier struct {
	webhookURL string
}

// NewDiscordNotifier creates a DiscordNotifier targeting the given webhook URL.
func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{webhookURL: webhookURL}
}

// InviteCreated fires when an admin creates a new invite.
// The context is accepted to satisfy the domain.Notifier interface; the
// underlying HTTP client does not support cancellation.
func (n *DiscordNotifier) InviteCreated(_ context.Context, inv domain.Invite) error {
	inline := true
	fields := []discordwebhook.Field{
		{Name: strPtr("label"), Value: strPtr(inv.Label), Inline: &inline},
		{Name: strPtr("created by"), Value: strPtr(inv.CreatedBy), Inline: &inline},
		{Name: strPtr("expires"), Value: strPtr(fmtExpiry(inv.ExpiresAt)), Inline: &inline},
		{Name: strPtr("max uses"), Value: strPtr(fmtMaxUses(inv.MaxUses)), Inline: &inline},
	}
	msg := discordwebhook.Message{
		Username: strPtr(botName),
		Embeds: &[]discordwebhook.Embed{{
			Title:  strPtr("invite created"),
			Fields: &fields,
		}},
	}
	if err := discordwebhook.SendMessage(n.webhookURL, msg); err != nil {
		return fmt.Errorf("notifications.DiscordNotifier.InviteCreated: %w", err)
	}
	return nil
}

// InviteUsed fires when a user registers via an invite.
func (n *DiscordNotifier) InviteUsed(_ context.Context, inv domain.Invite, username string) error {
	inline := true
	fields := []discordwebhook.Field{
		{Name: strPtr("new user"), Value: strPtr(username), Inline: &inline},
		{Name: strPtr("invite"), Value: strPtr(inv.Label), Inline: &inline},
		{Name: strPtr("uses"), Value: strPtr(fmtUseCount(inv)), Inline: &inline},
	}
	msg := discordwebhook.Message{
		Username: strPtr(botName),
		Embeds: &[]discordwebhook.Embed{{
			Title:  strPtr("new registration"),
			Fields: &fields,
		}},
	}
	if err := discordwebhook.SendMessage(n.webhookURL, msg); err != nil {
		return fmt.Errorf("notifications.DiscordNotifier.InviteUsed: %w", err)
	}
	return nil
}

// --- helpers ---

func strPtr(s string) *string { return &s }

func fmtExpiry(t *time.Time) string {
	if t == nil {
		return "never"
	}
	return t.UTC().Format("2006-01-02 15:04 UTC")
}

func fmtMaxUses(n *int) string {
	if n == nil {
		return "unlimited"
	}
	return fmt.Sprintf("%d", *n)
}

func fmtUseCount(inv domain.Invite) string {
	if inv.MaxUses == nil {
		return fmt.Sprintf("%d / ∞", inv.UseCount)
	}
	return fmt.Sprintf("%d / %d", inv.UseCount, *inv.MaxUses)
}
