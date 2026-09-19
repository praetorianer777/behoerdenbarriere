package api

import (
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
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
}

type subscoresDTO struct {
	Perceivable    *float64 `json:"perceivable"`
	Operable       *float64 `json:"operable"`
	Understandable *float64 `json:"understandable"`
	Robust         *float64 `json:"robust"`
}

type agencyDetailDTO struct {
	agencyDTO
	Subscores subscoresDTO  `json:"subscores"`
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
	Violations int      `json:"violations"`
}

type scanDTO struct {
	ID           int64             `json:"id"`
	AgencySlug   string            `json:"agency_slug"`
	AgencyName   string            `json:"agency_name"`
	Status       string            `json:"status"`
	StartedAt    time.Time         `json:"started_at"`
	FinishedAt   *time.Time        `json:"finished_at,omitempty"`
	Error        string            `json:"error,omitempty"`
	Score        *float64          `json:"score"`
	Grade        string            `json:"grade,omitempty"`
	Subscores    subscoresDTO      `json:"subscores"`
	PagesScanned int               `json:"pages_scanned"`
	PagesFailed  int               `json:"pages_failed"`
	Rules        []ruleDTO         `json:"rules,omitempty"`
	Pages        []pageDTO         `json:"pages,omitempty"`
	Changes      *trend.RuleChange `json:"changes,omitempty"`
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
			Score: p.Score, LoadMS: p.LoadMS, Error: p.Error, Violations: p.Violations,
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
		PagesScanned: d.PagesScanned, PagesFailed: d.PagesFailed,
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
