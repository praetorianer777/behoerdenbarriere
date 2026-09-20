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

// Source says where an entry came from. It decides who may overwrite it: the
// hand-kept list has been checked, an import has not.
type Source string

const (
	SourceSeed     Source = "seed"
	SourceWikidata Source = "wikidata"
)

type Agency struct {
	ID         int64     `json:"id"`
	Slug       string    `json:"slug"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	Level      Level     `json:"level"`
	State      string    `json:"state,omitempty"`
	Category   string    `json:"category,omitempty"`
	Active     bool      `json:"active"`
	Source     Source    `json:"source,omitempty"`
	ExternalID string    `json:"external_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
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

// ContactPhase says when during a visit a request went out. The phase is the whole
// point of recording contacts at all: a tracker that fires on page load is a different
// matter from one that loads because someone agreed to it.
type ContactPhase string

const (
	// PhaseBeforeConsent covers everything up to the moment a consent layer was
	// answered — and everything on a page that never asked, or whose layer stayed.
	PhaseBeforeConsent ContactPhase = "before_consent"
	PhaseAfterDeclined ContactPhase = "after_declined"
	PhaseAfterAccepted ContactPhase = "after_accepted"
)

// Contact is one host a page reached out to in one phase, and how often. Only hosts
// outside the page's own registrable domain are kept: the site loading its own assets
// says nothing about anyone's data leaving.
//
// What is recorded is the host name, not a verdict. Who ends up holding the data is a
// question the host name alone cannot answer.
type Contact struct {
	Host     string       `json:"host"`
	Phase    ContactPhase `json:"phase"`
	Requests int          `json:"requests"`
}

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
	Contacts   []Contact   `json:"contacts,omitempty"`
	// Text is the visible text of the page. It is carried through a scan so the
	// accessibility statement can be read, and deliberately not stored: we keep
	// findings about pages, not copies of them.
	Text  string  `json:"-"`
	Score float64 `json:"score"`
	Err   string  `json:"error,omitempty"`
}

func (p PageResult) Failed() bool { return p.Err != "" }
