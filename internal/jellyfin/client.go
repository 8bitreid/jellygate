package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/rmewborne/jellygate/internal/domain"
)

const (
	headerAuth        = "Authorization"
	headerContentType = "Content-Type"
	applicationJSON   = "application/json"
	// authHeader is sent on every request to identify jellygate as the client.
	clientAuthHeader = `MediaBrowser Client="jellygate", Device="server", DeviceId="jellygate", Version="1.0"`
)

// Client is an HTTP client for the Jellyfin API.
// It implements domain.JellyfinClient.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New returns a Client targeting the given Jellyfin base URL.
func New(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{},
	}
}

// Authenticate exchanges admin credentials for a Jellyfin access token.
func (c *Client) Authenticate(ctx context.Context, username, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"Username": username,
		"Pw":       password,
	})

	req, err := c.newRequest(ctx, http.MethodPost, "/Users/AuthenticateByName", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	var resp struct {
		AccessToken string `json:"AccessToken"`
	}
	if err := c.do(req, http.StatusOK, &resp); err != nil {
		return "", fmt.Errorf("jellyfin.Authenticate: %w", err)
	}
	if resp.AccessToken == "" {
		return "", fmt.Errorf("jellyfin.Authenticate: empty access token in response")
	}
	return resp.AccessToken, nil
}

// ListLibraries returns all virtual folders (libraries) visible to the admin token.
func (c *Client) ListLibraries(ctx context.Context, adminToken string) ([]domain.Library, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/Library/VirtualFolders", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(headerAuth, clientAuthHeader+`, Token="`+adminToken+`"`)

	var items []struct {
		ItemId string `json:"ItemId"`
		Name   string `json:"Name"`
	}
	if err := c.do(req, http.StatusOK, &items); err != nil {
		return nil, fmt.Errorf("jellyfin.ListLibraries: %w", err)
	}

	libs := make([]domain.Library, len(items))
	for i, it := range items {
		libs[i] = domain.Library{ID: it.ItemId, Name: it.Name}
	}
	return libs, nil
}

// CreateUser creates a new Jellyfin user and returns their user ID.
func (c *Client) CreateUser(ctx context.Context, adminToken, username, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"Name":     username,
		"Password": password,
	})

	req, err := c.newRequest(ctx, http.MethodPost, "/Users/New", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set(headerAuth, clientAuthHeader+`, Token="`+adminToken+`"`)

	var resp struct {
		Id string `json:"Id"`
	}
	if err := c.do(req, http.StatusOK, &resp); err != nil {
		return "", fmt.Errorf("jellyfin.CreateUser: %w", err)
	}
	if resp.Id == "" {
		return "", fmt.Errorf("jellyfin.CreateUser: empty user ID in response")
	}
	return resp.Id, nil
}

// SetLibraryAccess applies a library access policy to an existing user.
// Only the listed libraryIDs will be enabled; all others are blocked.
func (c *Client) SetLibraryAccess(ctx context.Context, adminToken, userID string, libraryIDs []string) error {
	// Build the EnabledFolders list and set EnforceParentalRating=false,
	// EnableAllFolders based on whether the list is empty.
	enableAll := len(libraryIDs) == 0
	policy := map[string]any{
		"EnableAllFolders": enableAll,
		"EnabledFolders":   libraryIDs,
	}
	body, _ := json.Marshal(policy)

	path := "/Users/" + url.PathEscape(userID) + "/Policy"
	req, err := c.newRequest(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set(headerAuth, clientAuthHeader+`, Token="`+adminToken+`"`)

	if err := c.do(req, http.StatusNoContent, nil); err != nil {
		return fmt.Errorf("jellyfin.SetLibraryAccess: %w", err)
	}
	return nil
}

// newRequest builds an HTTP request with common headers pre-set.
func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("jellyfin: build request %s %s: %w", method, path, err)
	}
	req.Header.Set(headerContentType, applicationJSON)
	req.Header.Set(headerAuth, clientAuthHeader)
	return req, nil
}

// do executes a request, checks the status code, and decodes the JSON body into dst.
// If dst is nil, the response body is discarded.
func (c *Client) do(req *http.Request, wantStatus int, dst any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		slog.Error("jellyfin request failed",
			"method", req.Method,
			"url", req.URL.String(),
			"status", resp.StatusCode,
		)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
