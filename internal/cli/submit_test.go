package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeongyu/jocohunt/internal/jocohunt"
)

func TestRunSubmitDryRunPrintsPayloadWithoutNetwork(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	var out bytes.Buffer
	var errOut bytes.Buffer

	err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"submit",
		"--title", "Launch Tool",
		"--url", "https://example.com",
		"--tagline", "Ship today",
		"--dry-run",
	}, &out, &errOut)

	if err != nil {
		t.Fatalf("run submit dry-run: %v", err)
	}
	if called {
		t.Fatal("dry-run should not call server")
	}
	var plan map[string]any
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatalf("decode dry-run JSON: %v", err)
	}
	if plan["endpoint"] != "/api/submit" {
		t.Fatalf("expected default endpoint, got %#v", plan["endpoint"])
	}
	payload, ok := plan["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected payload object, got %#v", plan["payload"])
	}
	if payload["name"] != "Launch Tool" || payload["title"] != "Launch Tool" {
		t.Fatalf("expected submit payload aliases, got %#v", payload)
	}
}

func TestRunSubmitRequiresConfirmationForLiveWrite(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	err := Run(context.Background(), []string{
		"submit",
		"--title", "Launch Tool",
		"--url", "https://example.com",
		"--tagline", "Ship today",
		"--session-cookie", "better-auth.session_token=abc",
	}, &out, &errOut)

	if err == nil {
		t.Fatal("expected confirm error")
	}
	if !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("expected --confirm error, got %v", err)
	}
}

func TestRunSubmitRejectsMissingFields(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	err := Run(context.Background(), []string{
		"submit",
		"--title", "Launch Tool",
		"--dry-run",
	}, &out, &errOut)

	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "url is required") {
		t.Fatalf("expected url validation error, got %v", err)
	}
}

func TestRunSubmitPostsWhenConfirmed(t *testing.T) {
	var gotCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"slug":"launch-tool"}`))
	}))
	defer server.Close()
	var out bytes.Buffer
	var errOut bytes.Buffer

	err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"submit",
		"--title", "Launch Tool",
		"--url", "https://example.com",
		"--tagline", "Ship today",
		"--session-cookie", "better-auth.session_token=abc",
		"--confirm",
	}, &out, &errOut)

	if err != nil {
		t.Fatalf("run submit: %v", err)
	}
	if gotCookie != "better-auth.session_token=abc" {
		t.Fatalf("expected cookie header, got %q", gotCookie)
	}
	if !strings.Contains(out.String(), "launch-tool") {
		t.Fatalf("expected response output, got %q", out.String())
	}
}

func TestRunSubmitUsesSavedSessionWhenCookieFlagMissing(t *testing.T) {
	// Given
	authFile := filepath.Join(t.TempDir(), "session.json")
	var gotCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"slug":"saved-session-tool"}`))
	}))
	defer server.Close()
	if err := jocohunt.SaveAuthSession(authFile, jocohunt.AuthSession{
		BaseURL:       server.URL,
		SessionCookie: "better-auth.session_token=saved",
	}); err != nil {
		t.Fatalf("save auth session: %v", err)
	}
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"submit",
		"--title", "Saved Session Tool",
		"--url", "https://example.com",
		"--tagline", "Ship today",
		"--confirm",
	}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("run submit: %v", err)
	}
	if gotCookie != "better-auth.session_token=saved" {
		t.Fatalf("expected saved cookie header, got %q", gotCookie)
	}
	if !strings.Contains(out.String(), "saved-session-tool") {
		t.Fatalf("expected response output, got %q", out.String())
	}
}

func TestRunSubmitPrefersCookieFlagOverSavedSession(t *testing.T) {
	// Given
	authFile := filepath.Join(t.TempDir(), "session.json")
	var gotCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"slug":"flag-session-tool"}`))
	}))
	defer server.Close()
	if err := jocohunt.SaveAuthSession(authFile, jocohunt.AuthSession{
		BaseURL:       server.URL,
		SessionCookie: "better-auth.session_token=saved",
	}); err != nil {
		t.Fatalf("save auth session: %v", err)
	}
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"submit",
		"--title", "Flag Session Tool",
		"--url", "https://example.com",
		"--tagline", "Ship today",
		"--session-cookie", "better-auth.session_token=flag",
		"--confirm",
	}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("run submit: %v", err)
	}
	if gotCookie != "better-auth.session_token=flag" {
		t.Fatalf("expected flag cookie header, got %q", gotCookie)
	}
}

func TestRunSubmitPrefersEnvCookieOverSavedSession(t *testing.T) {
	// Given
	t.Setenv("JOCOHUNT_SESSION_COOKIE", "better-auth.session_token=env")
	authFile := filepath.Join(t.TempDir(), "session.json")
	var gotCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"slug":"env-session-tool"}`))
	}))
	defer server.Close()
	if err := jocohunt.SaveAuthSession(authFile, jocohunt.AuthSession{
		BaseURL:       server.URL,
		SessionCookie: "better-auth.session_token=saved",
	}); err != nil {
		t.Fatalf("save auth session: %v", err)
	}
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"submit",
		"--title", "Env Session Tool",
		"--url", "https://example.com",
		"--tagline", "Ship today",
		"--confirm",
	}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("run submit: %v", err)
	}
	if gotCookie != "better-auth.session_token=env" {
		t.Fatalf("expected env cookie header, got %q", gotCookie)
	}
}

func TestRunSubmitDryRunRedactsSavedSessionSecrets(t *testing.T) {
	// Given
	authFile := filepath.Join(t.TempDir(), "session.json")
	if err := jocohunt.SaveAuthSession(authFile, jocohunt.AuthSession{
		BaseURL:       "https://jocohunt.jocoding.io",
		SessionCookie: "better-auth.session_token=saved",
		CSRFToken:     "csrf-secret",
	}); err != nil {
		t.Fatalf("save auth session: %v", err)
	}
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{
		"--auth-file", authFile,
		"submit",
		"--title", "Dry Run Tool",
		"--url", "https://example.com",
		"--tagline", "Ship today",
		"--dry-run",
	}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("run submit dry-run: %v", err)
	}
	body := out.String()
	if strings.Contains(body, "better-auth.session_token=saved") || strings.Contains(body, "csrf-secret") {
		t.Fatalf("dry-run leaked auth secrets: %s", body)
	}
	if !strings.Contains(body, `"sessionCookie": true`) || !strings.Contains(body, `"csrfToken": true`) {
		t.Fatalf("expected auth presence booleans, got %s", body)
	}
}

func TestRunSubmitIgnoresSavedSessionWhenBaseURLDiffers(t *testing.T) {
	// Given
	authFile := filepath.Join(t.TempDir(), "session.json")
	if err := jocohunt.SaveAuthSession(authFile, jocohunt.AuthSession{
		BaseURL:       "https://jocohunt.jocoding.io",
		SessionCookie: "better-auth.session_token=saved",
	}); err != nil {
		t.Fatalf("save auth session: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"slug":"should-not-send-cookie"}`))
	}))
	defer server.Close()
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"submit",
		"--title", "Saved Session Tool",
		"--url", "https://example.com",
		"--tagline", "Ship today",
		"--confirm",
	}, &out, &errOut)

	// Then
	if err == nil {
		t.Fatal("expected auth mismatch error")
	}
	if !strings.Contains(err.Error(), "auth login") && !strings.Contains(err.Error(), "session") {
		t.Fatalf("expected helpful auth error, got %v", err)
	}
}
