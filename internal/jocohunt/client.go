package jocohunt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewClient(rawBaseURL string, timeout time.Duration) (*Client, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	parsed, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("base url %q: %w", rawBaseURL, ErrInvalidKind)
	}
	return &Client{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) ListItems(ctx context.Context, q Query) ([]Item, error) {
	path, err := pathForQuery(q)
	if err != nil {
		return nil, err
	}
	body, _, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	if redirectPath := metaRefreshPath(body); redirectPath != "" {
		body, _, err = c.get(ctx, redirectPath)
		if err != nil {
			return nil, err
		}
	}
	items, err := parseItems(body)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", q.Kind, err)
	}
	return limitItems(items, q.Limit), nil
}

func (c *Client) get(ctx context.Context, path string) (string, http.Header, error) {
	ref, err := url.Parse(path)
	if err != nil {
		return "", nil, fmt.Errorf("parse request path: %w", err)
	}
	u := c.baseURL.ResolveReference(ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "jocohunt-cli/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("get %s: %w", u.String(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", resp.Header, fmt.Errorf("get %s: status %d", u.String(), resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", resp.Header, fmt.Errorf("read response: %w", err)
	}
	return string(data), resp.Header, nil
}

func pathForQuery(q Query) (string, error) {
	switch q.Kind {
	case KindProducts:
		return pathWithQuery("/products", map[string]string{"category": q.Category, "q": q.Search}), nil
	case KindIdeas:
		return pathWithQuery("/ideas", map[string]string{"tab": q.Tab}), nil
	case KindLeaderboard:
		period := strings.TrimSpace(q.Period)
		if period == "" {
			period = "weekly"
		}
		return "/leaderboard/" + url.PathEscape(period), nil
	default:
		return "", fmt.Errorf("kind %q: %w", q.Kind, ErrInvalidKind)
	}
}

func pathWithQuery(path string, values map[string]string) string {
	q := url.Values{}
	for key, value := range values {
		if strings.TrimSpace(value) != "" {
			q.Set(key, value)
		}
	}
	if len(q) == 0 {
		return path
	}
	return path + "?" + q.Encode()
}

func limitItems(items []Item, limit int) []Item {
	if limit <= 0 || limit >= len(items) {
		return items
	}
	return items[:limit]
}
