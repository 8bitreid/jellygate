package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rmewborne/jellygate/internal/handler"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Errorf("expected body %q, got %q", "ok", body)
	}
}
