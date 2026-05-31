package jocohunt

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
)

var jsonLDScriptPattern = regexp.MustCompile(`(?is)<script[^>]+type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
var productAnchorPattern = regexp.MustCompile(`(?is)<a[^>]+href=["'](/p/[^"']+)["'][^>]*>(.*?)</a>`)
var metaRefreshPattern = regexp.MustCompile(`(?is)<meta[^>]+http-equiv=["']refresh["'][^>]+content=["'][^"']*url=([^"';]+)[^"']*["']`)

func parseItems(page string) ([]Item, error) {
	matches := jsonLDScriptPattern.FindAllStringSubmatch(page, -1)
	for _, match := range matches {
		items, err := parseJSONLD(html.UnescapeString(strings.TrimSpace(match[1])))
		if err == nil {
			return items, nil
		}
	}
	if items := parseProductAnchors(page); len(items) > 0 {
		return items, nil
	}
	return nil, fmt.Errorf("no collection items found")
}

func parseJSONLD(raw string) ([]Item, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("decode json-ld: %w", err)
	}
	elements, ok := itemListElements(payload)
	if !ok {
		return nil, fmt.Errorf("missing itemListElement")
	}
	items := make([]Item, 0, len(elements))
	for _, element := range elements {
		item, ok := parseListElement(element)
		if ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func itemListElements(payload map[string]any) ([]any, bool) {
	mainEntity, ok := payload["mainEntity"].(map[string]any)
	if !ok {
		return nil, false
	}
	elements, ok := mainEntity["itemListElement"].([]any)
	return elements, ok
}

func parseListElement(raw any) (Item, bool) {
	element, ok := raw.(map[string]any)
	if !ok {
		return Item{}, false
	}
	nested, _ := element["item"].(map[string]any)
	item := Item{
		Position:    int(numberValue(element["position"])),
		URL:         stringValue(element["url"]),
		Title:       firstNonEmpty(stringValue(nested["name"]), stringValue(nested["headline"])),
		Description: firstNonEmpty(stringValue(nested["description"]), stringValue(nested["text"])),
	}
	if author, ok := nested["author"].(map[string]any); ok {
		item.Author = stringValue(author["name"])
	}
	if item.Title == "" {
		return Item{}, false
	}
	return item, true
}

func stringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func numberValue(value any) float64 {
	number, ok := value.(float64)
	if !ok {
		return 0
	}
	return number
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parseProductAnchors(page string) []Item {
	matches := productAnchorPattern.FindAllStringSubmatch(page, -1)
	items := make([]Item, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		urlPath := html.UnescapeString(match[1])
		if _, ok := seen[urlPath]; ok {
			continue
		}
		title := cleanText(match[2])
		if title == "" {
			continue
		}
		seen[urlPath] = struct{}{}
		items = append(items, Item{
			Position: len(items) + 1,
			Title:    title,
			URL:      "https://jocohunt.jocoding.io" + urlPath,
		})
	}
	return items
}

func cleanText(raw string) string {
	withoutTags := regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(raw, "")
	return strings.TrimSpace(html.UnescapeString(withoutTags))
}

func metaRefreshPath(page string) string {
	match := metaRefreshPattern.FindStringSubmatch(page)
	if len(match) != 2 {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(match[1]))
}
