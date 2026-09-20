package scoring

import (
	"math"
	"sort"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

// A score that does not say what it is made of is an assertion. These functions take
// the same numbers apart again: what each finding contributed, and what fixing it
// would give back.

// Reason is one finding's part in a page's score.
type Reason struct {
	RuleID    string          `json:"rule_id"`
	Impact    model.Impact    `json:"impact"`
	Principle model.Principle `json:"principle"`
	Help      string          `json:"help,omitempty"`
	HelpURL   string          `json:"help_url,omitempty"`
	Nodes     int             `json:"nodes"`

	// Weight is what the severity is worth, Penalty what this finding added to the
	// page's burden, and Share its part of the whole burden. The shares of a page add
	// up to one; they say where the weight sits.
	Weight  float64 `json:"weight"`
	Penalty float64 `json:"penalty"`
	Share   float64 `json:"share"`

	// PointsIfFixed is what the page would gain if this one finding were gone. It is
	// not the share converted into points: the curve is not linear, so fixing two
	// findings gains more than the two gains added up. This is the number to act on.
	PointsIfFixed float64 `json:"points_if_fixed"`
}

// PageExplanation is one page's score with its reasons.
type PageExplanation struct {
	URL      string   `json:"url"`
	Title    string   `json:"title,omitempty"`
	Score    float64  `json:"score"`
	DOMNodes int      `json:"dom_nodes"`
	Density  float64  `json:"density"`
	Weight   float64  `json:"weight"`
	IsEntry  bool     `json:"is_entry"`
	Priority bool     `json:"priority"`
	Reasons  []Reason `json:"reasons"`
}

// ExplainPage breaks a page's score into its reasons, heaviest first.
func ExplainPage(page model.PageResult) PageExplanation {
	out := PageExplanation{
		URL:      page.URL,
		Title:    page.Title,
		DOMNodes: page.DOMNodes,
		Weight:   pageWeight(page),
		IsEntry:  page.IsEntry,
		Priority: page.Priority,
		Score:    scoreFromViolations(page.Violations, page.DOMNodes),
		Reasons:  []Reason{},
	}
	if page.Failed() {
		return out
	}

	size := float64(page.DOMNodes)
	if size < minDOMNodes {
		size = minDOMNodes
	}

	var total float64
	penalties := make([]float64, len(page.Violations))
	for i, v := range page.Violations {
		penalties[i] = penaltyOf(v)
		total += penalties[i]
	}
	out.Density = round2(total / size * 1000)

	for i, v := range page.Violations {
		share := 0.0
		if total > 0 {
			share = penalties[i] / total
		}
		// What the page would score without this one finding, against what it scores
		// now: the gain that fixing it brings.
		without := scoreFromDensity((total - penalties[i]) / size * 1000)
		out.Reasons = append(out.Reasons, Reason{
			RuleID:        v.RuleID,
			Impact:        v.Impact,
			Principle:     v.Principle,
			Help:          v.Help,
			HelpURL:       v.HelpURL,
			Nodes:         nodeCount(v),
			Weight:        impactWeight[v.Impact],
			Penalty:       round2(penalties[i]),
			Share:         math.Round(share*1000) / 1000,
			PointsIfFixed: round2(without - out.Score),
		})
	}

	sort.Slice(out.Reasons, func(i, j int) bool {
		if out.Reasons[i].PointsIfFixed != out.Reasons[j].PointsIfFixed {
			return out.Reasons[i].PointsIfFixed > out.Reasons[j].PointsIfFixed
		}
		return out.Reasons[i].RuleID < out.Reasons[j].RuleID
	})
	return out
}

// Improvement is what one rule would give back across the whole site.
type Improvement struct {
	RuleID        string          `json:"rule_id"`
	Impact        model.Impact    `json:"impact"`
	Principle     model.Principle `json:"principle"`
	Help          string          `json:"help,omitempty"`
	HelpURL       string          `json:"help_url,omitempty"`
	Pages         int             `json:"pages"`
	Nodes         int             `json:"nodes"`
	PointsIfFixed float64         `json:"points_if_fixed"`
}

// SiteExplanation says how an agency's score came about.
type SiteExplanation struct {
	Score        float64           `json:"score"`
	Grade        string            `json:"grade"`
	Pages        []PageExplanation `json:"pages"`
	Improvements []Improvement     `json:"improvements"`
}

// ExplainSite breaks the agency score down by page and ranks what to fix first.
//
// The ranking answers one question: where does the first afternoon of work help most?
// So a rule's worth is measured against the agency's score, not against a single
// page's — a finding on the entry page counts three times over, and one that appears
// on eight pages is worth more than one that appears on one.
func ExplainSite(pages []model.PageResult) SiteExplanation {
	result := SiteScore(pages)
	out := SiteExplanation{
		Score:        result.Score,
		Grade:        result.Grade,
		Pages:        []PageExplanation{},
		Improvements: []Improvement{},
	}

	var weights float64
	for _, page := range pages {
		if page.Failed() {
			continue
		}
		out.Pages = append(out.Pages, ExplainPage(page))
		weights += pageWeight(page)
	}
	if weights == 0 {
		return out
	}

	type aggregate struct {
		Improvement
		gain float64
	}
	byRule := map[string]*aggregate{}

	for _, explained := range out.Pages {
		seen := map[string]bool{}
		for _, reason := range explained.Reasons {
			entry, ok := byRule[reason.RuleID]
			if !ok {
				entry = &aggregate{Improvement: Improvement{
					RuleID:    reason.RuleID,
					Impact:    reason.Impact,
					Principle: reason.Principle,
					Help:      reason.Help,
					HelpURL:   reason.HelpURL,
				}}
				byRule[reason.RuleID] = entry
			}
			entry.Nodes += reason.Nodes
			if !seen[reason.RuleID] {
				entry.Pages++
				seen[reason.RuleID] = true
			}
			// The page's gain counts towards the agency in the proportion the page
			// carries in the weighted mean.
			entry.gain += reason.PointsIfFixed * explained.Weight / weights
		}
	}

	for _, entry := range byRule {
		entry.PointsIfFixed = round2(entry.gain)
		out.Improvements = append(out.Improvements, entry.Improvement)
	}
	sort.Slice(out.Improvements, func(i, j int) bool {
		if out.Improvements[i].PointsIfFixed != out.Improvements[j].PointsIfFixed {
			return out.Improvements[i].PointsIfFixed > out.Improvements[j].PointsIfFixed
		}
		return out.Improvements[i].RuleID < out.Improvements[j].RuleID
	})
	return out
}
