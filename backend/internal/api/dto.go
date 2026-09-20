package api

import (
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/maildns"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/statement"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/thirdparty"
	"github.com/praetorianer777/behoerdenbarriere/internal/trend"
)

// The API answers with types of its own. Passing database structs through would tie
// the response shape to the schema, and every column added would leak outwards.

type agencyDTO struct {
	Slug      string     `json:"slug"`
	Name      string     `json:"name"`
	URL       string     `json:"url"`
	Level     string     `json:"level"`
	State     string     `json:"state,omitempty"`
	Category  string     `json:"category,omitempty"`
	Score     *float64   `json:"score"`
	Grade     string     `json:"grade,omitempty"`
	Delta     *float64   `json:"delta,omitempty"`
	Pages     int        `json:"pages"`
	ScannedAt *time.Time `json:"scanned_at,omitempty"`
	// Lighthouse is Google's score for the entry page. It is built the other way
	// round from ours, and where the two disagree that is worth seeing.
	Lighthouse *float64 `json:"lighthouse_score,omitempty"`
	// Provisional says the value rests on too few pages to describe the site. It is
	// carried with the score rather than left to the reader to work out from Pages.
	Provisional bool `json:"provisional,omitempty"`
}

type subscoresDTO struct {
	Perceivable    *float64 `json:"perceivable"`
	Operable       *float64 `json:"operable"`
	Understandable *float64 `json:"understandable"`
	Robust         *float64 `json:"robust"`
}

// failureDTO says why the last attempt produced nothing — bot protection, an
// unreachable host. It is not a judgement about the authority.
type failureDTO struct {
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}

type agencyDetailDTO struct {
	agencyDTO
	Failure   *failureDTO   `json:"failure,omitempty"`
	Subscores subscoresDTO  `json:"subscores"`
	Mail      *mailDTO      `json:"mail,omitempty"`
	Trend     trend.Summary `json:"trend"`
	History   []trend.Point `json:"history"`
	LatestID  int64         `json:"latest_scan_id,omitempty"`
}

