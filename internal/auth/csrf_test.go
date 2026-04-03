package auth_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/rmewborne/jellygate/internal/auth"
)

func TestCSRF_ValidRoundTrip(t *testing.T) {
	w := httptest.NewRecorder()
	token, err := auth.SetCSRFCookie(w, false)
	if err != nil {
		t.Fatalf("SetCSRFCookie: %v", err)
	}

	// Build a POST request that echoes the token back in the form field.
	form := url.Values{auth.CSRFFieldName: {token}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Attach the cookie from the response.
	for _, c := range w.Result().Cookies() {
		req.AddCookie(c)
	}

	if !auth.ValidateCSRF(req) {
		t.Error("expected CSRF validation to pass")
	}
}

func TestCSRF_MissingCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if auth.ValidateCSRF(req) {
		t.Error("expected CSRF validation to fail with no cookie")
	}
}

func TestCSRF_MismatchedToken(t *testing.T) {
	w := httptest.NewRecorder()
	_, err := auth.SetCSRFCookie(w, false)
	if err != nil {
		t.Fatalf("SetCSRFCookie: %v", err)
	}

	form := url.Values{auth.CSRFFieldName: {"wrong-token"}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range w.Result().Cookies() {
		req.AddCookie(c)
	}

	if auth.ValidateCSRF(req) {
		t.Error("expected CSRF validation to fail with mismatched token")
	}
}
