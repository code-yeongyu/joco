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

const defaultSubmitEndpoint = "/api/submit"

type SubmitProductInput struct {
	Title       string
	URL         string
	Tagline     string
	Description string
	Category    string
	Platform    string
	Pricing     string
	Website     string
	GitHub      string
}

type SubmitOptions struct {
	Endpoint      string
	SessionCookie string
	CSRFToken     string
}

type SubmitResult struct {
	StatusCode int
	Body       string
	Location   string
}

func (c *Client) SubmitProduct(ctx context.Context, input SubmitProductInput, opts SubmitOptions) (SubmitResult, error) {
	if err := input.Validate(); err != nil {
		return SubmitResult{}, err
	}
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		endpoint = defaultSubmitEndpoint
	}
	payload := input.Payload()
	data, err := json.Marshal(payload)
	if err != nil {
		return SubmitResult{}, fmt.Errorf("encode submit payload: %w", err)
	}
	ref, err := parseRelativeEndpoint(endpoint)
	if err != nil {
		return SubmitResult{}, err
	}
	u := c.baseURL.ResolveReference(ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(data))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("create submit request: %w", err)
	}
	req.Header.Set("User-Agent", "jocohunt-cli/0.1")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", originForURL(c.baseURL))
	req.Header.Set("Referer", c.baseURL.ResolveReference(&url.URL{Path: "/submit"}).String())
	if cookie := strings.TrimSpace(opts.SessionCookie); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if token := strings.TrimSpace(opts.CSRFToken); token != "" {
		req.Header.Set("X-CSRF-Token", token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SubmitResult{}, fmt.Errorf("submit %s: %w", u.String(), err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("read submit response: %w", err)
	}
	result := SubmitResult{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Location:   resp.Header.Get("Location"),
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("submit %s: status %d: %s", u.String(), resp.StatusCode, strings.TrimSpace(result.Body))
	}
	return result, nil
}

func parseRelativeEndpoint(endpoint string) (*url.URL, error) {
	ref, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse submit endpoint: %w", err)
	}
	if ref.Scheme != "" || ref.Host != "" {
		return nil, errorsForSubmit("endpoint must be a relative path")
	}
	return ref, nil
}

func (input SubmitProductInput) Validate() error {
	if strings.TrimSpace(input.Title) == "" {
		return errorsForSubmit("title is required")
	}
	if strings.TrimSpace(input.URL) == "" {
		return errorsForSubmit("url is required")
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(input.URL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errorsForSubmit("url must be an absolute http(s) URL")
	}
	if strings.TrimSpace(input.Tagline) == "" {
		return errorsForSubmit("tagline is required")
	}
	return nil
}

func (input SubmitProductInput) Payload() map[string]string {
	payload := map[string]string{
		"name":    strings.TrimSpace(input.Title),
		"title":   strings.TrimSpace(input.Title),
		"url":     strings.TrimSpace(input.URL),
		"tagline": strings.TrimSpace(input.Tagline),
	}
	addOptional(payload, "description", input.Description)
	addOptional(payload, "category", input.Category)
	addOptional(payload, "platform", input.Platform)
	addOptional(payload, "pricing", input.Pricing)
	addOptional(payload, "website", input.Website)
	addOptional(payload, "github", input.GitHub)
	return payload
}

func addOptional(payload map[string]string, key string, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		payload[key] = trimmed
	}
}

func errorsForSubmit(message string) error {
	return fmt.Errorf("submit product: %s", message)
}

func originForURL(u *url.URL) string {
	origin := &url.URL{Scheme: u.Scheme, Host: u.Host}
	return origin.String()
}
