package scanner

import (
	"strings"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
)

// axeResult bildet den Teil der axe.run-Antwort ab, den wir auswerten.
type axeResult struct {
	Violations []axeViolation `json:"violations"`
}

type axeViolation struct {
	ID          string   `json:"id"`
	Impact      string   `json:"impact"`
	Description string   `json:"description"`
	Help        string   `json:"help"`
	HelpURL     string   `json:"helpUrl"`
	Tags        []string `json:"tags"`
	NodeCount   int      `json:"nodeCount"`
	Sample      *axeNode `json:"sample"`
}

type axeNode struct {
	HTML   string   `json:"html"`
	Target []string `json:"target"`
}

const maxSampleHTML = 500

func toViolations(res axeResult) []model.Violation {
	out := make([]model.Violation, 0, len(res.Violations))
	for _, v := range res.Violations {
		impact := normalizeImpact(v.Impact)
		mv := model.Violation{
			RuleID:      v.ID,
			Impact:      impact,
			Description: v.Description,
			Help:        v.Help,
			HelpURL:     v.HelpURL,
			WCAGTags:    v.Tags,
			Principle:   scoring.Principle(v.Tags),
			NodeCount:   v.NodeCount,
		}
		if v.Sample != nil {
			mv.SampleHTML = truncate(v.Sample.HTML, maxSampleHTML)
			mv.SampleTarget = strings.Join(v.Sample.Target, " ")
		}
		if mv.NodeCount < 1 {
			mv.NodeCount = 1
		}
		out = append(out, mv)
	}
	return out
}

// axe lässt impact bei einzelnen Regeln leer; unbewertet einzustufen wäre dasselbe
// wie zu ignorieren, deshalb gilt die mittlere Stufe.
func normalizeImpact(raw string) model.Impact {
	switch model.Impact(raw) {
	case model.ImpactCritical:
		return model.ImpactCritical
	case model.ImpactSerious:
		return model.ImpactSerious
	case model.ImpactModerate:
		return model.ImpactModerate
	case model.ImpactMinor:
		return model.ImpactMinor
	default:
		return model.ImpactModerate
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
