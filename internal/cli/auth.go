package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/yeongyu/jocohunt/internal/jocohunt"
)

type authCaptureOptions struct {
	BrowserPath string
	Headless    bool
	Timeout     time.Duration
}

var captureAuthSessionFromBrowser = captureAuthSessionWithChromedp

func runAuth(ctx context.Context, client *jocohunt.Client, args []string, out io.Writer, errOut io.Writer, authFile string, baseURL string) error {
	if len(args) == 0 {
		printAuthUsage(errOut)
		return errors.New("missing auth command")
	}
	switch args[0] {
	case "login":
		return runAuthLogin(ctx, client, args[1:], out, errOut, authFile, baseURL)
	case "status":
		return runAuthStatus(ctx, client, args[1:], out, errOut, authFile, baseURL)
	case "logout":
		return runAuthLogout(args[1:], out, errOut, authFile)
	default:
		printAuthUsage(errOut)
		return fmt.Errorf("unknown auth command %q", args[0])
	}
}

func runAuthLogin(ctx context.Context, client *jocohunt.Client, args []string, out io.Writer, errOut io.Writer, authFile string, baseURL string) error {
	flags := flag.NewFlagSet("auth login", flag.ContinueOnError)
	flags.SetOutput(errOut)
	sessionCookie := flags.String("session-cookie", "", "authenticated JoCoHunt Cookie header to store")
	csrfToken := flags.String("csrf-token", "", "CSRF token to store")
	callbackPath := flags.String("callback", "/submit", "post-login callback path")
	printURL := flags.Bool("print-url", false, "print GitHub OAuth URL")
	noOpen := flags.Bool("no-open", false, "do not open the system browser")
	browserPath := flags.String("browser", "", "Chrome/Chromium executable path for login capture")
	headless := flags.Bool("headless", false, "run the login browser in headless mode")
	captureTimeout := flags.Duration("capture-timeout", 5*time.Minute, "max time to wait for auth cookie capture")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if strings.TrimSpace(*sessionCookie) != "" {
		path, err := sessionPath(authFile)
		if err != nil {
			return err
		}
		err = jocohunt.SaveAuthSession(path, jocohunt.AuthSession{
			BaseURL:       baseURL,
			SessionCookie: *sessionCookie,
			CSRFToken:     *csrfToken,
		})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "Logged in. Stored JoCoHunt session at %s\n", path)
		return err
	}
	login, err := client.StartGitHubLogin(ctx, *callbackPath)
	if err != nil {
		return err
	}
	if *printURL || *noOpen {
		_, err = fmt.Fprintf(out, "%s\nsession not saved; finish GitHub login in the browser, then run `jocohunt auth login --session-cookie ...`.\n", login.URL)
		return err
	}
	cookie, err := captureAuthSessionFromBrowser(ctx, login.URL, baseURL, authCaptureOptions{
		BrowserPath: *browserPath,
		Headless:    *headless,
		Timeout:     *captureTimeout,
	})
	if err == nil && strings.TrimSpace(cookie) != "" {
		path, err := sessionPath(authFile)
		if err != nil {
			return err
		}
		err = jocohunt.SaveAuthSession(path, jocohunt.AuthSession{
			BaseURL:       baseURL,
			SessionCookie: cookie,
			CSRFToken:     *csrfToken,
		})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "Logged in. Stored JoCoHunt session at %s\n", path)
		return err
	}
	if err := openBrowser(login.URL); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Opened GitHub login URL.\nAfter GitHub finishes, store the JoCoHunt session with `jocohunt auth login --session-cookie ...`.\n%s\n", login.URL)
	return err
}

func runAuthStatus(ctx context.Context, client *jocohunt.Client, args []string, out io.Writer, errOut io.Writer, authFile string, baseURL string) error {
	flags := flag.NewFlagSet("auth status", flag.ContinueOnError)
	flags.SetOutput(errOut)
	verify := flags.Bool("verify", false, "verify the stored session with the server")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	session, err := loadStoredSession(authFile)
	if err != nil {
		return err
	}
	if strings.TrimSpace(session.SessionCookie) == "" {
		_, err = fmt.Fprintln(out, "Not logged in")
		return err
	}
	if *verify {
		if !sameOrigin(baseURL, session.BaseURL) {
			_, err = fmt.Fprintln(out, "Not logged in")
			return err
		}
		status, err := client.VerifySession(ctx, session.SessionCookie)
		if err != nil {
			return err
		}
		if status.Authenticated {
			_, err = fmt.Fprintln(out, "Verified (session accepted)")
			return err
		}
		_, err = fmt.Fprintln(out, "Not logged in")
		return err
	}
	if strings.TrimSpace(session.CSRFToken) != "" {
		_, err = fmt.Fprintln(out, "Logged in (session cookie + csrf token stored)")
		return err
	}
	_, err = fmt.Fprintln(out, "Logged in (session cookie stored)")
	return err
}

func runAuthLogout(args []string, out io.Writer, errOut io.Writer, authFile string) error {
	flags := flag.NewFlagSet("auth logout", flag.ContinueOnError)
	flags.SetOutput(errOut)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	path, err := deleteStoredSession(authFile)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Logged out. Removed JoCoHunt session at %s\n", path)
	return err
}

func printAuthUsage(out io.Writer) {
	fmt.Fprintln(out, "usage: jocohunt auth <login|status|logout> [flags]")
}

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
