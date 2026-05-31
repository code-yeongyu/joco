package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeongyu/jocohunt/internal/jocohunt"
)

func TestRunAuthLoginSavesSessionCookieWhenCookieProvided(t *testing.T) {
	// Given
	authFile := filepath.Join(t.TempDir(), "session.json")
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{
		"--auth-file", authFile,
		"auth", "login",
		"--session-cookie", "better-auth.session_token=abc",
		"--csrf-token", "csrf",
	}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("auth login: %v", err)
	}
	if !strings.Contains(out.String(), "Logged in") {
		t.Fatalf("expected login confirmation, got %q", out.String())
	}
	out.Reset()
	errOut.Reset()
	if err := Run(context.Background(), []string{"--auth-file", authFile, "auth", "status"}, &out, &errOut); err != nil {
		t.Fatalf("auth status: %v", err)
	}
	if !strings.Contains(out.String(), "Logged in") {
		t.Fatalf("expected saved login status, got %q", out.String())
	}
}

func TestRunAuthLoginPrintsOAuthURLWhenNoCookieProvided(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"url":"https://github.com/login/oauth/authorize?client_id=test","redirect":true}`))
	}))
	defer server.Close()
	authFile := filepath.Join(t.TempDir(), "session.json")
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"auth", "login",
		"--no-open",
	}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("auth login oauth: %v", err)
	}
	if !strings.Contains(out.String(), "https://github.com/login/oauth/authorize?client_id=test") {
		t.Fatalf("expected OAuth URL, got %q", out.String())
	}
	if !strings.Contains(out.String(), "not saved") {
		t.Fatalf("expected no-save guidance, got %q", out.String())
	}
}

func TestRunAuthLoginCapturesSessionCookieWhenNoCookieProvided(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"url":"https://github.com/login/oauth/authorize?client_id=test","redirect":true}`))
	}))
	defer server.Close()
	previousCapture := captureAuthSessionFromBrowser
	captureAuthSessionFromBrowser = func(ctx context.Context, loginURL string, baseURL string, opts authCaptureOptions) (string, error) {
		return "better-auth.session_token=captured", nil
	}
	t.Cleanup(func() {
		captureAuthSessionFromBrowser = previousCapture
	})
	authFile := filepath.Join(t.TempDir(), "session.json")
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"auth", "login",
	}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("auth login capture: %v", err)
	}
	if !strings.Contains(out.String(), "Logged in") {
		t.Fatalf("expected login completion, got %q", out.String())
	}
	if strings.Contains(out.String(), "better-auth.session_token=") || strings.Contains(errOut.String(), "better-auth.session_token=") {
		t.Fatalf("auth output leaked cookie: out=%q err=%q", out.String(), errOut.String())
	}
	if _, err := os.Stat(authFile); err != nil {
		t.Fatalf("expected auth file to be written: %v", err)
	}
}

func TestRunAuthLogoutRemovesSavedSession(t *testing.T) {
	// Given
	authFile := filepath.Join(t.TempDir(), "session.json")
	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := Run(context.Background(), []string{
		"--auth-file", authFile,
		"auth", "login",
		"--session-cookie", "better-auth.session_token=abc",
	}, &out, &errOut); err != nil {
		t.Fatalf("seed auth login: %v", err)
	}
	out.Reset()
	errOut.Reset()

	// When
	err := Run(context.Background(), []string{"--auth-file", authFile, "auth", "logout"}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("auth logout: %v", err)
	}
	if !strings.Contains(out.String(), "Logged out") {
		t.Fatalf("expected logout confirmation, got %q", out.String())
	}
	out.Reset()
	errOut.Reset()
	if err := Run(context.Background(), []string{"--auth-file", authFile, "auth", "status"}, &out, &errOut); err != nil {
		t.Fatalf("auth status after logout: %v", err)
	}
	if !strings.Contains(out.String(), "Not logged in") {
		t.Fatalf("expected logged-out status, got %q", out.String())
	}
}

