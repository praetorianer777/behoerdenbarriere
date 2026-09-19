// Package scoring rechnet axe-Befunde in einen Score von 0 bis 100 um.
//
// Das Verfahren ist bewusst einfach nachvollziehbar: jeder Verstoß bekommt ein
// Gewicht nach seiner Schwere, die Summe wird auf die Seitengröße normiert und
// über eine Exponentialkurve auf 0..100 abgebildet.
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

// PageScore bewertet eine einzelne Seite.
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
		// Der Logarithmus dämpft die Anzahl: 50 gleichartige Verstöße sind schlimmer
		// als einer, aber nicht fünfzigmal so schlimm — sie haben dieselbe Ursache.
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

// SiteScore mittelt die Seiten einer Behörde gewichtet und liefert zusätzlich die
// vier Teilscores nach WCAG-Prinzip. Fehlgeschlagene Seiten gehen nicht ein —
// eine nicht erreichbare Seite ist kein Barrierefreiheitsbefund.
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

// Grade bildet einen Score auf eine Schulnote ab.
func Grade(score float64) string {
	for _, t := range gradeThresholds {
		if score >= t.min {
			return t.grade
		}
	}
	return "F"
}

// RuleSummary fasst gleiche Regelverstöße über alle Seiten eines Scans zusammen,
// sortiert nach Schwere und Häufigkeit — die Reihenfolge, in der eine Behörde die
// Mängel abarbeiten sollte.
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
