// Package scanner loads a page in a headless Chrome and checks it with axe-core.
package scanner

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/thirdparty"
)

//go:embed axe/axe.min.js
var axeJS string

// Pages are checked against WCAG 2.1 A and AA — the bar that BITV 2.0 and EN 301 549
// set for public bodies.
const axeRunScript = `(async () => {
  const result = await axe.run(document, {
    runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'] },
    resultTypes: ['violations'],
    elementRef: false
  });
  return JSON.stringify({
    violations: result.violations.map(v => ({
      id: v.id, impact: v.impact, description: v.description,
      help: v.help, helpUrl: v.helpUrl, tags: v.tags,
      nodeCount: v.nodes.length,
      sample: v.nodes.length ? { html: v.nodes[0].html, target: v.nodes[0].target.map(String) } : null
    }))
  });
})()`

// Counting and link collection descend into shadow roots. Modern government portals
// are built from web components; counted from the light DOM alone, bund.de has 28
// nodes instead of several thousand, and the violation density — and with it the
// score — would be off by two orders of magnitude.
const pageInfoScript = `(() => {
  let nodes = 0;
  const links = [];
  const walk = root => {
    for (const el of root.querySelectorAll('*')) {
      nodes++;
      if (el.tagName === 'A' && el.href) links.push(el.href);
      if (el.shadowRoot) walk(el.shadowRoot);
    }
  };
  walk(document);
  // Der sichtbare Text, gekappt: Er wird nur gelesen, um die Erklärung zur
  // Barrierefreiheit zu prüfen, und nicht gespeichert.
  const text = (document.body && document.body.innerText || '').slice(0, 40000);
  return JSON.stringify({
    title: document.title || '',
    domNodes: nodes,
    lang: document.documentElement.getAttribute('lang') || '',
    text: text,
    links: links.slice(0, 2000)
  });
})()`

type Scanner struct {
	allocCtx    context.Context
	cancel      context.CancelFunc
	pageTimeout time.Duration
}

type Options struct {
	// ChromeURL points at the DevTools HTTP interface of a running Chrome
	// (e.g. http://chrome:9222). Empty: Chrome is started locally.
	ChromeURL   string
	PageTimeout time.Duration
}

func New(ctx context.Context, opts Options) (*Scanner, error) {
	if opts.PageTimeout <= 0 {
		opts.PageTimeout = 30 * time.Second
	}

	var allocCtx context.Context
	var cancel context.CancelFunc
	if opts.ChromeURL == "" {
		allocCtx, cancel = chromedp.NewExecAllocator(ctx, append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.DisableGPU,
			chromedp.NoSandbox,
			chromedp.WindowSize(1280, 1024),
		)...)
	} else {
		wsURL, err := resolveWebSocketURL(ctx, opts.ChromeURL)
		if err != nil {
			return nil, err
		}
		allocCtx, cancel = chromedp.NewRemoteAllocator(ctx, wsURL)
	}
	return &Scanner{allocCtx: allocCtx, cancel: cancel, pageTimeout: opts.PageTimeout}, nil
}

func (s *Scanner) Close() { s.cancel() }

// DevToolsWebSocketURL asks a running Chrome for the address to drive it through.
// Exported so that other tools — the reflow check — reach the same Chrome the same
// way, including the host-name handling that a container setup needs.
func DevToolsWebSocketURL(ctx context.Context, chromeURL string) (string, error) {
	return resolveWebSocketURL(ctx, chromeURL)
}

// resolveWebSocketURL asks Chrome for its DevTools address. The detour is needed
// because the WebSocket URL differs on every start.
func resolveWebSocketURL(ctx context.Context, chromeURL string) (string, error) {
	base := strings.TrimSuffix(chromeURL, "/")
	if strings.HasPrefix(base, "ws://") || strings.HasPrefix(base, "wss://") {
		return base, nil
	}

	// Chrome's DevTools endpoint refuses any request whose Host header is a name
	// rather than an IP or "localhost" — in a compose setup, where the service is
	// reached as http://chrome:9222, it answers "Host header is specified and is not
	// an IP address or localhost" and the worker never starts. So the name is resolved
	// here, once, and everything after this talks to the address.
	base, err := withResolvedHost(ctx, base)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/json/version", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("chrome unreachable at %s: %w", chromeURL, err)
	}
	defer resp.Body.Close()
	var payload struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("chrome /json/version: %w", err)
	}
	if payload.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("chrome at %s returns no webSocketDebuggerUrl", chromeURL)
	}

	// Chrome answers with the host it knows itself by — inside a container that is
	// "localhost", which from another container points at that container itself. The
	// path carries the session id and has to be kept; the host is the one we asked,
	// already resolved to an address above.
	ws, err := url.Parse(payload.WebSocketDebuggerURL)
	if err != nil {
		return "", fmt.Errorf("chrome at %s returns an unusable webSocketDebuggerUrl: %w", chromeURL, err)
	}
	asked, err := url.Parse(base)
	if err == nil && asked.Host != "" {
		ws.Host = asked.Host
	}
	return ws.String(), nil
}