func TestRunAuthLogoutHelpDoesNotRemoveSavedSession(t *testing.T) {
	// Given
	authFile := filepath.Join(t.TempDir(), "session.json")
	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := Run(context.Background(), []string{
		"--auth-file", authFile,
		"auth", "login",
		"--session-cookie", "better-auth.session_token=abc",
	}, &out, &errOut); err != nil {
		t.Fatalf("seed auth login: %v", err)
	}
	out.Reset()
	errOut.Reset()

	// When
	err := Run(context.Background(), []string{"--auth-file", authFile, "auth", "logout", "--help"}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("auth logout help: %v", err)
	}
	if _, err := os.Stat(authFile); err != nil {
		t.Fatalf("expected auth file to remain after help: %v", err)
	}
	if !strings.Contains(errOut.String(), "Usage of auth logout") {
		t.Fatalf("expected logout help, got out=%q err=%q", out.String(), errOut.String())
	}
}

func TestRunAuthStatusVerifyUsesSavedSessionCookie(t *testing.T) {
	// Given
	var gotCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user":{"email":"maker@example.com"}}`))
	}))
	defer server.Close()
	authFile := filepath.Join(t.TempDir(), "session.json")
	var setupOut bytes.Buffer
	var setupErr bytes.Buffer
	if err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"auth", "login",
		"--session-cookie", "better-auth.session_token=abc",
	}, &setupOut, &setupErr); err != nil {
		t.Fatalf("seed auth login: %v", err)
	}
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"auth", "status",
		"--verify",
	}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("auth status verify: %v", err)
	}
	if gotCookie != "better-auth.session_token=abc" {
		t.Fatalf("expected saved cookie header, got %q", gotCookie)
	}
	if !strings.Contains(out.String(), "Verified") {
		t.Fatalf("expected verified status, got %q", out.String())
	}
}

func TestRunAuthStatusVerifyIgnoresSavedSessionWhenBaseURLDiffers(t *testing.T) {
	authFile := filepath.Join(t.TempDir(), "session.json")
	if err := jocohunt.SaveAuthSession(authFile, jocohunt.AuthSession{
		BaseURL:       "https://jocohunt.jocoding.io",
		SessionCookie: "better-auth.session_token=abc",
	}); err != nil {
		t.Fatalf("save auth session: %v", err)
	}

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user":{"email":"maker@example.com"}}`))
	}))
	defer server.Close()
	var out bytes.Buffer
	var errOut bytes.Buffer

	err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"auth", "status",
		"--verify",
	}, &out, &errOut)

	if err != nil {
		t.Fatalf("auth status verify mismatch: %v", err)
	}
	if called {
		t.Fatalf("expected verify to skip server call for mismatched base url")
	}
	if !strings.Contains(out.String(), "Not logged in") {
		t.Fatalf("expected mismatch to be treated as logged-out, got %q", out.String())
	}
}

func TestRunAuthOutputDoesNotLeakSecrets(t *testing.T) {
	// Given
	authFile := filepath.Join(t.TempDir(), "session.json")
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{
		"--auth-file", authFile,
		"auth", "login",
		"--session-cookie", "better-auth.session_token=abc",
		"--csrf-token", "csrf-secret",
	}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("auth login: %v", err)
	}
	combined := out.String() + errOut.String()
	if strings.Contains(combined, "better-auth.session_token=abc") || strings.Contains(combined, "csrf-secret") {
		t.Fatalf("auth output leaked secrets: %q", combined)
	}
}

func TestRunAuthFileFlagTakesPrecedenceOverConfigDir(t *testing.T) {
	// Given
	configDir := t.TempDir()
	t.Setenv("JOCOHUNT_CONFIG_DIR", configDir)
	authFile := filepath.Join(t.TempDir(), "session.json")
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{
		"--auth-file", authFile,
		"auth", "login",
		"--session-cookie", "better-auth.session_token=abc",
	}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("auth login: %v", err)
	}
	if _, err := os.Stat(authFile); err != nil {
		t.Fatalf("expected explicit auth file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(configDir, "session.json")); !os.IsNotExist(err) {
		t.Fatalf("expected config dir session to stay absent, stat err %v", err)
	}
}
