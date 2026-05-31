package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const authCookieName = "better-auth.session_token"

func captureAuthSessionWithChromedp(ctx context.Context, loginURL string, baseURL string, opts authCaptureOptions) (string, error) {
	if strings.TrimSpace(loginURL) == "" {
		return "", errors.New("auth login capture requires login URL")
	}
	trimmedBase := strings.TrimSpace(baseURL)
	if trimmedBase == "" {
		return "", errors.New("auth login capture requires base URL")
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	userDataDir, err := os.MkdirTemp("", "jocohunt-auth-")
	if err != nil {
		return "", fmt.Errorf("create auth browser dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(userDataDir)
	}()

	captureCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options,
		chromedp.UserDataDir(userDataDir),
		chromedp.Flag("headless", opts.Headless),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
	)
	if strings.TrimSpace(opts.BrowserPath) != "" {
		options = append(options, chromedp.ExecPath(strings.TrimSpace(opts.BrowserPath)))
	}
	allocCtx, allocCancel := chromedp.NewExecAllocator(captureCtx, options...)
	defer allocCancel()
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	if err := chromedp.Run(browserCtx, network.Enable(), chromedp.Navigate(loginURL)); err != nil {
		return "", fmt.Errorf("open login page: %w", err)
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-browserCtx.Done():
			if err := browserCtx.Err(); err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
				return "", err
			}
			return "", fmt.Errorf("auth login capture timed out after %s", timeout)
		case <-ticker.C:
			cookies, err := getCurrentCookies(browserCtx)
			if err != nil {
				continue
			}
			for _, cookie := range cookies {
				if cookie == nil || cookie.Name != authCookieName || strings.TrimSpace(cookie.Value) == "" {
					continue
				}
				return cookie.Name + "=" + cookie.Value, nil
			}
		}
	}
}

func getCurrentCookies(ctx context.Context) ([]*network.Cookie, error) {
	var cookies []*network.Cookie
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx)
		return err
	})); err != nil {
		return nil, err
	}
	return cookies, nil
}
