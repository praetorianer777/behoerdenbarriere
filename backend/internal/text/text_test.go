package text_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/praetorianer777/behoerdenbarriere/internal/text"
)

// Genau die Art Text, in der es passiert ist: deutsche Behördensprache mit Umlauten,
// Gedankenstrichen und typografischen Anführungszeichen.
const deutsch = `Diese Website ist mit der Barrierefreie-Informationstechnik-Verordnung ` +
	`„teilweise vereinbar“ — nicht barrierefrei sind Inhalte in Gebärdensprache und ` +
	`ältere PDF-Dokumente, größtenteils aus den Jahren vor 2020.`

func TestTruncateNeverBreaksACharacter(t *testing.T) {
	// An jeder Stelle abschneiden, nicht an einer glücklich gewählten: Der Fehler saß
	// genau an den Grenzen.
	for limit := range utf8.RuneCountInString(deutsch) + 5 {
		got := text.Truncate(deutsch, limit)
		if !utf8.ValidString(got) {
			t.Fatalf("limit %d produced broken text: %q", limit, got)
		}
		if strings.ContainsRune(got, utf8.RuneError) {
			t.Fatalf("limit %d produced a replacement character: %q", limit, got)
		}
	}
}

func TestTruncateMarksTheCutAndKeepsShortTextWhole(t *testing.T) {
	if got := text.Truncate("kurz", 10); got != "kurz" {
		t.Errorf("short text = %q, want it untouched", got)
	}
	got := text.Truncate("abcdefghij", 4)
	if got != "abcd…" {
		t.Errorf("truncated = %q, want abcd…", got)
	}
	// Die Grenze zählt Zeichen, nicht Bytes: Sonst wäre ein Satz mit Umlauten
	// willkürlich kürzer als derselbe Satz ohne.
	if got := text.Truncate("ääää", 3); got != "äää…" {
		t.Errorf("umlauts = %q, want äää…", got)
	}
	if text.Truncate(deutsch, 0) != "" {
		t.Error("a limit of zero has nothing to show")
	}
}

func TestSnapMovesOntoCharacterBoundaries(t *testing.T) {
	for start := range len(deutsch) + 1 {
		for _, width := range []int{0, 1, 5, 40} {
			from, to := text.Snap(deutsch, start, start+width)
			cut := deutsch[from:to]
			if !utf8.ValidString(cut) {
				t.Fatalf("[%d:%d] is broken: %q", start, start+width, cut)
			}
			if strings.ContainsRune(cut, utf8.RuneError) {
				t.Fatalf("[%d:%d] has a replacement character: %q", start, start+width, cut)
			}
		}
	}
}

// Nach außen gerundet: Ein Ausschnitt darf ein Zeichen länger sein als gewünscht, aber
// er darf nicht das Zeichen verlieren, um das es ging.
func TestSnapGrowsRatherThanShrinks(t *testing.T) {
	// "ä" beginnt bei Byte 0 und ist zwei Bytes lang.
	start, end := text.Snap("ätzend", 1, 2)
	if start != 0 {
		t.Errorf("start = %d, want 0", start)
	}
	if end != 2 {
		t.Errorf("end = %d, want 2", end)
	}
	if got := "ätzend"[start:end]; got != "ä" {
		t.Errorf("cut = %q, want ä", got)
	}
}

func TestSnapHandlesImpossibleRanges(t *testing.T) {
	if start, end := text.Snap("abc", -5, 99); start != 0 || end != 3 {
		t.Errorf("clamped range = %d, %d", start, end)
	}
	if start, end := text.Snap("abc", 3, 1); start > end {
		t.Errorf("start %d is behind end %d", start, end)
	}
}
