package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/yeongyu/jocohunt/internal/jocohunt"
)

const defaultBaseURL = "https://jocohunt.jocoding.io"

func Run(ctx context.Context, args []string, out io.Writer, errOut io.Writer) error {
	global := flag.NewFlagSet("jocohunt", flag.ContinueOnError)
	global.SetOutput(errOut)
	baseURL := global.String("base-url", defaultBaseURL, "JoCoHunt base URL")
	timeout := global.Duration("timeout", 10*time.Second, "HTTP timeout")
	authFile := global.String("auth-file", "", "JoCoHunt auth session file")
	if err := global.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	rest := global.Args()
	if len(rest) == 0 {
		printUsage(errOut)
		return errors.New("missing command")
	}
	client, err := jocohunt.NewClient(*baseURL, *timeout)
	if err != nil {
		return err
	}
	switch rest[0] {
	case "products":
		return runItems(ctx, client, rest[1:], out, errOut, jocohunt.KindProducts)
	case "ideas":
		return runItems(ctx, client, rest[1:], out, errOut, jocohunt.KindIdeas)
	case "leaderboard":
		return runItems(ctx, client, rest[1:], out, errOut, jocohunt.KindLeaderboard)
	case "inspect":
		return runInspect(ctx, client, out)
	case "auth":
		return runAuth(ctx, client, rest[1:], out, errOut, *authFile, *baseURL)
	case "submit", "upload":
		return runSubmit(ctx, client, rest[1:], out, errOut, *authFile, *baseURL)
	default:
		fmt.Fprintf(errOut, "unknown command: %s\n", rest[0])
		printUsage(errOut)
		return fmt.Errorf("unknown command %q", rest[0])
	}
}

func runItems(ctx context.Context, client *jocohunt.Client, args []string, out io.Writer, errOut io.Writer, kind jocohunt.Kind) error {
	flags := flag.NewFlagSet(string(kind), flag.ContinueOnError)
	flags.SetOutput(errOut)
	jsonOut := flags.Bool("json", false, "print JSON")
	limit := flags.Int("limit", 20, "maximum items")
	category := flags.String("category", "", "product category")
	search := flags.String("q", "", "product search query")
	tab := flags.String("tab", "", "ideas tab")
	period := flags.String("period", "weekly", "leaderboard period")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	items, err := client.ListItems(ctx, jocohunt.Query{
		Kind:     kind,
		Limit:    *limit,
		Category: *category,
		Search:   *search,
		Tab:      *tab,
		Period:   *period,
	})
	if err != nil {
		return err
	}
	return writeItems(out, items, *jsonOut)
}

func runInspect(ctx context.Context, client *jocohunt.Client, out io.Writer) error {
	report, err := client.Inspect(ctx)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeItems(out io.Writer, items []jocohunt.Item, jsonOut bool) error {
	if jsonOut {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(items)
	}
	if len(items) == 0 {
		_, err := fmt.Fprintln(out, "No items found")
		return err
	}
	for _, item := range items {
		line := strings.TrimSpace(fmt.Sprintf("%d. %s - %s", item.Position, item.Title, item.Author))
		if _, err := fmt.Fprintln(out, line); err != nil {
			return fmt.Errorf("write item: %w", err)
		}
	}
	return nil
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "usage: jocohunt [--base-url URL] [--timeout DURATION] <products|ideas|leaderboard|inspect|auth|submit> [flags]")
}
