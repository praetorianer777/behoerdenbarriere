// Package lighthouse asks the Lighthouse service for Google's accessibility score.
//
// It is the only established, openly documented accessibility score, and it is built
// the other way round from ours: every audit is strictly pass or fail, weighted by
// axe impact, so one missing alt text among a hundred images fails the whole audit.
// We weight by severity and damp the count. Carrying both numbers makes ours
// comparable and gives its weights an outside check.
package lighthouse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	url  string
	http *http.Client
}

// New returns a client, or nil when no service is configured. A nil client answers
// "no result" rather than an error: the Lighthouse number is an addition, and its
// absence must never cost us a scan.
func New(serviceURL string, timeout time.Duration) *Client {
	if serviceURL == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &Client{url: serviceURL, http: &http.Client{Timeout: timeout}}
}

// Result is what one audit produced.
type Result struct {
	Score        float64  `json:"score"`
	FailedAudits []string `json:"failed_audits"`
	Version      string   `json:"lighthouse_version"`
	FetchedURL   string   `json:"fetched_url"`
}

// Audit measures one page. The second return value says whether there is a result at
// all; a service that is down or slow is not a failed scan.
func (c *Client) Audit(ctx context.Context, pageURL string) (*Result, error) {
	if c == nil {
		return nil, nil
	}

	body, err := json.Marshal(map[string]string{"url": pageURL})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/audit", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lighthouse: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&failure)
		if failure.Error == "" {
			failure.Error = resp.Status
		}
		return nil, fmt.Errorf("lighthouse: %s", failure.Error)
	}

	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("lighthouse: %w", err)
	}
	// A score outside 0..100 means the service answered something we do not
	// understand, and a wrong number is worse than none.
	if result.Score < 0 || result.Score > 100 {
		return nil, fmt.Errorf("lighthouse: score %v outside 0..100", result.Score)
	}
	return &result, nil
}
