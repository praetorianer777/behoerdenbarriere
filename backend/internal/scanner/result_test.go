package scanner

import (
	"strings"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

func TestToViolationsMapsFields(t *testing.T) {
	got := toViolations(axeResult{Violations: []axeViolation{{
		ID:          "image-alt",
		Impact:      "critical",
		Description: "Images must have alternate text",
		Help:        "Bilder brauchen eine Alternative",
		HelpURL:     "https://dequeuniversity.com/rules/axe/4.10/image-alt",
		Tags:        []string{"cat.text-alternatives", "wcag2a", "wcag111"},
		NodeCount:   3,
		Sample:      &axeNode{HTML: `<img src="logo.png">`, Target: []string{"#header", "img"}},
	}}})

	if len(got) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(got))
	}
	v := got[0]
	if v.RuleID != "image-alt" || v.Impact != model.ImpactCritical || v.NodeCount != 3 {
		t.Errorf("fields not carried over: %+v", v)
	}
	if v.Principle != model.Perceivable {
		t.Errorf("principle = %s, want perceivable", v.Principle)
	}
	if v.SampleTarget != "#header img" {
		t.Errorf("sample target = %q", v.SampleTarget)
	}
}

// axe rates a few rules without an impact. Dropping them would be the same as
// declaring them harmless, so they have to land somewhere in the middle.
func TestToViolationsDefaultsMissingImpact(t *testing.T) {
	got := toViolations(axeResult{Violations: []axeViolation{{ID: "x", Impact: "", NodeCount: 1}}})
	if got[0].Impact != model.ImpactModerate {
		t.Fatalf("impact = %s, want moderate", got[0].Impact)
	}
}

// A violation always concerns at least one element; a zero count would make it
// weightless in the score.
func TestToViolationsNodeCountFloor(t *testing.T) {
	got := toViolations(axeResult{Violations: []axeViolation{{ID: "x", Impact: "minor", NodeCount: 0}}})
	if got[0].NodeCount != 1 {
		t.Fatalf("node count = %d, want 1", got[0].NodeCount)
	}
}

func TestToViolationsWithoutSample(t *testing.T) {
	got := toViolations(axeResult{Violations: []axeViolation{{ID: "x", Impact: "minor", NodeCount: 2}}})
	if got[0].SampleHTML != "" || got[0].SampleTarget != "" {
		t.Fatalf("unexpected sample: %+v", got[0])
	}
}

// The sample is stored per violation and per page; an unbounded blob of markup would
// blow up the database for no gain.
func TestToViolationsTruncatesSample(t *testing.T) {
	long := strings.Repeat("a", maxSampleHTML*2)
	got := toViolations(axeResult{Violations: []axeViolation{{
		ID: "x", Impact: "minor", NodeCount: 1, Sample: &axeNode{HTML: long},
	}}})
	if len([]rune(got[0].SampleHTML)) != maxSampleHTML+1 {
		t.Fatalf("sample not truncated: %d characters", len([]rune(got[0].SampleHTML)))
	}
}