type listDTO struct {
	Items   []agencyDTO `json:"items"`
	Total   int         `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
}

type ruleDTO struct {
	RuleID       string `json:"rule_id"`
	Impact       string `json:"impact"`
	Principle    string `json:"principle"`
	Help         string `json:"help,omitempty"`
	HelpURL      string `json:"help_url,omitempty"`
	Pages        int    `json:"pages"`
	Nodes        int    `json:"nodes"`
	SampleHTML   string `json:"sample_html,omitempty"`
	SampleTarget string `json:"sample_target,omitempty"`
}

type pageDTO struct {
	URL        string   `json:"url"`
	Title      string   `json:"title,omitempty"`
	Depth      int      `json:"depth"`
	IsEntry    bool     `json:"is_entry"`
	Priority   bool     `json:"priority"`
	HTTPStatus int      `json:"http_status,omitempty"`
	DOMNodes   int      `json:"dom_nodes"`
	Score      *float64 `json:"score"`
	LoadMS     int      `json:"load_ms,omitempty"`
	Error      string   `json:"error,omitempty"`
	Consent    string   `json:"consent,omitempty"`
	Violations int      `json:"violations"`
}

type scanDTO struct {
	ID               int64             `json:"id"`
	AgencySlug       string            `json:"agency_slug"`
	AgencyName       string            `json:"agency_name"`
	Status           string            `json:"status"`
	StartedAt        time.Time         `json:"started_at"`
	FinishedAt       *time.Time        `json:"finished_at,omitempty"`
	Error            string            `json:"error,omitempty"`
	Score            *float64          `json:"score"`
	Grade            string            `json:"grade,omitempty"`
	Subscores        subscoresDTO      `json:"subscores"`
	PagesScanned     int               `json:"pages_scanned"`
	PagesFailed      int               `json:"pages_failed"`
	PagesBlocked     int               `json:"pages_blocked"`
	Lighthouse       *float64          `json:"lighthouse_score,omitempty"`
	LighthouseFailed []string          `json:"lighthouse_failed,omitempty"`
	Rules            []ruleDTO         `json:"rules,omitempty"`
	Pages            []pageDTO         `json:"pages,omitempty"`
	Changes          *trend.RuleChange `json:"changes,omitempty"`
	// Explanation says how the score came about: what each finding contributed and
	// what fixing it would give back.
	Explanation *scoring.SiteExplanation `json:"explanation,omitempty"`
	// Statement is the check of the legally required accessibility statement. It
	// stands beside the score, not in it.
	Statement *statement.Result `json:"statement,omitempty"`
	// ThirdParties are the outside hosts the checked pages contacted. Another
	// question about the same site, and deliberately not part of the score.
	ThirdParties []thirdparty.Observation `json:"third_parties,omitempty"`
}

// thirdPartyDTO is one third party across the country: on how many authorities it was
// seen, in which phase of a visit.
type thirdPartyDTO struct {
	Host       string `json:"host"`
	Domain     string `json:"domain"`
	Group      string `json:"group"`
	PublicBody bool   `json:"public_body"`
	Phase      string `json:"phase"`
	Agencies   int    `json:"agencies"`
	Pages      int    `json:"pages"`
}

// mailDTO is a published mail record together with the two readings that decide how it
// has to be presented: whether the receiving company is under US jurisdiction, and
// whether the host is a filter sitting in front of mailboxes that are somewhere else.
type mailDTO struct {
	maildns.Record
	USBased bool `json:"us_based"`
	Filter  bool `json:"filter"`
}

func toMailDTO(record *maildns.Record) *mailDTO {
	if record == nil {
		return nil
	}
	return &mailDTO{
		Record:  *record,
		USBased: maildns.USBased[record.Provider],
		Filter:  maildns.Filters[record.Provider],
	}
}

// mailCountDTO is one group of authorities with one mail provider.
type mailCountDTO struct {
	// Name is the state or the level; empty when the count is for the whole country.
	Name     string `json:"name,omitempty"`
	Provider string `json:"provider"`
	Agencies int    `json:"agencies"`
	// Filter says the host screens mail rather than keeping it. Where it is true, the
	// record says where mail is checked and nothing about where it lands.
	Filter bool `json:"filter"`
	// US says the provider is a company under US jurisdiction. A statement about the
	// company, not about where a server stands — and not a legal finding.
	US bool `json:"us_based"`
}

type mailSummaryDTO struct {
	Total      int            `json:"total"`
	CheckedAt  *time.Time     `json:"checked_at,omitempty"`
	ByProvider []mailCountDTO `json:"by_provider"`
	ByState    []mailCountDTO `json:"by_state"`
	ByLevel    []mailCountDTO `json:"by_level"`
}

type thirdPartyListDTO struct {
	Items []thirdPartyDTO `json:"items"`
	// Scanned is what the numbers are out of. Without it a count of twelve says
	// nothing: twelve out of twenty is a pattern, twelve out of four hundred is not.
	Scanned int `json:"scanned"`
}

type groupDTO struct {
	Name     string   `json:"name"`
	Agencies int      `json:"agencies"`
	AvgScore *float64 `json:"avg_score"`
}

type ruleCountDTO struct {
	RuleID   string `json:"rule_id"`
	Impact   string `json:"impact"`
	Agencies int    `json:"agencies"`
	Pages    int    `json:"pages"`
}

type statsDTO struct {
	Agencies  int            `json:"agencies"`
	Scanned   int            `json:"scanned"`
	AvgScore  *float64       `json:"avg_score"`
	Grades    map[string]int `json:"grades"`
	ByLevel   []groupDTO     `json:"by_level"`
	ByState   []groupDTO     `json:"by_state"`
	TopRules  []ruleCountDTO `json:"top_rules"`
	States    []string       `json:"states"`
	UpdatedAt *time.Time     `json:"updated_at,omitempty"`
}

type usageDayDTO struct {
	Day      string `json:"day"`
	Visitors int64  `json:"visitors"`
	Views    int64  `json:"views"`
}

type usageKeyDTO struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type usageDTO struct {
	Since     string        `json:"since"`
	Until     string        `json:"until"`
	Visitors  int64         `json:"visitors"`
	Views     int64         `json:"views"`
	Days      []usageDayDTO `json:"days"`
	Pages     []usageKeyDTO `json:"pages"`
	Agencies  []usageKeyDTO `json:"agencies"`
	Endpoints []usageKeyDTO `json:"endpoints"`
}

func toUsageDTO(u store.UsageSummary) usageDTO {
	dto := usageDTO{
		Since: u.Since.Format(time.DateOnly), Until: u.Until.Format(time.DateOnly),
		Visitors: u.Visitors, Views: u.Views,
		Days:      make([]usageDayDTO, 0, len(u.Days)),
		Pages:     usageKeyDTOs(u.Pages),
		Agencies:  usageKeyDTOs(u.Agencies),
		Endpoints: usageKeyDTOs(u.Endpoints),
	}
	for _, d := range u.Days {
		dto.Days = append(dto.Days, usageDayDTO{
			Day: d.Day.Format(time.DateOnly), Visitors: d.Visitors, Views: d.Views,
		})
	}
	return dto
}

func usageKeyDTOs(in []store.UsageKey) []usageKeyDTO {
	out := make([]usageKeyDTO, 0, len(in))
	for _, k := range in {
		out = append(out, usageKeyDTO{Key: k.Key, Count: k.Count})
	}
	return out
}

func toAgencyDTO(a store.AgencyListing) agencyDTO {
	dto := agencyDTO{
		Slug: a.Slug, Name: a.Name, URL: a.URL, Level: string(a.Level),
		State: a.State, Category: a.Category,
		Score: a.Score, Grade: a.Grade, Pages: a.Pages, ScannedAt: a.ScannedAt,
		Lighthouse: a.LighthouseScore,
		// Ein Wert über eine Seite ist kein Wert über eine Website. Das gehört an den
		// Wert selbst, nicht in eine Fußnote in der Detailansicht.
		Provisional: a.Score != nil && scoring.Provisional(a.Pages),
	}
	if a.Score != nil && a.PrevScore != nil {
		delta := round2(*a.Score - *a.PrevScore)
		dto.Delta = &delta
	}
	return dto
}

func toRuleDTOs(rules []scoring.RuleSummary) []ruleDTO {
	out := make([]ruleDTO, 0, len(rules))
	for _, r := range rules {
		dto := ruleDTO{
			RuleID: r.RuleID, Impact: string(r.Impact), Principle: string(r.Principle),
			Help: r.Help, HelpURL: r.HelpURL, Pages: r.Pages, Nodes: r.Nodes,
		}
		if r.Sample != nil {
			dto.SampleHTML = r.Sample.SampleHTML
			dto.SampleTarget = r.Sample.SampleTarget
		}
		out = append(out, dto)
	}
	return out
}

func toPageDTOs(pages []store.PageDetail) []pageDTO {
	out := make([]pageDTO, 0, len(pages))
	for _, p := range pages {
		out = append(out, pageDTO{
			URL: p.URL, Title: p.Title, Depth: p.Depth, IsEntry: p.IsEntry,
			Priority: p.Priority, HTTPStatus: p.HTTPStatus, DOMNodes: p.DOMNodes,
			Score: p.Score, LoadMS: p.LoadMS, Error: p.Error, Consent: p.Consent,
			Violations: p.Violations,
		})
	}
	return out
}

func toScanDTO(d store.ScanDetail) scanDTO {
	return scanDTO{
		ID: d.ID, AgencySlug: d.AgencySlug, AgencyName: d.AgencyName, Status: d.Status,
		StartedAt: d.StartedAt, FinishedAt: d.FinishedAt, Error: d.Error,
		Score: d.Score, Grade: d.Grade,
		Subscores:    subscoresDTO{d.Perceivable, d.Operable, d.Understandable, d.Robust},
		PagesScanned: d.PagesScanned, PagesFailed: d.PagesFailed, PagesBlocked: d.PagesBlocked,
		Lighthouse: d.LighthouseScore, LighthouseFailed: d.LighthouseFailed,
		Statement: d.Statement,
	}
}

func round2(v float64) float64 {
	return float64(int64(v*100+sign(v)*0.5)) / 100
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}
