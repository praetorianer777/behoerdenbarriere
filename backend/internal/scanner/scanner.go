// Package scanner lädt eine Seite in einem headless Chrome und prüft sie mit axe-core.
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

// Die Seite wird gegen WCAG 2.1 A und AA geprüft — das ist der Maßstab, den
// BITV 2.0 und EN 301 549 für öffentliche Stellen setzen.
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

const pageInfoScript = `JSON.stringify({
  title: document.title || '',
  domNodes: document.getElementsByTagName('*').length,
  lang: document.documentElement.getAttribute('lang') || '',
  links: Array.from(document.querySelectorAll('a[href]')).map(a => a.href).slice(0, 2000)
})`

type Scanner struct {
	allocCtx    context.Context
	cancel      context.CancelFunc
	pageTimeout time.Duration
}

type Options struct {
	// ChromeURL zeigt auf das DevTools-HTTP-Interface eines laufenden Chrome
	// (z. B. http://chrome:9222). Leer: Chrome wird lokal gestartet.
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

// resolveWebSocketURL fragt Chrome nach seiner DevTools-Adresse. Der Umweg ist nötig,
// weil die WebSocket-URL bei jedem Start eine andere ist.
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
		return "", fmt.Errorf("chrome nicht erreichbar unter %s: %w", chromeURL, err)
	}
	defer resp.Body.Close()
	var payload struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("chrome /json/version: %w", err)
	}
	if payload.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("chrome unter %s liefert keine webSocketDebuggerUrl", chromeURL)
	}
	return payload.WebSocketDebuggerURL, nil
}

// PageScan ist das Ergebnis einer geprüften Seite samt der dort gefundenen Links,
// die der Crawler weiterverfolgen kann.
type PageScan struct {
	Result model.PageResult
	Links  []string
}

// Scan lädt eine URL, prüft sie mit axe und liest die ausgehenden Links aus.
// Ein Fehler steckt im Ergebnis (PageResult.Err), damit ein einzelner Ausfall den
// Scan einer Behörde nicht beendet.
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
		out.Result.Err = fmt.Sprintf("seiteninfo: %v", err)
		return out
	}
	out.Result.Title = info.Title
	out.Result.DOMNodes = info.DOMNodes
	out.Links = info.Links

	var axeRes axeResult
	if err := json.Unmarshal([]byte(axeJSON), &axeRes); err != nil {
		out.Result.Err = fmt.Sprintf("axe-ergebnis: %v", err)
		return out
	}
	out.Result.Violations = toViolations(axeRes)
	return out
}

// Behörden-CMS laden Navigation und Consent-Banner oft nach. Eine kurze Ruhephase
// nach dem Load liefert das DOM, das ein Besucher tatsächlich vorfindet; scheitert
// sie am Timeout, wird mit dem geprüft, was bis dahin da ist.
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

// statusRecorder merkt sich den HTTP-Status des Hauptdokuments. chromedp liefert ihn
// nicht direkt; ohne ihn ließen sich 404-Seiten nicht von echten Inhalten trennen.
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
