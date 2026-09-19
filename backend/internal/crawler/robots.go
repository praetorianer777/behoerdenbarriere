package crawler

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

// UserAgent names the project and carries a contact address, so an administrator who
// finds us in their logs can see who is asking and why.
const UserAgent = "behoerdenbarriere/0.1 (+https://github.com/praetorianer777/behoerdenbarriere)"

// robotsCache fetches robots.txt once per host and keeps the answer for the duration
// of a scan.
type robotsCache struct {
	client *http.Client

	mu    sync.Mutex
	hosts map[string]*robotsEntry
}

type robotsEntry struct {
	group *robotstxt.Group
	once  sync.Once
}

func newRobotsCache(client *http.Client) *robotsCache {
	return &robotsCache{client: client, hosts: map[string]*robotsEntry{}}
}

// Allowed reports whether robots.txt permits this URL. A robots.txt that cannot be
// fetched is treated as permission: a server that is briefly unavailable should not
// silently drop an authority out of the ranking. A robots.txt that forbids us is
// obeyed, even though the law obliges the authority to be accessible.
func (r *robotsCache) Allowed(ctx context.Context, rawURL string) bool {
	group := r.group(ctx, rawURL)
	if group == nil {
		return true
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	return group.Test(path)
}

// CrawlDelay returns the pause robots.txt asks for, or zero.
func (r *robotsCache) CrawlDelay(ctx context.Context, rawURL string) time.Duration {
	group := r.group(ctx, rawURL)
	if group == nil {
		return 0
	}
	return group.CrawlDelay
}

func (r *robotsCache) group(ctx context.Context, rawURL string) *robotstxt.Group {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return nil
	}
	key := u.Scheme + "://" + u.Host

	r.mu.Lock()
	entry, ok := r.hosts[key]
	if !ok {
		entry = &robotsEntry{}
		r.hosts[key] = entry
	}
	r.mu.Unlock()

	entry.once.Do(func() { entry.group = r.fetch(ctx, key) })
	return entry.group
}

func (r *robotsCache) fetch(ctx context.Context, base string) *robotstxt.Group {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/robots.txt", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil
	}
	data, err := robotstxt.FromStatusAndBytes(resp.StatusCode, body)
	if err != nil {
		return nil
	}
	return data.FindGroup(UserAgent)
}