// withResolvedHost replaces a host name in the URL with one of its addresses.
func withResolvedHost(ctx context.Context, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	host := u.Hostname()
	if host == "" || host == "localhost" || net.ParseIP(host) != nil {
		return rawURL, nil
	}

	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil || len(addrs) == 0 {
		return "", fmt.Errorf("chrome host %q cannot be resolved: %w", host, err)
	}
	if port := u.Port(); port != "" {
		u.Host = net.JoinHostPort(addrs[0], port)
	} else {
		u.Host = addrs[0]
	}
	return u.String(), nil
}

// PageScan is the outcome of a checked page together with the links found on it,
// which the crawler can follow.
type PageScan struct {
	Result model.PageResult
	Links  []string

	// ConsentLabel is the wording on the button that was clicked. It is not stored;
	// it exists so that a surprising result can be traced back to what was pressed.
	ConsentLabel string
}

// Scan loads a URL, checks it with axe and reads the outgoing links. A failure is
// carried in the result (PageResult.Err) so that one bad page does not end an agency's
// scan.
func (s *Scanner) Scan(ctx context.Context, url string) PageScan {
	out := PageScan{Result: model.PageResult{URL: url}}

	tabCtx, cancelTab := chromedp.NewContext(s.allocCtx)
	defer cancelTab()
	runCtx, cancelRun := context.WithTimeout(tabCtx, s.pageTimeout)
	defer cancelRun()

	status := newStatusRecorder(url)
	contacts := newContactRecorder(url)
	chromedp.ListenTarget(runCtx, status.handle)
	chromedp.ListenTarget(runCtx, contacts.handle)

	var infoJSON, axeJSON string
	consent := model.ConsentNone
	started := time.Now()
	err := chromedp.Run(runCtx,
		network.Enable(),
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		waitForQuiet(),
		beginDecision(contacts),
		dismissConsent(&consent, &out.ConsentLabel),
		afterConsent(contacts, &consent),
		evalJSON(pageInfoScript, &infoJSON),
		chromedp.Evaluate(axeJS, nil),
		evalPromise(axeRunScript, &axeJSON),
	)
	out.Result.Consent = consent
	out.Result.Contacts = contacts.contacts()
	out.Result.LoadMS = int(time.Since(started).Milliseconds())
	out.Result.HTTPStatus = status.code()

	if err != nil {
		out.Result.Err = err.Error()
		return out
	}

	var info struct {
		Title    string   `json:"title"`
		DOMNodes int      `json:"domNodes"`
		Lang     string   `json:"lang"`
		Text     string   `json:"text"`
		Links    []string `json:"links"`
	}
	if err := json.Unmarshal([]byte(infoJSON), &info); err != nil {
		out.Result.Err = fmt.Sprintf("page info: %v", err)
		return out
	}
	out.Result.Title = info.Title
	out.Result.DOMNodes = info.DOMNodes
	out.Result.Text = info.Text
	out.Links = info.Links

	var axeRes axeResult
	if err := json.Unmarshal([]byte(axeJSON), &axeRes); err != nil {
		out.Result.Err = fmt.Sprintf("axe result: %v", err)
		return out
	}
	out.Result.Violations = toViolations(axeRes)
	return out
}

// dismissConsent clears the consent layer out of the way before axe runs, and records
// what it took. A failure here is not a failure of the scan: the page is then checked
// as it stands, marked as blocked, and the result says so.
func dismissConsent(out *model.Consent, label *string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		step, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()

		var raw string
		if err := chromedp.Run(step, chromedp.Evaluate(consentScript, &raw)); err != nil {
			return nil
		}
		var attempt struct {
			State string `json:"state"`
			Label string `json:"label"`
		}
		if err := json.Unmarshal([]byte(raw), &attempt); err != nil {
			return nil
		}
		if attempt.State == "none" {
			*out = model.ConsentNone
			return nil
		}
		if attempt.State == "blocked" {
			*out = model.ConsentBlocked
			return nil
		}
		*label = attempt.Label

		// Banners take their time going away — bmi.bund.de needs close to three
		// seconds, and a single short look afterwards declared it blocked although the
		// click had worked. So the page is asked repeatedly until the layer is gone.
		if blocked := stillBlocked(step); blocked {
			*out = model.ConsentBlocked
			return nil
		}

		if attempt.State == "declined" {
			*out = model.ConsentDeclined
		} else {
			*out = model.ConsentAccepted
		}
		return nil
	})
}

func beginDecision(contacts *contactRecorder) chromedp.Action {
	return chromedp.ActionFunc(func(context.Context) error {
		contacts.beginDecision()
		return nil
	})
}

// afterConsent switches the recording over and gives the page a moment to act on the
// decision. Without the pause the requests a banner releases on "accept" would land in
// the wrong phase or be missed entirely, and that distinction is the whole point.
func afterConsent(contacts *contactRecorder, state *model.Consent) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		contacts.afterConsent(*state)
		if *state == model.ConsentDeclined || *state == model.ConsentAccepted {
			settle, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			_ = chromedp.Run(settle, chromedp.Sleep(1500*time.Millisecond))
		}
		return nil
	})
}

