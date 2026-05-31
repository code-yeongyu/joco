package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/yeongyu/jocohunt/internal/jocohunt"
)

func runSubmit(ctx context.Context, client *jocohunt.Client, args []string, out io.Writer, errOut io.Writer, authFile string, baseURL string) error {
	flags := flag.NewFlagSet("submit", flag.ContinueOnError)
	flags.SetOutput(errOut)
	title := flags.String("title", "", "product title")
	productURL := flags.String("url", "", "product URL")
	tagline := flags.String("tagline", "", "short product tagline")
	description := flags.String("description", "", "product description")
	category := flags.String("category", "", "product category")
	platform := flags.String("platform", "", "target platform")
	pricing := flags.String("pricing", "", "pricing model")
	website := flags.String("website", "", "website URL")
	github := flags.String("github", "", "GitHub URL")
	endpoint := flags.String("submit-endpoint", "/api/submit", "submit API endpoint")
	sessionCookie := flags.String("session-cookie", "", "authenticated JoCoHunt Cookie header")
	csrfToken := flags.String("csrf-token", "", "CSRF token header value")
	dryRun := flags.Bool("dry-run", false, "print request plan without posting")
	confirm := flags.Bool("confirm", false, "perform the live product submission")
	jsonOut := flags.Bool("json", false, "print JSON response")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	input := jocohunt.SubmitProductInput{
		Title:       *title,
		URL:         *productURL,
		Tagline:     *tagline,
		Description: *description,
		Category:    *category,
		Platform:    *platform,
		Pricing:     *pricing,
		Website:     *website,
		GitHub:      *github,
	}
	stored, err := loadStoredSession(authFile)
	if err != nil {
		return err
	}
	storedCookie := ""
	storedToken := ""
	if sameOrigin(baseURL, stored.BaseURL) {
		storedCookie = stored.SessionCookie
		storedToken = stored.CSRFToken
	}
	cookie := firstNonBlank(*sessionCookie, os.Getenv("JOCOHUNT_SESSION_COOKIE"), storedCookie)
	token := firstNonBlank(*csrfToken, os.Getenv("JOCOHUNT_CSRF_TOKEN"), storedToken)
	if err := input.Validate(); err != nil {
		return err
	}
	if *dryRun {
		return writeSubmitPlan(out, input, *endpoint, cookie != "", token != "")
	}
	if !*confirm {
		return errors.New("live submit requires --confirm")
	}
	if strings.TrimSpace(cookie) == "" {
		return errors.New("live submit requires auth login, --session-cookie, or JOCOHUNT_SESSION_COOKIE")
	}
	result, err := client.SubmitProduct(ctx, input, jocohunt.SubmitOptions{
		Endpoint:      *endpoint,
		SessionCookie: cookie,
		CSRFToken:     token,
	})
	if err != nil {
		return err
	}
	if *jsonOut {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	if result.Location != "" {
		_, err = fmt.Fprintf(out, "Submitted: status %d location %s\n%s\n", result.StatusCode, result.Location, result.Body)
		return err
	}
	_, err = fmt.Fprintf(out, "Submitted: status %d\n%s\n", result.StatusCode, result.Body)
	return err
}

func writeSubmitPlan(out io.Writer, input jocohunt.SubmitProductInput, endpoint string, hasCookie bool, hasToken bool) error {
	plan := struct {
		Endpoint string            `json:"endpoint"`
		Payload  map[string]string `json:"payload"`
		Auth     struct {
			SessionCookie bool `json:"sessionCookie"`
			CSRFToken     bool `json:"csrfToken"`
		} `json:"auth"`
	}{
		Endpoint: endpoint,
		Payload:  input.Payload(),
	}
	plan.Auth.SessionCookie = hasCookie
	plan.Auth.CSRFToken = hasToken
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(plan)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sameOrigin(left string, right string) bool {
	leftParsed, err := url.Parse(strings.TrimSpace(left))
	if err != nil || leftParsed.Scheme == "" || leftParsed.Host == "" {
		return false
	}
	rightParsed, err := url.Parse(strings.TrimSpace(right))
	if err != nil || rightParsed.Scheme == "" || rightParsed.Host == "" {
		return false
	}
	return strings.EqualFold(leftParsed.Scheme, rightParsed.Scheme) && strings.EqualFold(leftParsed.Host, rightParsed.Host)
}
