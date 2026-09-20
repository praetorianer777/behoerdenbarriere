// Package trend reads an authority's history out of its scans: has it improved, how
// fast, and what that pace would mean if it held.
//
// Pure functions over the scan list, so the arithmetic can be checked without a
// database.
package trend

import (
	"math"
	"sort"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
)

// Point is one finished scan.
type Point struct {
	At    time.Time `json:"at"`
	Score float64   `json:"score"`
	Grade string    `json:"grade"`
}

// Direction says which way the site has moved since the previous scan.
type Direction string

const (
	Improved  Direction = "improved"
	Declined  Direction = "declined"
	Unchanged Direction = "unchanged"
	Unknown   Direction = "unknown"
)

// A score moves a little between two scans without anything having changed on the
// site: a rotating teaser, an ad, a different page caught by the crawl. Below this
// many points, nothing is claimed.
const noiseFloor = 0.5

// Summary is what the ranking and the detail page show about the history.
type Summary struct {
	Latest    *Point    `json:"latest,omitempty"`
	Previous  *Point    `json:"previous,omitempty"`
	Direction Direction `json:"direction"`

	// Change since the previous scan, and over the usual windows. nil means there is
	// no scan old enough to compare against — which is not the same as no change.
	DeltaLast *float64 `json:"delta_last,omitempty"`
	Delta30   *float64 `json:"delta_30d,omitempty"`
	Delta90   *float64 `json:"delta_90d,omitempty"`
	Delta365  *float64 `json:"delta_365d,omitempty"`

	// PointsPerMonth is the slope of a straight line through the scans of the last
	// year. Several scans say more about the pace than the difference between the
	// last two, which a single bad crawl can dominate.
	PointsPerMonth *float64 `json:"points_per_month,omitempty"`

	// MonthsToGradeA extrapolates that pace. Only set when the trend points upwards
	// and the grade has not been reached yet — anything else would be a promise the
	// data does not carry.
	MonthsToGradeA *float64 `json:"months_to_grade_a,omitempty"`

	Scans int `json:"scans"`
}

// Summarize expects the finished scans of one authority in any order.
func Summarize(points []Point, now time.Time) Summary {
	sorted := append([]Point(nil), points...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].At.Before(sorted[j].At) })

	s := Summary{Direction: Unknown, Scans: len(sorted)}
	if len(sorted) == 0 {
		return s
	}

	latest := sorted[len(sorted)-1]
	s.Latest = &latest

	if len(sorted) > 1 {
		previous := sorted[len(sorted)-2]
		s.Previous = &previous
		delta := round2(latest.Score - previous.Score)
		s.DeltaLast = &delta
		s.Direction = direction(delta)
	}

	s.Delta30 = deltaOver(sorted, latest, now, 30*24*time.Hour)
	s.Delta90 = deltaOver(sorted, latest, now, 90*24*time.Hour)
	s.Delta365 = deltaOver(sorted, latest, now, 365*24*time.Hour)

	if slope, ok := pointsPerMonth(sorted, now); ok {
		s.PointsPerMonth = &slope
		if months, ok := monthsToGradeA(latest.Score, slope); ok {
			s.MonthsToGradeA = &months
		}
	}
	return s
}

func direction(delta float64) Direction {
	switch {
	case delta > noiseFloor:
		return Improved
	case delta < -noiseFloor:
		return Declined
	default:
		return Unchanged
	}
}

// deltaOver compares the latest score with the last one taken before the window
// opened — the oldest scan still inside it would compare against a point the window
// was meant to exclude.
func deltaOver(sorted []Point, latest Point, now time.Time, window time.Duration) *float64 {
	cutoff := now.Add(-window)

	var reference *Point
	for i := range sorted {
		if sorted[i].At.After(cutoff) {
			break
		}
		reference = &sorted[i]
	}
	if reference == nil || reference.At.Equal(latest.At) {
		return nil
	}
	delta := round2(latest.Score - reference.Score)
	return &delta
}

// pointsPerMonth fits a straight line through the scans of the last year by least
// squares. Fewer than three scans, or all of them on one day, say nothing about a
// pace.
func pointsPerMonth(sorted []Point, now time.Time) (float64, bool) {
	cutoff := now.Add(-365 * 24 * time.Hour)
	window := make([]Point, 0, len(sorted))
	for _, p := range sorted {
		if !p.At.Before(cutoff) {
			window = append(window, p)
		}
	}
	if len(window) < 3 {
		return 0, false
	}

	base := window[0].At
	var sumX, sumY, sumXY, sumXX float64
	for _, p := range window {
		x := p.At.Sub(base).Hours() / 24 / 30.44
		sumX += x
		sumY += p.Score
		sumXY += x * p.Score
		sumXX += x * x
	}
	n := float64(len(window))
	denominator := n*sumXX - sumX*sumX
	if math.Abs(denominator) < 1e-9 {
		return 0, false
	}
	return round2((n*sumXY - sumX*sumY) / denominator), true
}

// monthsToGradeA extrapolates the pace. It answers only when the trend points upwards
// and the grade is not reached yet; a declining or flat site would otherwise get a
// date it is not heading for.
func monthsToGradeA(score, pointsPerMonth float64) (float64, bool) {
	const gradeA = 90.0
	if score >= gradeA || pointsPerMonth <= 0 {
		return 0, false
	}
	return round2((gradeA - score) / pointsPerMonth), true
}

// RuleChange is what changed between two scans, per rule.
type RuleChange struct {
	Fixed      []scoring.RuleSummary `json:"fixed"`
	Introduced []scoring.RuleSummary `json:"introduced"`
	Remaining  []scoring.RuleSummary `json:"remaining"`
}

// CompareRules puts the rules of the previous scan against those of the current one.
// Which rules an authority has got rid of says more about its work than the score,
// which also moves when the crawl catches different pages.
func CompareRules(previous, current []scoring.RuleSummary) RuleChange {
	before := index(previous)
	after := index(current)

	change := RuleChange{
		Fixed:      []scoring.RuleSummary{},
		Introduced: []scoring.RuleSummary{},
		Remaining:  []scoring.RuleSummary{},
	}
	for _, rule := range previous {
		if _, still := after[rule.RuleID]; !still {
			change.Fixed = append(change.Fixed, rule)
		}
	}
	for _, rule := range current {
		if _, had := before[rule.RuleID]; had {
			change.Remaining = append(change.Remaining, rule)
		} else {
			change.Introduced = append(change.Introduced, rule)
		}
	}
	return change
}

func index(rules []scoring.RuleSummary) map[string]scoring.RuleSummary {
	out := make(map[string]scoring.RuleSummary, len(rules))
	for _, r := range rules {
		out[r.RuleID] = r
	}
	return out
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
