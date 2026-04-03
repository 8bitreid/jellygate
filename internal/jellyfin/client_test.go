package jellyfin_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rmewborne/jellygate/internal/jellyfin"
)

// fakeJellyfin builds a test server that routes requests by method+path.
func fakeJellyfin(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for pattern, fn := range routes {
		mux.HandleFunc(pattern, fn)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func TestAuthenticate_Success(t *testing.T) {
	srv := fakeJellyfin(t, map[string]http.HandlerFunc{
		"POST /Users/AuthenticateByName": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"AccessToken": "tok-123"})
		},
	})

	client := jellyfin.New(srv.URL)
	token, err := client.Authenticate(context.Background(), "admin", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "tok-123" {
		t.Errorf("want token %q, got %q", "tok-123", token)
	}
}

func TestAuthenticate_BadCredentials(t *testing.T) {
	srv := fakeJellyfin(t, map[string]http.HandlerFunc{
		"POST /Users/AuthenticateByName": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		},
	})

	client := jellyfin.New(srv.URL)
	_, err := client.Authenticate(context.Background(), "admin", "wrong")
	if err == nil {
		t.Fatal("expected error for bad credentials, got nil")
	}
}

func TestListLibraries(t *testing.T) {
	srv := fakeJellyfin(t, map[string]http.HandlerFunc{
		"GET /Library/VirtualFolders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, []map[string]string{
				{"ItemId": "lib-1", "Name": "Movies"},
				{"ItemId": "lib-2", "Name": "TV Shows"},
			})
		},
	})

	client := jellyfin.New(srv.URL)
	libs, err := client.ListLibraries(context.Background(), "admin-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(libs) != 2 {
		t.Fatalf("want 2 libraries, got %d", len(libs))
	}
	if libs[0].ID != "lib-1" || libs[0].Name != "Movies" {
		t.Errorf("unexpected first library: %+v", libs[0])
	}
}

func TestCreateUser_Success(t *testing.T) {
	srv := fakeJellyfin(t, map[string]http.HandlerFunc{
		"POST /Users/New": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"Id": "user-abc"})
		},
	})

	client := jellyfin.New(srv.URL)
	uid, err := client.CreateUser(context.Background(), "admin-token", "newuser", "pass123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid != "user-abc" {
		t.Errorf("want user ID %q, got %q", "user-abc", uid)
	}
}

func TestCreateUser_SendsCorrectPayload(t *testing.T) {
	var gotBody map[string]string
	srv := fakeJellyfin(t, map[string]http.HandlerFunc{
		"POST /Users/New": func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&gotBody)
			writeJSON(w, http.StatusOK, map[string]string{"Id": "user-xyz"})
		},
	})

	client := jellyfin.New(srv.URL)
	client.CreateUser(context.Background(), "tok", "alice", "hunter2")

	if gotBody["Name"] != "alice" {
		t.Errorf("want Name=alice, got %q", gotBody["Name"])
	}
	if gotBody["Password"] != "hunter2" {
		t.Errorf("want Password=hunter2, got %q", gotBody["Password"])
	}
}

func TestSetLibraryAccess_SendsPolicy(t *testing.T) {
	var gotPolicy map[string]any
	srv := fakeJellyfin(t, map[string]http.HandlerFunc{
		"POST /Users/user-123/Policy": func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&gotPolicy)
			w.WriteHeader(http.StatusNoContent)
		},
	})

	client := jellyfin.New(srv.URL)
	err := client.SetLibraryAccess(context.Background(), "tok", "user-123", []string{"lib-1", "lib-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPolicy["EnableAllFolders"] != false {
		t.Errorf("want EnableAllFolders=false when libraries specified")
	}
	folders, _ := gotPolicy["EnabledFolders"].([]any)
	if len(folders) != 2 {
		t.Errorf("want 2 enabled folders, got %d", len(folders))
	}
}

func TestSetLibraryAccess_EmptyListEnablesAll(t *testing.T) {
	var gotPolicy map[string]any
	srv := fakeJellyfin(t, map[string]http.HandlerFunc{
		"POST /Users/user-123/Policy": func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&gotPolicy)
			w.WriteHeader(http.StatusNoContent)
		},
	})

	client := jellyfin.New(srv.URL)
	client.SetLibraryAccess(context.Background(), "tok", "user-123", []string{})

	if gotPolicy["EnableAllFolders"] != true {
		t.Errorf("want EnableAllFolders=true when no libraries specified")
	}
}
