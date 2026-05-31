package jocohunt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	githubLoginEndpoint = "/api/auth/sign-in/social"
	sessionEndpoint     = "/api/auth/get-session"
)

type GitHubLogin struct {
	URL      string `json:"url"`
	Redirect bool   `json:"redirect"`
}

type SessionStatus struct {
	Authenticated bool
	Body          string
}

func (c *Client) StartGitHubLogin(ctx context.Context, callbackURL string) (GitHubLogin, error) {
	callback := strings.TrimSpace(callbackURL)
	if callback == "" {
		callback = "/submit"
	}
	payload := map[string]string{
		"provider":    "github",
		"callbackURL": callback,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return GitHubLogin{}, fmt.Errorf("encode github login payload: %w", err)
	}
	u, err := c.resolve(githubLoginEndpoint)
	if err != nil {
		return GitHubLogin{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(data))
	if err != nil {
		return GitHubLogin{}, fmt.Errorf("create github login request: %w", err)
	}
	req.Header.Set("User-Agent", "jocohunt-cli/0.1")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", originForURL(c.baseURL))
	req.Header.Set("Referer", c.baseURL.ResolveReference(&url.URL{Path: "/sign-in"}).String())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return GitHubLogin{}, fmt.Errorf("start github login %s: %w", u.String(), err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return GitHubLogin{}, fmt.Errorf("read github login response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GitHubLogin{}, fmt.Errorf("start github login %s: status %d: %s", u.String(), resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var login GitHubLogin
	if err := json.Unmarshal(body, &login); err != nil {
		return GitHubLogin{}, fmt.Errorf("decode github login response: %w", err)
	}
	if strings.TrimSpace(login.URL) == "" {
		return GitHubLogin{}, fmt.Errorf("start github login %s: missing oauth url", u.String())
	}
	return login, nil
}

func (c *Client) VerifySession(ctx context.Context, sessionCookie string) (SessionStatus, error) {
	cookie := strings.TrimSpace(sessionCookie)
	if cookie == "" {
		return SessionStatus{}, errorsForAuth("session cookie is required")
	}
	u, err := c.resolve(sessionEndpoint)
	if err != nil {
		return SessionStatus{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return SessionStatus{}, fmt.Errorf("create session request: %w", err)
	}
	req.Header.Set("User-Agent", "jocohunt-cli/0.1")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", originForURL(c.baseURL))
	req.Header.Set("Referer", c.baseURL.String())
	req.Header.Set("Cookie", cookie)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SessionStatus{}, fmt.Errorf("verify session %s: %w", u.String(), err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return SessionStatus{}, fmt.Errorf("read session response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SessionStatus{}, fmt.Errorf("verify session %s: status %d: %s", u.String(), resp.StatusCode, strings.TrimSpace(string(body)))
	}
	trimmed := strings.TrimSpace(string(body))
	return SessionStatus{
		Authenticated: trimmed != "" && trimmed != "null",
		Body:          trimmed,
	}, nil
}

func (c *Client) resolve(path string) (*url.URL, error) {
	ref, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parse request path: %w", err)
	}
	return c.baseURL.ResolveReference(ref), nil
}

func errorsForAuth(message string) error {
	return fmt.Errorf("auth: %s", message)
}
