package scoring

import (
	"strings"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

// Gewichte je axe-Impact. Das Verhältnis 10:6:3:1 bildet ab, dass ein fehlendes
// Formularlabel eine Seite für Screenreader-Nutzer unbedienbar macht, ein fehlendes
// title-Attribut an einem Iframe dagegen nur stört.
var impactWeight = map[model.Impact]float64{
	model.ImpactCritical: 10,
	model.ImpactSerious:  6,
	model.ImpactModerate: 3,
	model.ImpactMinor:    1,
}

// Steilheit der Score-Kurve. K = 18 bedeutet: eine Seite mit einer Verstoßdichte von
// 18 Gewichtspunkten je 1000 DOM-Knoten landet bei 100/e ≈ 37 Punkten. Kalibriert an
// Stichproben: gepflegte Auftritte liegen unter 2, erkennbar vernachlässigte über 20.
const decayK = 18.0

// Mindestgröße für die Normierung. Ohne sie bekäme eine fast leere Seite mit einem
// einzigen Verstoß eine absurd hohe Dichte und würde das Mittel der Behörde ruinieren.
const minDOMNodes = 50

// Gewichte der Seitenarten im Behörden-Mittelwert: die Einstiegsseite prägt den
// Eindruck, die rechtlich relevanten Seiten (Erklärung zur Barrierefreiheit, Kontakt,
// Formulare) sind der eigentliche Prüfgegenstand.
const (
	weightEntry    = 3.0
	weightPriority = 2.0
	weightOther    = 1.0
)

// Notengrenzen.
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

// Zuordnung der axe-Tags zu den vier WCAG-Prinzipien. axe liefert Tags wie
// "wcag111" (Erfolgskriterium 1.1.1); die erste Ziffer nach "wcag" ist das Prinzip.
// Konformitätstags derselben Familie ("wcag2aa", "wcag21aa") tragen keine Ziffernfolge
// und dürfen nicht als Kriterium gelesen werden — "wcag2aa" wäre sonst Prinzip 2.
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
	// Regeln ohne verwertbaren WCAG-Tag (axe-eigene "best-practice"-Regeln) zählen als
	// robust: sie betreffen fast immer die technische Auswertbarkeit des Markups.
	return model.Robust
}

// Principle bestimmt das WCAG-Prinzip eines Verstoßes aus seinen axe-Tags.
func Principle(tags []string) model.Principle { return principleFromTags(tags) }