// stillBlocked waits for the layer to disappear and reports whether it is still there.
func stillBlocked(ctx context.Context) bool {
	deadline := time.Now().Add(6 * time.Second)
	for {
		var raw string
		if err := chromedp.Run(ctx,
			chromedp.Sleep(400*time.Millisecond),
			chromedp.Evaluate(consentCheckScript, &raw),
		); err != nil {
			return false
		}
		var check struct {
			Blocked bool `json:"blocked"`
		}
		if err := json.Unmarshal([]byte(raw), &check); err != nil {
			return false
		}
		if !check.Blocked {
			return false
		}
		if time.Now().After(deadline) {
			return true
		}
	}
}

// Government CMSes often load navigation and consent banners late. A short settling
// period after load yields the DOM a visitor actually meets; if it runs into the
// timeout, whatever is there by then gets checked.
func waitForQuiet() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		quiet, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = chromedp.Run(quiet, chromedp.Sleep(1500*time.Millisecond))
		return nil
	})
}

func evalJSON(script string, out *string) chromedp.Action {
	return chromedp.Evaluate(script, out)
}

func evalPromise(script string, out *string) chromedp.Action {
	return chromedp.Evaluate(script, out, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
		return p.WithAwaitPromise(true)
	})
}

// statusRecorder keeps the HTTP status of the main document. chromedp does not expose
// it directly, and without it a 404 page could not be told from real content.
type statusRecorder struct {
	url    string
	status int
}

func newStatusRecorder(url string) *statusRecorder { return &statusRecorder{url: url} }

func (s *statusRecorder) handle(ev interface{}) {
	e, ok := ev.(*network.EventResponseReceived)
	if !ok || e.Type != network.ResourceTypeDocument {
		return
	}
	if s.status == 0 {
		s.status = int(e.Response.Status)
	}
}

func (s *statusRecorder) code() int { return s.status }

// maxContactHosts caps what one page can produce. An ad-heavy page can talk to
// hundreds of hosts; beyond this many the picture does not get clearer, and the
// database should not grow without a bound.
const maxContactHosts = 200

// contactRecorder notes which hosts a page reaches out to, and in which phase of the
// visit. The phase is what makes the record worth anything: the same request means
// something different before and after a consent decision, and afterwards nobody can
// tell the two apart from the host name alone.
//
// Requests within the site's own registrable domain are not recorded — they say nothing
// about data leaving.
type contactRecorder struct {
	mu     sync.Mutex
	site   string
	phase  model.ContactPhase
	counts map[contactKey]int
}

type contactKey struct {
	host  string
	phase model.ContactPhase
}

// phasePending holds the requests that go out while the consent layer is being
// answered. A banner usually fires its trackers in the click handler, so those requests
// are already on their way before we know which button was pressed — counting them as
// "before consent" would blame a site for exactly the thing it did right.
const phasePending model.ContactPhase = "pending"

func newContactRecorder(pageURL string) *contactRecorder {
	host := ""
	if u, err := url.Parse(pageURL); err == nil {
		host = u.Hostname()
	}
	return &contactRecorder{
		site:   host,
		phase:  model.PhaseBeforeConsent,
		counts: map[contactKey]int{},
	}
}

func (c *contactRecorder) handle(ev interface{}) {
	e, ok := ev.(*network.EventRequestWillBeSent)
	if !ok {
		return
	}
	u, err := url.Parse(e.Request.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || thirdparty.SameSite(host, c.site) {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	key := contactKey{host: host, phase: c.phase}
	if _, known := c.counts[key]; !known && len(c.counts) >= maxContactHosts {
		return
	}
	c.counts[key]++
}

// beginDecision is called before the consent layer is answered.
func (c *contactRecorder) beginDecision() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.phase = phasePending
}

// afterConsent moves the recording into the phase that follows the consent decision and
// files what happened during the click under it. A layer that was never there, or that
// stayed, has nothing that came "after" a decision: nobody decided anything, so those
// requests belong where the rest do.
func (c *contactRecorder) afterConsent(state model.Consent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch state {
	case model.ConsentDeclined:
		c.phase = model.PhaseAfterDeclined
	case model.ConsentAccepted:
		c.phase = model.PhaseAfterAccepted
	default:
		c.phase = model.PhaseBeforeConsent
	}

	for key, count := range c.counts {
		if key.phase != phasePending {
			continue
		}
		delete(c.counts, key)
		c.counts[contactKey{host: key.host, phase: c.phase}] += count
	}
}

func (c *contactRecorder) contacts() []model.Contact {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]model.Contact, 0, len(c.counts))
	for key, count := range c.counts {
		out = append(out, model.Contact{Host: key.host, Phase: key.phase, Requests: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Host != out[j].Host {
			return out[i].Host < out[j].Host
		}
		return out[i].Phase < out[j].Phase
	})
	return out
}
