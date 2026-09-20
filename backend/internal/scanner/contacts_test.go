package scanner

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chromedp/cdproto/network"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

func request(rawURL string) *network.EventRequestWillBeSent {
	return &network.EventRequestWillBeSent{Request: &network.Request{URL: rawURL}}
}

func TestContactRecorderKeepsPhasesApart(t *testing.T) {
	rec := newContactRecorder("https://www.stadt-musterau.de/")

	rec.handle(request("https://fonts.gstatic.com/s/roboto.woff2"))
	rec.handle(request("https://fonts.gstatic.com/s/roboto-bold.woff2"))
	rec.handle(request("https://www.stadt-musterau.de/style.css"))
	rec.handle(request("data:image/png;base64,AAAA"))

	rec.afterConsent(model.ConsentDeclined)
	rec.handle(request("https://fonts.gstatic.com/s/roboto.woff2"))
	rec.handle(request("https://www.youtube.com/embed/1"))

	got := map[contactKey]int{}
	for _, c := range rec.contacts() {
		got[contactKey{host: c.Host, phase: c.Phase}] = c.Requests
	}

	want := map[contactKey]int{
		{host: "fonts.gstatic.com", phase: model.PhaseBeforeConsent}: 2,
		{host: "fonts.gstatic.com", phase: model.PhaseAfterDeclined}: 1,
		{host: "www.youtube.com", phase: model.PhaseAfterDeclined}:   1,
	}
	if len(got) != len(want) {
		t.Fatalf("contacts = %v, want %v", got, want)
	}
	for key, count := range want {
		if got[key] != count {
			t.Errorf("%s in %s = %d, want %d", key.host, key.phase, got[key], count)
		}
	}
}

// A page that never showed a layer, or one whose layer stayed, has nothing that came
// "after" a decision: nobody decided anything.
func TestContactRecorderStaysBeforeConsentWithoutADecision(t *testing.T) {
	for _, state := range []model.Consent{model.ConsentNone, model.ConsentBlocked} {
		rec := newContactRecorder("https://www.stadt-musterau.de/")
		rec.afterConsent(state)
		rec.handle(request("https://www.google-analytics.com/collect"))

		contacts := rec.contacts()
		if len(contacts) != 1 || contacts[0].Phase != model.PhaseBeforeConsent {
			t.Errorf("consent %q: contacts = %+v", state, contacts)
		}
	}
}

func TestContactRecorderStopsAtTheCap(t *testing.T) {
	rec := newContactRecorder("https://www.stadt-musterau.de/")
	for i := range maxContactHosts + 50 {
		rec.handle(request(fmt.Sprintf("https://host%d.example.com/", i)))
	}
	// A host already known still counts up — only new ones are turned away.
	rec.handle(request("https://host0.example.com/again"))

	contacts := rec.contacts()
	if len(contacts) != maxContactHosts {
		t.Fatalf("hosts = %d, want %d", len(contacts), maxContactHosts)
	}
	for _, c := range contacts {
		if c.Host == "host0.example.com" && c.Requests != 2 {
			t.Errorf("requests of a known host = %d, want 2", c.Requests)
		}
	}
}

// serveOn puts a server on a given loopback address, so a test page can contact a host
// that is genuinely not its own.
func serveOn(t *testing.T, addr string, handler http.Handler) string {
	t.Helper()
	listener, err := net.Listen("tcp", addr+":0")
	if err != nil {
		t.Skipf("cannot listen on %s: %v", addr, err)
	}
	srv := httptest.Server{Listener: listener, Config: &http.Server{Handler: handler}}
	srv.Start()
	t.Cleanup(srv.Close)
	return srv.URL
}

func pixel() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/gif")
		_, _ = w.Write([]byte("GIF89a"))
	})
}

