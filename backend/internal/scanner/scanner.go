// Package scanner loads a page in a headless Chrome and checks it with axe-core.
package scanner

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
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
  return JSON.stringify({
    title: document.title || '',
    domNodes: nodes,
    lang: document.documentElement.getAttribute('lang') || '',
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

// resolveWebSocketURL asks Chrome for its DevTools address. The detour is needed
// because the WebSocket URL differs on every start.
func resolveWebSocketURL(ctx context.Context, chromeURL string) (string, error) {
	base := strings.TrimSuffix(chromeURL, "/")
	if strings.HasPrefix(base, "ws://") || strings.HasPrefix(base, "wss://") {
		return base, nil
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
	return payload.WebSocketDebuggerURL, nil
}

// PageScan is the outcome of a checked page together with the links found on it,
// which the crawler can follow.
type PageScan struct {
	Result model.PageResult
	Links  []string
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
	chromedp.ListenTarget(runCtx, status.handle)

	var infoJSON, axeJSON string
	started := time.Now()
	err := chromedp.Run(runCtx,
		network.Enable(),
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		waitForQuiet(),
		evalJSON(pageInfoScript, &infoJSON),
		chromedp.Evaluate(axeJS, nil),
		evalPromise(axeRunScript, &axeJSON),
	)
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
		Links    []string `json:"links"`
	}
	if err := json.Unmarshal([]byte(infoJSON), &info); err != nil {
		out.Result.Err = fmt.Sprintf("page info: %v", err)
		return out
	}
	out.Result.Title = info.Title
	out.Result.DOMNodes = info.DOMNodes
	out.Links = info.Links

	var axeRes axeResult
	if err := json.Unmarshal([]byte(axeJSON), &axeRes); err != nil {
		out.Result.Err = fmt.Sprintf("axe result: %v", err)
		return out
	}
	out.Result.Violations = toViolations(axeRes)
	return out
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
