// Package statement checks the accessibility statement that § 12b BGG and § 7 BITV 2.0
// oblige every public body to publish.
//
// This is the one thing we measure that is not a proxy. The score says how many
// machine-detectable barriers a site has; this says whether the authority has done
// what the law plainly requires. An authority can argue about weights. It cannot
// argue about a missing statement.
package statement

import (
	"regexp"
	"strings"
)

// Requirement is one of the contents the law prescribes.
type Requirement string

const (
	// Reachable: the statement has to be reachable from the start page — the law says
	// from every page, and a link on the start page is what can be checked without
	// crawling the whole site twice.
	Reachable Requirement = "reachable"
	// Conformance: the statement has to say whether the site is fully, partially or
	// not conformant.
	Conformance Requirement = "conformance"
	// Shortcomings: what is not accessible, and why.
	Shortcomings Requirement = "shortcomings"
	// Date: when the assessment was made.
	Date Requirement = "date"
	// Feedback: an electronic way to report a barrier.
	Feedback Requirement = "feedback"
	// Enforcement: the reference to the Schlichtungsstelle nach dem BGG.
	Enforcement Requirement = "enforcement"
)

// Requirements in the order the law lists them.
var Requirements = []Requirement{Reachable, Conformance, Shortcomings, Date, Feedback, Enforcement}

// Finding records one requirement and what was found for it. The evidence matters as
// much as the verdict: a checklist people can argue with is worth more than a badge
// they can only accept.
type Finding struct {
	Requirement Requirement `json:"requirement"`
	Met         bool        `json:"met"`
	Evidence    string      `json:"evidence,omitempty"`
}

// Result is what a statement check produced.
type Result struct {
	URL      string    `json:"url,omitempty"`
	Found    bool      `json:"found"`
	Findings []Finding `json:"findings"`
}

// Complete reports whether every prescribed element is there.
func (r Result) Complete() bool {
	if !r.Found {
		return false
	}
	for _, f := range r.Findings {
		if !f.Met {
			return false
		}
	}
	return true
}

// Met counts the requirements that are satisfied.
func (r Result) Met() int {
	met := 0
	for _, f := range r.Findings {
		if f.Met {
			met++
		}
	}
	return met
}

// statementPath recognises the page itself. The wording is not prescribed, so this is
// a list of what German authorities actually use.
var statementPath = regexp.MustCompile(`(?i)(erklaerung|erklärung)[-_]?(zur)?[-_]?barrierefreiheit|barrierefreiheitserklaerung|barrierefreiheitserklärung|accessibility[-_]?statement|barrierefreiheit`)

// IsStatementURL reports whether a URL looks like the accessibility statement.
func IsStatementURL(rawURL string) bool {
	return statementPath.MatchString(rawURL)
}

// The wordings are not prescribed, so these patterns follow what authorities write.
// Two of them were deliberately narrowed after they fired on the wrong sentence: a
// conformance status recognised from "nicht barrierefrei" matched the list of
// shortcomings, and a phone number recognised from "tel." matched "Untertitel.".
var (
	conformance  = regexp.MustCompile(`(?i)(vollständig|vollstaendig|teilweise|nicht)\s+(vereinbar|konform)|konformitätsstatus|konformitaetsstatus`)
	shortcomings = regexp.MustCompile(`(?i)nicht\s+barrierefreie?\s+inhalte|folgende\s+(bereiche|inhalte|elemente)\s+sind\s+nicht|nicht\s+vereinbare?\s+inhalte`)
	dateMention  = regexp.MustCompile(`(?i)(\d{1,2}\.\s*\d{1,2}\.\s*\d{4})|(\d{1,2}\.\s*(januar|februar|märz|maerz|april|mai|juni|juli|august|september|oktober|november|dezember)\s+\d{4})|(\d{4}-\d{2}-\d{2})`)
	assessment   = regexp.MustCompile(`(?i)(erstellt|überprüft|ueberprueft|aktualisiert|stand|bewertung|prüfung|pruefung)`)
	feedback     = regexp.MustCompile(`(?i)(barriere[n]?\s+melden|feedback|rückmeldung|rueckmeldung|kontaktieren|mitteilen|melden\s+sie)`)
	contactWay   = regexp.MustCompile(`(?i)(mailto:|[\w.\-]+@[\w.\-]+\.[a-z]{2,}|kontaktformular|\btelefon\b|\btel\.)`)
	enforcement  = regexp.MustCompile(`(?i)(schlichtungsstelle|schlichtungsverfahren|durchsetzungsverfahren|bgg\s*§?\s*16|schlichtungsstelle-bgg)`)
)

// Page is what the checker needs about a page: its address, its text, and how far it
// sits from the start page.
type Page struct {
	URL   string
	Text  string
	Depth int
}

// Check looks for the statement among the crawled pages and reads it.
//
// Nothing here decides whether a statement is *good* — only whether the prescribed
// elements are present. Whether the conformance status is honest is a question for a
// human, and pretending otherwise would be the same overreach we criticise elsewhere.
func Check(pages []Page) Result {
	result := Result{Findings: make([]Finding, 0, len(Requirements))}

	var statement *Page
	for i := range pages {
		if !IsStatementURL(pages[i].URL) {
			continue
		}
		// The most specific page wins: a site often has both a section on
		// accessibility and the statement itself.
		if statement == nil || len(pages[i].URL) > len(statement.URL) {
			statement = &pages[i]
		}
	}

	if statement == nil {
		for _, requirement := range Requirements {
			result.Findings = append(result.Findings, Finding{Requirement: requirement})
		}
		return result
	}

	result.Found = true
	result.URL = statement.URL
	text := strings.Join(strings.Fields(statement.Text), " ")

	add := func(requirement Requirement, met bool, evidence string) {
		result.Findings = append(result.Findings, Finding{
			Requirement: requirement, Met: met, Evidence: evidence,
		})
	}

	// Depth 1 means the crawler reached it from the start page, which is what the law
	// asks for in the direction we can check without crawling twice.
	add(Reachable, statement.Depth <= 1, "")
	add(Conformance, conformance.MatchString(text), excerpt(text, conformance))
	add(Shortcomings, shortcomings.MatchString(text), excerpt(text, shortcomings))
	add(Date, dateMention.MatchString(text) && assessment.MatchString(text), excerpt(text, dateMention))
	add(Feedback, feedback.MatchString(text) && contactWay.MatchString(text), excerpt(text, feedback))
	add(Enforcement, enforcement.MatchString(text), excerpt(text, enforcement))
	return result
}

const evidenceWindow = 90

// excerpt returns the sentence around the match, so a reader can judge the verdict
// instead of believing it.
func excerpt(text string, pattern *regexp.Regexp) string {
	location := pattern.FindStringIndex(text)
	if location == nil {
		return ""
	}
	start := max(location[0]-evidenceWindow/2, 0)
	end := min(location[1]+evidenceWindow, len(text))

	out := strings.TrimSpace(text[start:end])
	if start > 0 {
		out = "… " + out
	}
	if end < len(text) {
		out += " …"
	}
	return out
}
