// Package scoring turns axe findings into a score from 0 to 100.
//
// The method is deliberately easy to follow: every violation gets a weight by its
// severity, the sum is normalized by page size and mapped onto 0..100 through an
// exponential curve.
package scoring

import (
	"math"
	"sort"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

type Result struct {
	Score      float64                     `json:"score"`
	Grade      string                      `json:"grade"`
	Principles map[model.Principle]float64 `json:"principles"`
	Pages      int                         `json:"pages"`
}

// PageScore rates a single page.
func PageScore(page model.PageResult) float64 {
	return scoreFromViolations(page.Violations, page.DOMNodes)
}

func scoreFromViolations(violations []model.Violation, domNodes int) float64 {
	raw := 0.0
	for _, v := range violations {
		nodes := v.NodeCount
		if nodes < 1 {
			nodes = 1
		}
		// The logarithm dampens the count: 50 violations of one kind are worse than one,
		// but not fifty times worse — they share a single cause.
		raw += impactWeight[v.Impact] * (1 + math.Log(float64(nodes)))
	}
	if raw == 0 {
		return 100
	}
	size := float64(domNodes)
	if size < minDOMNodes {
		size = minDOMNodes
	}
	density := raw / size * 1000
	return round2(100 * math.Exp(-density/decayK))
}

// SiteScore averages an agency's pages by weight and adds the four subscores per WCAG
// principle. Failed pages are left out — a page that would not load is an outage, not
// an accessibility finding.
func SiteScore(pages []model.PageResult) Result {
	res := Result{Principles: map[model.Principle]float64{}}

	var sum, weights float64
	counted := 0
	for _, p := range pages {
		if p.Failed() {
			continue
		}
		w := pageWeight(p)
		sum += scoreFromViolations(p.Violations, p.DOMNodes) * w
		weights += w
		counted++
	}
	res.Pages = counted
	if weights == 0 {
		return res
	}
	res.Score = round2(sum / weights)
	res.Grade = Grade(res.Score)

	for _, principle := range model.Principles {
		var psum, pweights float64
		for _, p := range pages {
			if p.Failed() {
				continue
			}
			w := pageWeight(p)
			psum += scoreFromViolations(filterPrinciple(p.Violations, principle), p.DOMNodes) * w
			pweights += w
		}
		if pweights > 0 {
			res.Principles[principle] = round2(psum / pweights)
		}
	}
	return res
}

func pageWeight(p model.PageResult) float64 {
	switch {
	case p.IsEntry:
		return weightEntry
	case p.Priority:
		return weightPriority
	default:
		return weightOther
	}
}

func filterPrinciple(violations []model.Violation, principle model.Principle) []model.Violation {
	out := make([]model.Violation, 0, len(violations))
	for _, v := range violations {
		if v.Principle == principle {
			out = append(out, v)
		}
	}
	return out
}

// Grade maps a score onto a school grade.
func Grade(score float64) string {
	for _, t := range gradeThresholds {
		if score >= t.min {
			return t.grade
		}
	}
	return "F"
}

// RuleSummary folds violations of the same rule across all pages of a scan together,
// ordered by severity and frequency — the order in which an agency should work through
// them.
type RuleSummary struct {
	RuleID    string           `json:"rule_id"`
	Impact    model.Impact     `json:"impact"`
	Help      string           `json:"help,omitempty"`
	HelpURL   string           `json:"help_url,omitempty"`
	Principle model.Principle  `json:"principle"`
	Pages     int              `json:"pages"`
	Nodes     int              `json:"nodes"`
	Sample    *model.Violation `json:"sample,omitempty"`
}

func SummarizeRules(pages []model.PageResult) []RuleSummary {
	byRule := map[string]*RuleSummary{}
	for _, p := range pages {
		if p.Failed() {
			continue
		}
		seen := map[string]bool{}
		for i, v := range p.Violations {
			s, ok := byRule[v.RuleID]
			if !ok {
				sample := p.Violations[i]
				s = &RuleSummary{
					RuleID:    v.RuleID,
					Impact:    v.Impact,
					Help:      v.Help,
					HelpURL:   v.HelpURL,
					Principle: v.Principle,
					Sample:    &sample,
				}
				byRule[v.RuleID] = s
			}
			s.Nodes += v.NodeCount
			if !seen[v.RuleID] {
				s.Pages++
				seen[v.RuleID] = true
			}
		}
	}

	out := make([]RuleSummary, 0, len(byRule))
	for _, s := range byRule {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		wi, wj := impactWeight[out[i].Impact], impactWeight[out[j].Impact]
		if wi != wj {
			return wi > wj
		}
		if out[i].Nodes != out[j].Nodes {
			return out[i].Nodes > out[j].Nodes
		}
		return out[i].RuleID < out[j].RuleID
	})
	return out
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
