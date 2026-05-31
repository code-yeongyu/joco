package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunSubmitUsesStoredSessionCookie(t *testing.T) {
	t.Setenv("JOCOHUNT_CONFIG_DIR", t.TempDir())
	var gotCookie string
	var gotCSRF string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotCSRF = r.Header.Get("X-CSRF-Token")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"slug":"stored-auth"}`))
	}))
	defer server.Close()
	var setupOut bytes.Buffer
	var setupErr bytes.Buffer
	if err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"auth", "login",
		"--session-cookie", "better-auth.session_token=stored",
		"--csrf-token", "csrf-stored",
	}, &setupOut, &setupErr); err != nil {
		t.Fatalf("store session: %v", err)
	}
	var out bytes.Buffer
	var errOut bytes.Buffer

	err := Run(context.Background(), []string{
		"--base-url", server.URL,
		"submit",
		"--title", "Stored Auth Tool",
		"--url", "https://example.com",
		"--tagline", "Stored auth",
		"--confirm",
	}, &out, &errOut)

	if err != nil {
		t.Fatalf("submit with stored session: %v", err)
	}
	if gotCookie != "better-auth.session_token=stored" {
		t.Fatalf("expected stored cookie, got %q", gotCookie)
	}
	if gotCSRF != "csrf-stored" {
		t.Fatalf("expected stored csrf token, got %q", gotCSRF)
	}
}
