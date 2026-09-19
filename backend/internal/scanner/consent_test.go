package scanner

import (
	"context"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

// The page behind every banner carries the same violation — a missing alt text. Whether
// the scan finds it is the whole question: a banner that is not cleared away means the
// score describes the banner.
const behindBanner = `<main><h1>Amt</h1><img src="wappen.png"></main>`

func bannerPage(banner string) string {
	return `<!doctype html>
<html lang="de">
<head><title>Amt mit Banner</title><style>
  #layer { position: fixed; inset: 0; background: #fff; z-index: 9999; padding: 2rem; }
  .hidden { display: none; }
</style></head>
<body>
  ` + behindBanner + `
  ` + banner + `
  <script>
    function dismiss() { document.getElementById('layer').classList.add('hidden') }
  </script>
</body>
</html>`
}

// A banner that offers a choice: a careful visitor declines, and so do we.
const bannerWithDecline = `<div id="layer" role="dialog" aria-label="Cookie-Einstellungen">
  <h2>Cookies und Einwilligung</h2>
  <p>Wir verwenden Cookies.</p>
  <button type="button" onclick="dismiss()">Alle akzeptieren</button>
  <button type="button" onclick="dismiss()">Nur notwendige</button>
</div>`

// A banner with no way out but consent — then consent it is, because a visitor has to
// click something to see the page.
const bannerAcceptOnly = `<div id="layer" role="dialog" aria-label="Cookie-Hinweis">
  <h2>Cookie-Hinweis</h2>
  <button type="button" onclick="dismiss()">Einverstanden</button>
</div>`

// A banner with nothing to click. It stays, and the result has to say so.
const bannerWithoutButton = `<div id="layer" role="dialog" aria-label="Cookie-Hinweis">
  <h2>Cookies</h2>
  <p>Diese Seite verwendet Cookies. Durch die weitere Nutzung stimmen Sie zu.</p>
</div>`

func scanBanner(t *testing.T, banner string) (model.PageResult, map[string]bool) {
	t.Helper()
	s := newTestScanner(t)
	got := s.Scan(context.Background(), servePage(t, bannerPage(banner)))
	if got.Result.Err != "" {
		t.Fatalf("scan failed: %s", got.Result.Err)
	}
	rules := map[string]bool{}
	for _, v := range got.Result.Violations {
		rules[v.RuleID] = true
	}
	return got.Result, rules
}

func TestConsentDeclinedRevealsThePage(t *testing.T) {
	result, rules := scanBanner(t, bannerWithDecline)

	if result.Consent != model.ConsentDeclined {
		t.Fatalf("consent = %s, want declined", result.Consent)
	}
	if !rules["image-alt"] {
		t.Fatalf("the violation behind the banner was not found: %v", sortedKeys(rules))
	}
}

func TestConsentAcceptedWhenNothingCanBeDeclined(t *testing.T) {
	result, rules := scanBanner(t, bannerAcceptOnly)

	if result.Consent != model.ConsentAccepted {
		t.Fatalf("consent = %s, want accepted", result.Consent)
	}
	if !rules["image-alt"] {
		t.Fatalf("the violation behind the banner was not found: %v", sortedKeys(rules))
	}
}

// The honest case: the banner stays, and the result says so instead of passing the
// banner off as the site.
func TestConsentBlockedIsReported(t *testing.T) {
	result, _ := scanBanner(t, bannerWithoutButton)

	if result.Consent != model.ConsentBlocked {
		t.Fatalf("consent = %s, want blocked", result.Consent)
	}
}

func TestPageWithoutBannerIsUntouched(t *testing.T) {
	s := newTestScanner(t)
	got := s.Scan(context.Background(), servePage(t, cleanPage))

	if got.Result.Err != "" {
		t.Fatalf("scan failed: %s", got.Result.Err)
	}
	if got.Result.Consent != model.ConsentNone {
		t.Fatalf("consent = %s, want none", got.Result.Consent)
	}
}

// The common consent tools render in a shadow root, where a plain querySelector never
// looks.
func TestConsentInsideAShadowRoot(t *testing.T) {
	page := `<!doctype html>
<html lang="de">
<head><title>Amt mit Shadow-Banner</title></head>
<body>
  ` + behindBanner + `
  <div id="host"></div>
  <script>
    const root = document.getElementById('host').attachShadow({ mode: 'open' });
    root.innerHTML = ` + "`" + `
      <style>#layer { position: fixed; inset: 0; background: #fff; z-index: 9999; }</style>
      <div id="layer" role="dialog" aria-label="Cookie-Einstellungen">
        <h2>Cookies</h2>
        <button type="button" id="ok">Alle akzeptieren</button>
        <button type="button" id="no">Ablehnen</button>
      </div>` + "`" + `;
    for (const button of root.querySelectorAll('button')) {
      button.addEventListener('click', () => root.getElementById('layer').remove());
    }
  </script>
</body>
</html>`

	s := newTestScanner(t)
	got := s.Scan(context.Background(), servePage(t, page))
	if got.Result.Err != "" {
		t.Fatalf("scan failed: %s", got.Result.Err)
	}
	if got.Result.Consent != model.ConsentDeclined {
		t.Fatalf("consent = %s, want declined", got.Result.Consent)
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
