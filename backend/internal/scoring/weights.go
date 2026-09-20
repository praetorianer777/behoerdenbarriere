package scoring

import (
	"strings"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

// Weight per axe impact. The ratio 10:6:3:1 reflects that a missing form label makes a
// page unusable with a screen reader, while a missing title on an iframe only annoys.
var impactWeight = map[model.Impact]float64{
	model.ImpactCritical: 10,
	model.ImpactSerious:  6,
	model.ImpactModerate: 3,
	model.ImpactMinor:    1,
}

// Steepness of the score curve. K = 18 means a page with a violation density of 18
// weight points per 1000 DOM nodes lands at 100/e ≈ 37 points. Calibrated against
// samples: well-kept sites stay below 2, visibly neglected ones go past 20.
const decayK = 18.0

// Floor for the normalization. Without it a nearly empty page with a single violation
// would reach an absurd density and ruin the agency's average.
const minDOMNodes = 50

// Weights of the page kinds in the agency average: the entry page shapes the
// impression, and the legally relevant pages (accessibility statement, contact, forms)
// are what an audit actually looks at.
const (
	weightEntry    = 3.0
	weightPriority = 2.0
	weightOther    = 1.0
)

// Grade thresholds.
var gradeThresholds = []struct {
	min   float64
	grade string
}{
	{90, "A"},
	{80, "B"},
	{70, "C"},
	{60, "D"},
	{50, "E"},
}

// Maps axe tags onto the four WCAG principles. axe emits tags such as "wcag111"
// (success criterion 1.1.1); the first digit after "wcag" is the principle. Conformance
// tags of the same family ("wcag2aa", "wcag21aa") carry no digit sequence and must not
// be read as a criterion — "wcag2aa" would otherwise come out as principle 2.
func principleFromTags(tags []string) model.Principle {
	for _, tag := range tags {
		if !strings.HasPrefix(tag, "wcag") {
			continue
		}
		rest := strings.TrimPrefix(tag, "wcag")
		if len(rest) < 3 || strings.TrimLeft(rest, "0123456789") != "" {
			continue
		}
		switch rest[0] {
		case '1':
			return model.Perceivable
		case '2':
			return model.Operable
		case '3':
			return model.Understandable
		case '4':
			return model.Robust
		}
	}
	// Rules without a usable WCAG tag (axe's own "best-practice" rules) count as robust:
	// they almost always concern whether the markup can be interpreted at all.
	return model.Robust
}

// Principle derives the WCAG principle of a violation from its axe tags.
func Principle(tags []string) model.Principle { return principleFromTags(tags) }

// MinPages is the number of checked pages below which a result is not yet a site score.
// The score is defined as a weighted mean over entry page, priority pages and the rest;
// over one page that definition is simply not met — the number then describes that
// page, and nothing else.
//
// It happens: the federal CMS ships robots.txt with a three-minute crawl delay, which
// we honour, and the ranking ended up with authorities measured over a single page
// sitting between authorities measured over a hundred.
const MinPages = 5

// Provisional reports whether a result rests on too few pages to be read as a score for
// the whole site.
func Provisional(pages int) bool { return pages > 0 && pages < MinPages }
