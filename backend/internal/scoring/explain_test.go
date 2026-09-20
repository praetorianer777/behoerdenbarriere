package scoring

import (
	"math"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

func TestExplainPageAddsUpToTheScore(t *testing.T) {
	page := page("/", 800, true, false,
		v("image-alt", model.ImpactCritical, model.Perceivable, 12),
		v("color-contrast", model.ImpactSerious, model.Perceivable, 30),
		v("region", model.ImpactModerate, model.Robust, 1),
	)

	got := ExplainPage(page)
	if got.Score != PageScore(page) {
		t.Fatalf("explanation and score disagree: %v vs %v", got.Score, PageScore(page))
	}
	if len(got.Reasons) != 3 {
		t.Fatalf("%d reasons for 3 findings", len(got.Reasons))
	}

	// The shares say where the weight sits, so they have to account for all of it.
	var shares float64
	for _, reason := range got.Reasons {
		shares += reason.Share
	}
	if math.Abs(shares-1) > 0.01 {
		t.Errorf("shares add up to %v, want 1", shares)
	}
}

// The order is the order to work in: what gives back the most comes first.
func TestExplainPageRanksByWhatFixingGivesBack(t *testing.T) {
	got := ExplainPage(page("/", 800, true, false,
		v("region", model.ImpactModerate, model.Robust, 1),
		v("image-alt", model.ImpactCritical, model.Perceivable, 20),
	))

	if got.Reasons[0].RuleID != "image-alt" {
		t.Fatalf("order = %s first", got.Reasons[0].RuleID)
	}
	if got.Reasons[0].PointsIfFixed <= got.Reasons[1].PointsIfFixed {
		t.Errorf("points not ordered: %+v", got.Reasons)
	}
}

// The promised points have to be the points that actually appear: fixing the finding
// and rescoring the page must give what was promised.
func TestPointsIfFixedMatchAReScore(t *testing.T) {
	first := v("image-alt", model.ImpactCritical, model.Perceivable, 12)
	second := v("color-contrast", model.ImpactSerious, model.Perceivable, 30)

	before := ExplainPage(page("/", 800, true, false, first, second))
	after := PageScore(page("/", 800, true, false, second))

	var promised float64
	for _, reason := range before.Reasons {
		if reason.RuleID == "image-alt" {
			promised = reason.PointsIfFixed
		}
	}
	if math.Abs((before.Score+promised)-after) > 0.02 {
		t.Fatalf("promised %v points, rescoring gives %v", promised, after-before.Score)
	}
}

// Fixing two findings gains more than the two gains added up — the curve is not
// linear. Saying so is the reason share and points are two different numbers.
func TestGainsAreNotAdditive(t *testing.T) {
	first := v("image-alt", model.ImpactCritical, model.Perceivable, 12)
	second := v("color-contrast", model.ImpactSerious, model.Perceivable, 30)

	explained := ExplainPage(page("/", 800, true, false, first, second))
	var sum float64
	for _, reason := range explained.Reasons {
		sum += reason.PointsIfFixed
	}
	both := 100 - explained.Score

	if both <= sum {
		t.Fatalf("fixing both gains %v, the single gains add up to %v", both, sum)
	}
}

func TestExplainPageWithoutFindings(t *testing.T) {
	got := ExplainPage(page("/", 800, true, false))
	if got.Score != 100 || len(got.Reasons) != 0 {
		t.Fatalf("clean page: %+v", got)
	}
	if got.Density != 0 {
		t.Errorf("density = %v", got.Density)
	}
}

func TestExplainPageCarriesItsWeight(t *testing.T) {
	entry := ExplainPage(page("/", 800, true, false))
	priority := ExplainPage(page("/kontakt", 800, false, true))
	ordinary := ExplainPage(page("/presse", 800, false, false))

	if entry.Weight != 3 || priority.Weight != 2 || ordinary.Weight != 1 {
		t.Fatalf("weights = %v, %v, %v", entry.Weight, priority.Weight, ordinary.Weight)
	}
}

func TestExplainSiteRanksWhatHelpsMost(t *testing.T) {
	// The same rule on three ordinary pages against a heavier one on the entry page.
	pages := []model.PageResult{
		page("/", 800, true, false, v("image-alt", model.ImpactCritical, model.Perceivable, 3)),
		page("/a", 800, false, false, v("color-contrast", model.ImpactSerious, model.Perceivable, 25)),
		page("/b", 800, false, false, v("color-contrast", model.ImpactSerious, model.Perceivable, 25)),
		page("/c", 800, false, false, v("color-contrast", model.ImpactSerious, model.Perceivable, 25)),
	}

	got := ExplainSite(pages)
	if got.Score != SiteScore(pages).Score {
		t.Fatalf("explanation and score disagree: %v vs %v", got.Score, SiteScore(pages).Score)
	}
	if len(got.Improvements) != 2 {
		t.Fatalf("%d improvements", len(got.Improvements))
	}

	first := got.Improvements[0]
	if first.RuleID != "color-contrast" {
		t.Errorf("the rule on three pages is not first: %+v", got.Improvements)
	}
	if first.Pages != 3 || first.Nodes != 75 {
		t.Errorf("aggregation: %d pages, %d elements", first.Pages, first.Nodes)
	}
}

// A page that could not be loaded says nothing about barriers and must not turn up
// among the reasons.
func TestExplainSiteLeavesOutFailedPages(t *testing.T) {
	got := ExplainSite([]model.PageResult{
		page("/", 800, true, false, v("image-alt", model.ImpactCritical, model.Perceivable, 2)),
		{URL: "/kaputt", Err: "timeout"},
	})

	if len(got.Pages) != 1 {
		t.Fatalf("%d pages explained", len(got.Pages))
	}
}

func TestExplainSiteWithoutAnyUsablePage(t *testing.T) {
	got := ExplainSite([]model.PageResult{{URL: "/x", Err: "dns"}})
	if len(got.Pages) != 0 || len(got.Improvements) != 0 {
		t.Fatalf("explanation out of nothing: %+v", got)
	}
}

// What a fix gives back depends on where it sits: the entry page counts three times
// over in the agency's score, so the same finding there is worth more.
func TestImprovementsFollowThePageWeight(t *testing.T) {
	onEntry := ExplainSite([]model.PageResult{
		page("/", 800, true, false, v("image-alt", model.ImpactCritical, model.Perceivable, 10)),
		page("/a", 800, false, false),
		page("/b", 800, false, false),
	})
	onSubpage := ExplainSite([]model.PageResult{
		page("/", 800, true, false),
		page("/a", 800, false, false, v("image-alt", model.ImpactCritical, model.Perceivable, 10)),
		page("/b", 800, false, false),
	})

	if onEntry.Improvements[0].PointsIfFixed <= onSubpage.Improvements[0].PointsIfFixed {
		t.Fatalf("entry page not worth more: %v vs %v",
			onEntry.Improvements[0].PointsIfFixed, onSubpage.Improvements[0].PointsIfFixed)
	}
}
