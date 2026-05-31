package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_Run_printsJSONProducts_whenProductsCommandUsesJSON(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<script type="application/ld+json">{"@type":"CollectionPage","mainEntity":{"@type":"ItemList","itemListElement":[{"position":1,"url":"https://jocohunt.jocoding.io/p/demo","item":{"name":"Demo Product","description":"Ship faster","author":{"name":"@maker"}}}]}}</script>`))
	}))
	defer server.Close()
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{"--base-url", server.URL, "products", "--json"}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("run products: %v", err)
	}
	if !strings.Contains(out.String(), `"title": "Demo Product"`) {
		t.Fatalf("expected JSON product title, got %s", out.String())
	}
}

func Test_Run_returnsUsageError_whenCommandUnknown(t *testing.T) {
	// Given
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{"wat"}, &out, &errOut)

	// Then
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("expected unknown command message, got %q", errOut.String())
	}
}

func Test_Run_printsHelpWithoutError_whenHelpFlagProvided(t *testing.T) {
	// Given
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{"--help"}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("expected help without error, got %v", err)
	}
	if !strings.Contains(errOut.String(), "Usage of jocohunt") {
		t.Fatalf("expected help text, got %q", errOut.String())
	}
}

func Test_Run_printsEmptyMessage_whenCollectionHasNoItems(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<script type="application/ld+json">{"@type":"CollectionPage","mainEntity":{"@type":"ItemList","itemListElement":[]}}</script>`))
	}))
	defer server.Close()
	var out bytes.Buffer
	var errOut bytes.Buffer

	// When
	err := Run(context.Background(), []string{"--base-url", server.URL, "leaderboard"}, &out, &errOut)

	// Then
	if err != nil {
		t.Fatalf("run leaderboard: %v", err)
	}
	if !strings.Contains(out.String(), "No items found") {
		t.Fatalf("expected empty message, got %q", out.String())
	}
}
