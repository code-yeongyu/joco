package jocohunt

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
)

var routePattern = regexp.MustCompile(`href=["'](/(?:products|ideas|leaderboard|submit|sign-in|p/)[^"']*)["']`)

func (c *Client) Inspect(ctx context.Context) (InspectReport, error) {
	body, headers, err := c.get(ctx, "/")
	if err != nil {
		return InspectReport{}, err
	}
	routes := publicRoutes(body)
	report := InspectReport{
		BaseURL:         c.baseURL.String(),
		Status:          http.StatusOK,
		SecurityHeaders: securityHeaders(headers),
		PublicRoutes:    routes,
	}
	return report, nil
}

func securityHeaders(headers http.Header) map[string]bool {
	names := []string{
		"Content-Security-Policy",
		"Strict-Transport-Security",
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Permissions-Policy",
	}
	result := make(map[string]bool, len(names))
	for _, name := range names {
		result[name] = headers.Get(name) != ""
	}
	return result
}

func publicRoutes(page string) []string {
	seen := map[string]struct{}{}
	for _, match := range routePattern.FindAllStringSubmatch(page, -1) {
		seen[match[1]] = struct{}{}
	}
	routes := make([]string, 0, len(seen))
	for route := range seen {
		routes = append(routes, route)
	}
	sort.Strings(routes)
	return routes
}

func FormatSecuritySummary(report InspectReport) string {
	return fmt.Sprintf("status=%d routes=%d security_headers=%d", report.Status, len(report.PublicRoutes), countPresent(report.SecurityHeaders))
}

func countPresent(values map[string]bool) int {
	count := 0
	for _, present := range values {
		if present {
			count++
		}
	}
	return count
}
