// Package wikidata imports German public authorities from Wikidata.
//
// The hand-kept list covers the federal level, the states and the largest cities —
// about a hundred authorities. The level where most people actually deal with an
// authority is the district, and there are some four hundred of those. Maintaining
// that by hand would mean the list is never current; importing it means the data is
// only as good as Wikidata. So both: the import fills the breadth, the hand-kept list
// stays authoritative wherever the two meet.
package wikidata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

// Endpoint is the Wikidata Query Service.
const Endpoint = "https://query.wikidata.org/sparql"

// UserAgent identifies us to the query service, which asks for a contact address and
// turns away clients that do not give one.
const UserAgent = "behoerdenbarriere/0.1 (+https://github.com/praetorianer777/behoerdenbarriere)"

// classQuery asks which classes of district Wikidata knows.
//
// A district is typed by its state — "Landkreis in Bayern", "Kreis in Nordrhein-
// Westfalen" — and those thirteen classes are subclasses of Q106658. Asking for
// instances of Q106658 itself returns mostly the districts abolished in the reforms of
// 2007 and 2011.
const classQuery = `
SELECT ?class ?classLabel WHERE {
  ?class wdt:P279 wd:Q106658 .
  SERVICE wikibase:label { bd:serviceParam wikibase:language "de,en". }
}
`

// districtQuery asks for the districts of one class that have a website.
//
// One query per class, each a direct instance-of lookup. The obvious query — every
// district in the country, found through the subclass path and placed in its state by
// walking the administrative tree — runs past the query service's minute and returns
// nothing at all, whether asked for the whole country or for a single state.
const districtQuery = `
SELECT ?item ?itemLabel ?website WHERE {
  ?item wdt:P31 wd:%s ;
        wdt:P856 ?website .
  FILTER NOT EXISTS { ?item wdt:P576 ?dissolved }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "de,en". }
}
`

type Client struct {
	endpoint string
	http     *http.Client
}

func New(endpoint string, timeout time.Duration) *Client {
	if endpoint == "" {
		endpoint = Endpoint
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &Client{endpoint: endpoint, http: &http.Client{Timeout: timeout}}
}

type response struct {
	Results struct {
		Bindings []map[string]struct {
			Value string `json:"value"`
		} `json:"bindings"`
	} `json:"results"`
}

// FetchDistricts imports the districts of all states, one class at a time, with a
// pause between requests — the query service is a public good and we are a guest.
func (c *Client) FetchDistricts(ctx context.Context) ([]model.Agency, error) {
	classes, err := c.classes(ctx)
	if err != nil {
		return nil, err
	}

	var out []model.Agency
	var failures []string
	for i, class := range classes {
		if i > 0 {
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				return out, ctx.Err()
			}
		}

		agencies, err := c.fetch(ctx, fmt.Sprintf(districtQuery, class.ID))
		if err != nil {
			// The service drops a query that ran long under load; the same query
			// usually goes through a moment later. One retry, then it is a hole and
			// gets reported as one.
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return out, ctx.Err()
			}
			agencies, err = c.fetch(ctx, fmt.Sprintf(districtQuery, class.ID))
		}
		if err != nil {
			failures = append(failures, class.State+": "+err.Error())
			continue
		}
		for i := range agencies {
			agencies[i].State = class.State
		}
		out = append(out, agencies...)
	}

	// A class that did not answer leaves a hole, and a hole that goes unmentioned
	// looks like a state without districts.
	if len(failures) > 0 {
		return out, fmt.Errorf("%d of %d classes did not answer: %s",
			len(failures), len(classes), strings.Join(failures, "; "))
	}
	return out, nil
}

type districtClass struct {
	ID    string
	State string
}

func (c *Client) classes(ctx context.Context) ([]districtClass, error) {
	bindings, err := c.rows(ctx, classQuery)
	if err != nil {
		return nil, fmt.Errorf("district classes: %w", err)
	}

	out := make([]districtClass, 0, len(bindings))
	for _, row := range bindings {
		id := entityID(row["class"].Value)
		state := stateOfClass(row["classLabel"].Value)
		// "former district of Germany" carries no state and is exactly what we do not
		// want: authorities that no longer exist.
		if id == "" || state == "" {
			continue
		}
		out = append(out, districtClass{ID: id, State: state})
	}
	return out, nil
}

// stateOfClass reads the state out of the class name, which is where Wikidata puts it.
func stateOfClass(label string) string {
	for _, separator := range []string{" in ", " im "} {
		if _, state, found := strings.Cut(label, separator); found {
			return strings.TrimSpace(state)
		}
	}
	return ""
}

func (c *Client) fetch(ctx context.Context, query string) ([]model.Agency, error) {
	bindings, err := c.rows(ctx, query)
	if err != nil {
		return nil, err
	}
	return Agencies(bindings), nil
}

func (c *Client) rows(ctx context.Context, query string) ([]binding, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.endpoint+"?"+url.Values{"query": {query}, "format": {"json"}}.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/sparql-results+json")
	request.Header.Set("User-Agent", UserAgent)

	resp, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("wikidata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wikidata: %s", resp.Status)
	}

	var parsed response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("wikidata: %w", err)
	}
	return parsed.Results.Bindings, nil
}

type binding = map[string]struct {
	Value string `json:"value"`
}

// Agencies maps query results onto our model and drops what is unusable. An entry
// without a name or a workable address is worth nothing, and a half-imported list
// would look complete in the ranking.
func Agencies(bindings []binding) []model.Agency {
	out := make([]model.Agency, 0, len(bindings))
	seen := map[string]bool{}

	for _, row := range bindings {
		name := strings.TrimSpace(row["itemLabel"].Value)
		website := strings.TrimSpace(row["website"].Value)
		id := entityID(row["item"].Value)
		if name == "" || website == "" || id == "" {
			continue
		}
		// A label that is still the identifier means Wikidata has no German name for
		// it, and an entry called "Q123456" helps nobody.
		if strings.HasPrefix(name, "Q") && strings.Trim(name, "Q0123456789") == "" {
			continue
		}

		normalized := normalizeURL(website)
		if normalized == "" || seen[id] {
			continue
		}
		seen[id] = true

		out = append(out, model.Agency{
			Slug:       Slug(name),
			Name:       name,
			URL:        normalized,
			Level:      model.LevelKreis,
			Category:   "landkreis",
			Active:     true,
			Source:     model.SourceWikidata,
			ExternalID: id,
		})
	}
	return out
}

func entityID(uri string) string {
	if uri == "" {
		return ""
	}
	parts := strings.Split(strings.TrimSuffix(uri, "/"), "/")
	id := parts[len(parts)-1]
	if !strings.HasPrefix(id, "Q") {
		return ""
	}
	return id
}

// normalizeURL settles on https and drops what we cannot use. Wikidata holds plenty of
// http addresses that have long redirected.
func normalizeURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	u.Scheme = "https"
	u.Fragment = ""
	u.RawQuery = ""
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

var slugReplacer = strings.NewReplacer(
	"ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss",
	"Ä", "ae", "Ö", "oe", "Ü", "ue",
	" ", "-", "/", "-", ".", "", ",", "", "(", "", ")", "", "'", "",
)

// Slug builds the identifier an authority is known by in the API and the URL.
func Slug(name string) string {
	slug := slugReplacer.Replace(strings.ToLower(strings.TrimSpace(name)))
	var b strings.Builder
	lastDash := false
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' && !lastDash && b.Len() > 0:
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
