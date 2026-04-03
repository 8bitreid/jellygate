package notifications_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/internal/notifications"
)

func fakeWebhook(t *testing.T, statusCode int, fn func(body map[string]any)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("want POST, got %s", r.Method)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode payload: %v", err)
		}
		if fn != nil {
			fn(payload)
		}
		w.WriteHeader(statusCode)
	}))
}

func TestDiscordNotifier_InviteCreated(t *testing.T) {
	var got map[string]any
	srv := fakeWebhook(t, http.StatusNoContent, func(body map[string]any) { got = body })
	defer srv.Close()

	n := notifications.NewDiscordNotifier(srv.URL)
	inv := domain.Invite{
		ID:        "inv-1",
		Token:     "tok",
		Label:     "friends",
		CreatedBy: "admin",
	}

	if err := n.InviteCreated(context.Background(), inv); err != nil {
		t.Fatalf("InviteCreated: %v", err)
	}

	if got["username"] != "jellygate" {
		t.Errorf("want username jellygate, got %v", got["username"])
	}
	embeds := got["embeds"].([]any)
	if len(embeds) == 0 {
		t.Fatal("expected at least one embed")
	}
	embed := embeds[0].(map[string]any)
	if embed["title"] != "invite created" {
		t.Errorf("want title 'invite created', got %v", embed["title"])
	}

	// Verify invite label appears in a field value.
	fields := embed["fields"].([]any)
	var foundLabel bool
	for _, f := range fields {
		field := f.(map[string]any)
		if v, ok := field["value"].(string); ok && strings.Contains(v, "friends") {
			foundLabel = true
		}
	}
	if !foundLabel {
		t.Error("expected invite label 'friends' in embed fields")
	}
}

func TestDiscordNotifier_InviteCreated_WithExpiry(t *testing.T) {
	var got map[string]any
	srv := fakeWebhook(t, http.StatusNoContent, func(body map[string]any) { got = body })
	defer srv.Close()

	n := notifications.NewDiscordNotifier(srv.URL)
	exp := time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC)
	max := 5
	inv := domain.Invite{Label: "vip", CreatedBy: "admin", ExpiresAt: &exp, MaxUses: &max}

	if err := n.InviteCreated(context.Background(), inv); err != nil {
		t.Fatalf("InviteCreated: %v", err)
	}

	embeds := got["embeds"].([]any)
	embed := embeds[0].(map[string]any)
	fields := embed["fields"].([]any)

	var foundExpiry, foundMax bool
	for _, f := range fields {
		field := f.(map[string]any)
		v, _ := field["value"].(string)
		if strings.Contains(v, "2026-12-31") {
			foundExpiry = true
		}
		if strings.Contains(v, "5") {
			foundMax = true
		}
	}
	if !foundExpiry {
		t.Error("expected expiry date in fields")
	}
	if !foundMax {
		t.Error("expected max uses in fields")
	}
}

func TestDiscordNotifier_InviteUsed(t *testing.T) {
	var got map[string]any
	srv := fakeWebhook(t, http.StatusNoContent, func(body map[string]any) { got = body })
	defer srv.Close()

	n := notifications.NewDiscordNotifier(srv.URL)
	inv := domain.Invite{Label: "friends", UseCount: 1}

	if err := n.InviteUsed(context.Background(), inv, "alice"); err != nil {
		t.Fatalf("InviteUsed: %v", err)
	}

	embeds := got["embeds"].([]any)
	embed := embeds[0].(map[string]any)
	if embed["title"] != "new registration" {
		t.Errorf("want title 'new registration', got %v", embed["title"])
	}

	fields := embed["fields"].([]any)
	var foundUser bool
	for _, f := range fields {
		field := f.(map[string]any)
		if v, ok := field["value"].(string); ok && v == "alice" {
			foundUser = true
		}
	}
	if !foundUser {
		t.Error("expected username 'alice' in embed fields")
	}
}

func TestDiscordNotifier_WebhookError(t *testing.T) {
	srv := fakeWebhook(t, http.StatusInternalServerError, nil)
	defer srv.Close()

	n := notifications.NewDiscordNotifier(srv.URL)
	err := n.InviteCreated(context.Background(), domain.Invite{Label: "x", CreatedBy: "admin"})
	if err == nil {
		t.Error("expected error on non-2xx response")
	}
	if !strings.Contains(err.Error(), "InviteCreated") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestNoopNotifier(t *testing.T) {
	n := &notifications.NoopNotifier{}
	if err := n.InviteCreated(context.Background(), domain.Invite{}); err != nil {
		t.Errorf("NoopNotifier.InviteCreated: %v", err)
	}
	if err := n.InviteUsed(context.Background(), domain.Invite{}, "user"); err != nil {
		t.Errorf("NoopNotifier.InviteUsed: %v", err)
	}
}
