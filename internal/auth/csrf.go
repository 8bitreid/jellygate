package auth

import (
	"crypto/subtle"
	"net/http"
)

const (
	CSRFCookieName = "jg_csrf"
	CSRFFieldName  = "csrf_token"
)

// SetCSRFCookie generates a CSRF token and writes it as a cookie.
// The same token must be submitted in the form field on POST.
func SetCSRFCookie(w http.ResponseWriter, secure bool) (string, error) {
	token, err := GenerateToken()
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // must be readable by the form template
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	return token, nil
}

// ValidateCSRF checks that the CSRF cookie matches the form field value.
// Uses constant-time comparison to prevent timing attacks.
func ValidateCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(CSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	formValue := r.FormValue(CSRFFieldName)
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(formValue)) == 1
}