func TestScanRecordsAThirdPartyBeforeConsent(t *testing.T) {
	s := newTestScanner(t)
	tracker := serveOn(t, "127.0.0.2", pixel())

	page := fmt.Sprintf(`<!doctype html><html lang="de"><head><title>Amt</title></head>
<body><h1>Amt</h1><img src="%s/zaehler.gif" alt="">
</body></html>`, tracker)

	got := s.Scan(context.Background(), servePage(t, page))
	if got.Result.Err != "" {
		t.Fatalf("scan failed: %s", got.Result.Err)
	}

	var found bool
	for _, c := range got.Result.Contacts {
		if c.Host == "127.0.0.2" {
			found = true
			if c.Phase != model.PhaseBeforeConsent {
				t.Errorf("phase = %q, want before_consent", c.Phase)
			}
		}
		// The page's own host is not a third party.
		if c.Host == "127.0.0.1" {
			t.Errorf("the site's own host was recorded: %+v", c)
		}
	}
	if !found {
		t.Errorf("the third party was not recorded, got %+v", got.Result.Contacts)
	}
}

// What a banner releases only after it has been answered belongs in the later phase —
// otherwise a site that asks first would look like one that does not.
func TestScanSeparatesWhatLoadsAfterDeclining(t *testing.T) {
	s := newTestScanner(t)
	tracker := serveOn(t, "127.0.0.3", pixel())

	page := fmt.Sprintf(`<!doctype html><html lang="de"><head><title>Amt</title></head>
<body>
  <h1>Amt</h1>
  <div id="banner" style="position:fixed;inset:0;background:#fff">
    <p>Wir verwenden Cookies.</p>
    <button id="ok">Alle akzeptieren</button>
    <button id="no">Nur notwendige</button>
  </div>
  <script>
    document.getElementById('no').addEventListener('click', () => {
      document.getElementById('banner').remove();
      const img = new Image();
      img.src = '%s/spaet.gif';
      document.body.appendChild(img);
    });
  </script>
</body></html>`, tracker)

	got := s.Scan(context.Background(), servePage(t, page))
	if got.Result.Err != "" {
		t.Fatalf("scan failed: %s", got.Result.Err)
	}
	if got.Result.Consent != model.ConsentDeclined {
		t.Fatalf("consent = %q, want declined", got.Result.Consent)
	}

	var found bool
	for _, c := range got.Result.Contacts {
		if c.Host == "127.0.0.3" {
			found = true
			if c.Phase != model.PhaseAfterDeclined {
				t.Errorf("phase = %q, want after_declined", c.Phase)
			}
		}
	}
	if !found {
		t.Errorf("the late request was not recorded, got %+v", got.Result.Contacts)
	}
}

// A banner fires its requests inside the click handler, before anyone can know which
// button was pressed. Counting those as "before consent" would put the worst label on
// the one moment the visitor was actually asked.
func TestContactRecorderFilesTheClickUnderTheDecision(t *testing.T) {
	cases := map[model.Consent]model.ContactPhase{
		model.ConsentDeclined: model.PhaseAfterDeclined,
		model.ConsentAccepted: model.PhaseAfterAccepted,
		model.ConsentBlocked:  model.PhaseBeforeConsent,
		model.ConsentNone:     model.PhaseBeforeConsent,
	}
	for state, want := range cases {
		rec := newContactRecorder("https://www.stadt-musterau.de/")
		rec.beginDecision()
		rec.handle(request("https://www.googletagmanager.com/gtm.js"))
		rec.afterConsent(state)

		contacts := rec.contacts()
		if len(contacts) != 1 || contacts[0].Phase != want {
			t.Errorf("consent %q: contacts = %+v, want phase %q", state, contacts, want)
		}
	}
}

// What was already on its way before the banner was touched stays where it was, even
// when the click lands in the same phase.
func TestContactRecorderDoesNotMoveEarlierRequests(t *testing.T) {
	rec := newContactRecorder("https://www.stadt-musterau.de/")
	rec.handle(request("https://fonts.gstatic.com/s/roboto.woff2"))
	rec.beginDecision()
	rec.handle(request("https://fonts.gstatic.com/s/roboto.woff2"))
	rec.afterConsent(model.ConsentDeclined)

	got := map[model.ContactPhase]int{}
	for _, c := range rec.contacts() {
		got[c.Phase] = c.Requests
	}
	if got[model.PhaseBeforeConsent] != 1 || got[model.PhaseAfterDeclined] != 1 {
		t.Errorf("phases = %v, want one request in each", got)
	}
}
