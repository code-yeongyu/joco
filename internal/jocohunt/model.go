package jocohunt

import "errors"

type Kind string

const (
	KindProducts    Kind = "products"
	KindIdeas       Kind = "ideas"
	KindLeaderboard Kind = "leaderboard"
)

var ErrInvalidKind = errors.New("jocohunt: invalid kind")

type Query struct {
	Kind     Kind
	Limit    int
	Category string
	Search   string
	Tab      string
	Period   string
}

type Item struct {
	Position    int    `json:"position"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
}

type InspectReport struct {
	BaseURL         string          `json:"baseUrl"`
	Status          int             `json:"status"`
	SecurityHeaders map[string]bool `json:"securityHeaders"`
	PublicRoutes    []string        `json:"publicRoutes"`
}
