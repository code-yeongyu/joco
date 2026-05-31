package jocohunt

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestClientStartGitHubLoginReturnsOAuthURLWhenServerProvidesRedirect(t *testing.T) {
	// Given
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/sign-in/social" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotBody = readTestBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"url":"https://github.com/login/oauth/authorize?client_id=test","redirect":true}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// When
	login, err := client.StartGitHubLogin(context.Background(), "/submit")

	// Then
	if err != nil {
		t.Fatalf("start github login: %v", err)
	}
	if login.URL != "https://github.com/login/oauth/authorize?client_id=test" {
		t.Fatalf("unexpected login URL: %q", login.URL)
	}
	if !strings.Contains(gotBody, `"provider":"github"`) || !strings.Contains(gotBody, `"callbackURL":"/submit"`) {
		t.Fatalf("unexpected login request body: %s", gotBody)
	}
}

func TestClientVerifySessionReportsAuthenticatedWhenSessionEndpointReturnsObject(t *testing.T) {
	// Given
	var gotCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/get-session" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user":{"email":"maker@example.com"}}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// When
	status, err := client.VerifySession(context.Background(), "better-auth.session_token=abc")

	// Then
	if err != nil {
		t.Fatalf("verify session: %v", err)
	}
	if !status.Authenticated {
		t.Fatal("expected authenticated session")
	}
	if gotCookie != "better-auth.session_token=abc" {
		t.Fatalf("expected cookie header, got %q", gotCookie)
	}
}

func TestSessionStoreRoundTripsSessionWithUserOnlyPermissions(t *testing.T) {
	// Given
	path := filepath.Join(t.TempDir(), "session.json")
	session := AuthSession{
		BaseURL:       "https://jocohunt.jocoding.io",
		SessionCookie: "better-auth.session_token=abc",
		CSRFToken:     "csrf",
		CreatedAt:     time.Date(2026, 5, 31, 1, 2, 3, 0, time.UTC),
	}

	// When
	if err := SaveAuthSession(path, session); err != nil {
		t.Fatalf("save auth session: %v", err)
	}
	loaded, err := LoadAuthSession(path)

	// Then
	if err != nil {
		t.Fatalf("load auth session: %v", err)
	}
	if loaded.SessionCookie != session.SessionCookie || loaded.CSRFToken != session.CSRFToken {
		t.Fatalf("unexpected loaded session: %#v", loaded)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat session file: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 auth file, got %v", info.Mode().Perm())
	}
}

func TestLoadAuthSessionReturnsErrNoAuthSessionWhenFileMissing(t *testing.T) {
	// Given
	path := filepath.Join(t.TempDir(), "missing.json")

	// When
	_, err := LoadAuthSession(path)

	// Then
	if !errors.Is(err, ErrNoAuthSession) {
		t.Fatalf("expected ErrNoAuthSession, got %v", err)
	}
}

func TestLoadAuthSessionReturnsDecodeErrorForCorruptFile(t *testing.T) {
	// Given
	path := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(path, []byte(`{bad json`), 0o600); err != nil {
		t.Fatalf("write corrupt auth file: %v", err)
	}

	// When
	_, err := LoadAuthSession(path)

	// Then
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "decode auth file") {
		t.Fatalf("expected decode context, got %v", err)
	}
}

func TestSaveAuthSessionRejectsEmptyCookie(t *testing.T) {
	// Given
	path := filepath.Join(t.TempDir(), "session.json")

	// When
	err := SaveAuthSession(path, AuthSession{BaseURL: "https://jocohunt.jocoding.io"})

	// Then
	if err == nil {
		t.Fatal("expected empty cookie error")
	}
	if !strings.Contains(err.Error(), "session cookie is required") {
		t.Fatalf("expected cookie validation error, got %v", err)
	}
}

func readTestBody(t *testing.T, r *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(data)
}
