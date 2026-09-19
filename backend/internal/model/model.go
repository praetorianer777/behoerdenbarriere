// Package model holds the types shared by scanner, scoring, store and API.
package model

import "time"

type Impact string

const (
	ImpactCritical Impact = "critical"
	ImpactSerious  Impact = "serious"
	ImpactModerate Impact = "moderate"
	ImpactMinor    Impact = "minor"
)

type Principle string

const (
	Perceivable    Principle = "perceivable"
	Operable       Principle = "operable"
	Understandable Principle = "understandable"
	Robust         Principle = "robust"
)

var Principles = []Principle{Perceivable, Operable, Understandable, Robust}

type Level string

const (
	LevelBund    Level = "bund"
	LevelLand    Level = "land"
	LevelKreis   Level = "kreis"
	LevelKommune Level = "kommune"
)

type Agency struct {
	ID        int64     `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Level     Level     `json:"level"`
	State     string    `json:"state,omitempty"`
	Category  string    `json:"category,omitempty"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type Violation struct {
	RuleID       string    `json:"rule_id"`
	Impact       Impact    `json:"impact"`
	Description  string    `json:"description,omitempty"`
	Help         string    `json:"help,omitempty"`
	HelpURL      string    `json:"help_url,omitempty"`
	WCAGTags     []string  `json:"wcag_tags,omitempty"`
	Principle    Principle `json:"principle"`
	NodeCount    int       `json:"node_count"`
	SampleHTML   string    `json:"sample_html,omitempty"`
	SampleTarget string    `json:"sample_target,omitempty"`
}

// Consent says what happened to a consent layer before the page was checked. It
// belongs in the result: a page that stayed behind a banner was not really checked,
// and a clean banner must not be read as a clean site.
type Consent string

const (
	ConsentNone     Consent = "none"     // no layer found
	ConsentDeclined Consent = "declined" // declined, which is what the site must work without
	ConsentAccepted Consent = "accepted" // nothing to decline, so accepted to get to the page
	ConsentBlocked  Consent = "blocked"  // the layer stayed; what follows describes the banner
)

// PageResult is the outcome of checking a single page.
type PageResult struct {
	URL        string      `json:"url"`
	Title      string      `json:"title,omitempty"`
	Depth      int         `json:"depth"`
	IsEntry    bool        `json:"is_entry"`
	Priority   bool        `json:"priority"`
	HTTPStatus int         `json:"http_status"`
	Consent    Consent     `json:"consent,omitempty"`
	DOMNodes   int         `json:"dom_nodes"`
	LoadMS     int         `json:"load_ms"`
	Violations []Violation `json:"violations"`
	Score      float64     `json:"score"`
	Err        string      `json:"error,omitempty"`
}

func (p PageResult) Failed() bool { return p.Err != "" }
