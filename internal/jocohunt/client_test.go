package jocohunt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_Client_ListItems_returnsProducts_whenCollectionPageHasJsonLD(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head><script type="application/ld+json">{
			"@context":"https://schema.org",
			"@type":"CollectionPage",
			"mainEntity":{
				"@type":"ItemList",
				"itemListElement":[
					{"@type":"ListItem","position":1,"url":"https://jocohunt.jocoding.io/p/demo","item":{"@type":"SoftwareApplication","name":"Demo Product","description":"A useful Korean builder tool","author":{"@type":"Person","name":"@maker"}}}
				]
			}
		}</script></head></html>`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// When
	items, err := client.ListItems(context.Background(), Query{Kind: KindProducts, Limit: 10})

	// Then
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Title != "Demo Product" {
		t.Fatalf("expected title Demo Product, got %q", items[0].Title)
	}
	if items[0].Author != "@maker" {
		t.Fatalf("expected author @maker, got %q", items[0].Author)
	}
}

func Test_Client_ListItems_rejectsInvalidKind_whenKindUnknown(t *testing.T) {
	// Given
	client, err := NewClient("https://jocohunt.jocoding.io", 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// When
	_, err = client.ListItems(context.Background(), Query{Kind: Kind("bad")})

	// Then
	if err == nil {
		t.Fatal("expected invalid kind error")
	}
}

func Test_Client_ListItems_returnsProducts_whenProductPageUsesAnchors(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<ul>
			<li><p><a class="font-semibold" href="/p/demo1">QuadWork</a><span> — AI agent workspace</span></p></li>
			<li><p><a class="font-semibold" href="/p/demo2">AgentBridge</a><span> — Shared memory for agents</span></p></li>
		</ul>`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// When
	items, err := client.ListItems(context.Background(), Query{Kind: KindProducts, Limit: 2})

	// Then
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Title != "QuadWork" {
		t.Fatalf("expected first product QuadWork, got %q", items[0].Title)
	}
}

func Test_Client_ListItems_preservesProductQuery_whenCategoryAndSearchProvided(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("category"); got != "ai-tools" {
			t.Fatalf("expected category query ai-tools, got %q", got)
		}
		if got := r.URL.Query().Get("q"); got != "agent" {
			t.Fatalf("expected search query agent, got %q", got)
		}
		_, _ = w.Write([]byte(`<a class="font-semibold" href="/p/demo">Query Product</a>`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// When
	items, err := client.ListItems(context.Background(), Query{
		Kind:     KindProducts,
		Limit:    1,
		Category: "ai-tools",
		Search:   "agent",
	})

	// Then
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func Test_Client_ListItems_preservesIdeasQuery_whenTabProvided(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ideas" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("tab"); got != "recent" {
			t.Fatalf("expected tab query recent, got %q", got)
		}
		_, _ = w.Write([]byte(`<a class="font-semibold" href="/p/idea">Query Idea</a>`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// When
	items, err := client.ListItems(context.Background(), Query{
		Kind:  KindIdeas,
		Limit: 1,
		Tab:   "recent",
	})

	// Then
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func Test_Client_ListItems_returnsLeaderboard_whenPageUsesProductAnchors(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<a class="font-semibold" href="/p/rank">Ranked Tool</a>`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// When
	items, err := client.ListItems(context.Background(), Query{Kind: KindLeaderboard, Limit: 1})

	// Then
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if items[0].Title != "Ranked Tool" {
		t.Fatalf("expected Ranked Tool, got %q", items[0].Title)
	}
}

func Test_Client_ListItems_returnsEmptyLeaderboard_whenJsonLDItemListIsEmpty(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<script type="application/ld+json">{"@type":"CollectionPage","mainEntity":{"@type":"ItemList","itemListElement":[]}}</script>`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// When
	items, err := client.ListItems(context.Background(), Query{Kind: KindLeaderboard, Limit: 1})

	// Then
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty leaderboard, got %d items", len(items))
	}
}

func Test_Client_ListItems_followsMetaRefresh_whenLeaderboardRedirects(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/leaderboard/weekly":
			_, _ = w.Write([]byte(`<meta http-equiv="refresh" content="1;url=/leaderboard/weekly/2026/23">`))
		case "/leaderboard/weekly/2026/23":
			_, _ = w.Write([]byte(`<script type="application/ld+json">{"@type":"CollectionPage","mainEntity":{"@type":"ItemList","itemListElement":[]}}</script>`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// When
	items, err := client.ListItems(context.Background(), Query{Kind: KindLeaderboard, Period: "weekly"})

	// Then
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected redirected empty leaderboard, got %d items", len(items))
	}
}

func Test_Client_InspectSecurityHeaders_reportsHeaders_whenHomeResponds(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		_, _ = w.Write([]byte(`<html><title>ok</title></html>`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// When
	report, err := client.Inspect(context.Background())

	// Then
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !report.SecurityHeaders["Content-Security-Policy"] {
		t.Fatal("expected Content-Security-Policy to be present")
	}
	if !report.SecurityHeaders["Strict-Transport-Security"] {
		t.Fatal("expected Strict-Transport-Security to be present")
	}
}
