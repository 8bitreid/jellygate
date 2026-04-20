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
// The webhook URL is read from settings on each notification; if not configured
// the notification is silently skipped.
type DiscordNotifier struct {
	settings domain.SettingsStore
}

// NewDiscordNotifier creates a DiscordNotifier that reads its webhook URL from settings.
func NewDiscordNotifier(settings domain.SettingsStore) *DiscordNotifier {
	return &DiscordNotifier{settings: settings}
}

// InviteCreated fires when an admin creates a new invite.
func (n *DiscordNotifier) InviteCreated(ctx context.Context, inv domain.Invite) error {
	webhookURL, err := n.settings.GetDiscordWebhookURL(ctx)
	if err != nil {
		return nil // not configured — silently skip
	}
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
	if err := discordwebhook.SendMessage(webhookURL, msg); err != nil {
		return fmt.Errorf("notifications.DiscordNotifier.InviteCreated: %w", err)
	}
	return nil
}

// InviteUsed fires when a user registers via an invite.
func (n *DiscordNotifier) InviteUsed(ctx context.Context, inv domain.Invite, username string) error {
	webhookURL, err := n.settings.GetDiscordWebhookURL(ctx)
	if err != nil {
		return nil // not configured — silently skip
	}
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
	if err := discordwebhook.SendMessage(webhookURL, msg); err != nil {
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
