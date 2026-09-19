package trend

import (
	"math"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
)

var now = time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

func daysAgo(d int, score float64) Point {
	return Point{At: now.AddDate(0, 0, -d), Score: score, Grade: scoring.Grade(score)}
}

func TestSummarizeWithoutScans(t *testing.T) {
	got := Summarize(nil, now)
	if got.Latest != nil || got.Direction != Unknown || got.Scans != 0 {
		t.Fatalf("summary = %+v", got)
	}
}

// A single scan is a state, not a trend. Nothing may be claimed about a direction.
func TestSummarizeWithOneScan(t *testing.T) {
	got := Summarize([]Point{daysAgo(3, 71)}, now)
	if got.Latest == nil || got.Latest.Score != 71 {
		t.Fatalf("latest = %+v", got.Latest)
	}
	if got.Previous != nil || got.DeltaLast != nil || got.Direction != Unknown {
		t.Fatalf("a direction was claimed: %+v", got)
	}
	if got.PointsPerMonth != nil {
		t.Fatalf("a pace was claimed: %v", *got.PointsPerMonth)
	}
}

func TestSummarizeSortsByTime(t *testing.T) {
	got := Summarize([]Point{daysAgo(1, 80), daysAgo(40, 60), daysAgo(20, 70)}, now)
	if got.Latest.Score != 80 || got.Previous.Score != 70 {
		t.Fatalf("latest = %v, previous = %v", got.Latest.Score, got.Previous.Score)
	}
}

func TestDirection(t *testing.T) {
	cases := []struct {
		name   string
		points []Point
		want   Direction
	}{
		{"improved", []Point{daysAgo(30, 60), daysAgo(1, 72)}, Improved},
		{"declined", []Point{daysAgo(30, 72), daysAgo(1, 60)}, Declined},
		{"unchanged", []Point{daysAgo(30, 70), daysAgo(1, 70)}, Unchanged},
		// A tenth of a point is a rotating teaser, not work on the site.
		{"noise counts as unchanged", []Point{daysAgo(30, 70), daysAgo(1, 70.3)}, Unchanged},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Summarize(c.points, now).Direction; got != c.want {
				t.Fatalf("direction = %s, want %s", got, c.want)
			}
		})
	}
}

func TestDeltaWindows(t *testing.T) {
	got := Summarize([]Point{
		daysAgo(400, 40),
		daysAgo(100, 50),
		daysAgo(40, 60),
		daysAgo(1, 75),
	}, now)

	if got.Delta30 == nil || *got.Delta30 != 15 {
		t.Errorf("delta 30d = %v, want 15", value(got.Delta30))
	}
	if got.Delta90 == nil || *got.Delta90 != 25 {
		t.Errorf("delta 90d = %v, want 25", value(got.Delta90))
	}
	if got.Delta365 == nil || *got.Delta365 != 35 {
		t.Errorf("delta 365d = %v, want 35", value(got.Delta365))
	}
}

// Without a scan from before the window, the change over it is unknown — and that is
// not the same as no change.
func TestDeltaWindowWithoutOlderScan(t *testing.T) {
	got := Summarize([]Point{daysAgo(10, 60), daysAgo(1, 70)}, now)
	if got.Delta30 != nil {
		t.Fatalf("delta 30d = %v although nothing is that old", *got.Delta30)
	}
	if got.DeltaLast == nil || *got.DeltaLast != 10 {
		t.Fatalf("delta to the previous scan = %v", value(got.DeltaLast))
	}
}

func TestPointsPerMonth(t *testing.T) {
	// Five points a month over four months.
	got := Summarize([]Point{
		daysAgo(91, 50), daysAgo(61, 55), daysAgo(30, 60), daysAgo(0, 65),
	}, now)
	if got.PointsPerMonth == nil {
		t.Fatal("no pace computed")
	}
	if math.Abs(*got.PointsPerMonth-5) > 0.3 {
		t.Fatalf("pace = %v, want about 5", *got.PointsPerMonth)
	}
}

// Two scans are not a pace: one bad crawl would set the tempo for the whole year.
func TestPointsPerMonthNeedsThreeScans(t *testing.T) {
	got := Summarize([]Point{daysAgo(30, 60), daysAgo(1, 70)}, now)
	if got.PointsPerMonth != nil {
		t.Fatalf("pace from two scans: %v", *got.PointsPerMonth)
	}
}

func TestPointsPerMonthIgnoresScansOlderThanAYear(t *testing.T) {
	got := Summarize([]Point{
		daysAgo(500, 10), daysAgo(400, 20),
		daysAgo(60, 70), daysAgo(30, 70), daysAgo(1, 70),
	}, now)
	if got.PointsPerMonth == nil {
		t.Fatal("no pace computed")
	}
	if math.Abs(*got.PointsPerMonth) > 0.1 {
		t.Fatalf("pace = %v — the old scans pulled it along", *got.PointsPerMonth)
	}
}

func TestProjectionOnlyWhenImproving(t *testing.T) {
	improving := Summarize([]Point{daysAgo(60, 60), daysAgo(30, 65), daysAgo(0, 70)}, now)
	if improving.MonthsToGradeA == nil {
		t.Fatal("no projection although the site is improving")
	}
	// 20 points at 5 a month.
	if math.Abs(*improving.MonthsToGradeA-4) > 0.3 {
		t.Fatalf("projection = %v months, want about 4", *improving.MonthsToGradeA)
	}

	declining := Summarize([]Point{daysAgo(60, 70), daysAgo(30, 65), daysAgo(0, 60)}, now)
	if declining.MonthsToGradeA != nil {
		t.Fatalf("a declining site got a date: %v", *declining.MonthsToGradeA)
	}

	arrived := Summarize([]Point{daysAgo(60, 90), daysAgo(30, 93), daysAgo(0, 95)}, now)
	if arrived.MonthsToGradeA != nil {
		t.Fatalf("a site already at grade A got a date: %v", *arrived.MonthsToGradeA)
	}
}

func rule(id string, impact model.Impact) scoring.RuleSummary {
	return scoring.RuleSummary{RuleID: id, Impact: impact, Pages: 1, Nodes: 1}
}

func TestCompareRules(t *testing.T) {
	previous := []scoring.RuleSummary{
		rule("image-alt", model.ImpactCritical),
		rule("color-contrast", model.ImpactSerious),
	}
	current := []scoring.RuleSummary{
		rule("color-contrast", model.ImpactSerious),
		rule("label", model.ImpactCritical),
	}

	got := CompareRules(previous, current)
	if len(got.Fixed) != 1 || got.Fixed[0].RuleID != "image-alt" {
		t.Errorf("fixed = %v", ids(got.Fixed))
	}
	if len(got.Introduced) != 1 || got.Introduced[0].RuleID != "label" {
		t.Errorf("introduced = %v", ids(got.Introduced))
	}
	if len(got.Remaining) != 1 || got.Remaining[0].RuleID != "color-contrast" {
		t.Errorf("remaining = %v", ids(got.Remaining))
	}
}

// The first scan has nothing to compare against: everything it finds is simply there,
// not newly introduced by the authority.
func TestCompareRulesAgainstNoPreviousScan(t *testing.T) {
	got := CompareRules(nil, []scoring.RuleSummary{rule("label", model.ImpactCritical)})
	if len(got.Introduced) != 1 || len(got.Fixed) != 0 {
		t.Fatalf("change = %+v", got)
	}
}

func value(p *float64) any {
	if p == nil {
		return "nil"
	}
	return *p
}

func ids(rules []scoring.RuleSummary) []string {
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		out = append(out, r.RuleID)
	}
	return out
}
