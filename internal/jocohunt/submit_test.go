package jocohunt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSubmitProductPostsPayloadWithAuthHeaders(t *testing.T) {
	var seen bool
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = true
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/submit" {
			t.Fatalf("expected /api/submit, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Cookie"); got != "better-auth.session_token=abc" {
			t.Fatalf("expected session cookie, got %q", got)
		}
		if got := r.Header.Get("Origin"); got != server.URL {
			t.Fatalf("expected origin %q, got %q", server.URL, got)
		}
		if got := r.Header.Get("Referer"); got != server.URL+"/submit" {
			t.Fatalf("expected submit referer, got %q", got)
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["name"] != "Launch Tool" || payload["title"] != "Launch Tool" {
			t.Fatalf("expected title aliases, got %#v", payload)
		}
		if payload["url"] != "https://example.com" || payload["tagline"] != "Ship today" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"slug":"launch-tool"}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	result, err := client.SubmitProduct(context.Background(), SubmitProductInput{
		Title:   "Launch Tool",
		URL:     "https://example.com",
		Tagline: "Ship today",
	}, SubmitOptions{SessionCookie: "better-auth.session_token=abc"})

	if err != nil {
		t.Fatalf("submit product: %v", err)
	}
	if !seen {
		t.Fatal("expected request")
	}
	if result.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", result.StatusCode)
	}
	if !strings.Contains(result.Body, "launch-tool") {
		t.Fatalf("expected response body, got %q", result.Body)
	}
}

func TestSubmitProductRejectsMissingRequiredFields(t *testing.T) {
	client, err := NewClient("https://jocohunt.jocoding.io", 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.SubmitProduct(context.Background(), SubmitProductInput{
		Title: "Missing URL",
	}, SubmitOptions{SessionCookie: "better-auth.session_token=abc"})

	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSubmitProductReturnsHelpfulErrorForAuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"로그인이 필요합니다"}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.SubmitProduct(context.Background(), SubmitProductInput{
		Title:   "Launch Tool",
		URL:     "https://example.com",
		Tagline: "Ship today",
	}, SubmitOptions{SessionCookie: "better-auth.session_token=abc"})

	if err == nil {
		t.Fatal("expected auth error")
	}
	if !strings.Contains(err.Error(), "status 401") || !strings.Contains(err.Error(), "로그인이 필요합니다") {
		t.Fatalf("expected status and body in error, got %v", err)
	}
}

func TestSubmitProductRejectsAbsoluteEndpointBeforeRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"slug":"should-not-be-called"}`))
	}))
	defer server.Close()
	client, err := NewClient("https://jocohunt.jocoding.io", 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.SubmitProduct(context.Background(), SubmitProductInput{
		Title:   "Launch Tool",
		URL:     "https://example.com",
		Tagline: "Ship today",
	}, SubmitOptions{
		Endpoint:      server.URL + "/api/submit",
		SessionCookie: "better-auth.session_token=abc",
	})

	if err == nil {
		t.Fatal("expected endpoint rejection")
	}
	if called {
		t.Fatal("expected absolute endpoint to be rejected before request")
	}
}
