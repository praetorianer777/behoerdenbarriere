// Package seed reads the list of authorities and checks it before anything touches
// the database. A typo in a URL would otherwise only show up as a failed scan weeks
// later.
package seed

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"

	seeddata "github.com/praetorianer777/behoerdenbarriere/data"
	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

type entry struct {
	Slug     string `yaml:"slug"`
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Level    string `yaml:"level"`
	State    string `yaml:"state"`
	Category string `yaml:"category"`
}

var validLevels = map[model.Level]bool{
	model.LevelBund:    true,
	model.LevelLand:    true,
	model.LevelKreis:   true,
	model.LevelKommune: true,
}

// Load returns the embedded list.
func Load() ([]model.Agency, error) { return Parse(seeddata.SeedsYAML) }

// Parse reads a list and rejects it as a whole if a single entry is unusable —
// a half-loaded list would look like a complete one in the ranking.
func Parse(raw []byte) ([]model.Agency, error) {
	var entries []entry
	// Strict: an unexpected key is not a harmless extra, it is a truncated name. In a
	// flow mapping an unquoted comma ends the value, so
	//   {slug: bmwsb, name: Bundesministerium für Wohnen, Stadtentwicklung und Bauwesen}
	// parses as the name "Bundesministerium für Wohnen" plus a key nobody wrote, and
	// the authority went into the ranking under half its name without a word of
	// complaint.
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&entries); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("seeds: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("seeds: list is empty")
	}

	out := make([]model.Agency, 0, len(entries))
	seen := make(map[string]int, len(entries))
	for i, e := range entries {
		where := fmt.Sprintf("entry %d (%s)", i+1, e.Slug)

		if e.Slug == "" || e.Name == "" {
			return nil, fmt.Errorf("%s: slug and name are required", where)
		}
		if first, dup := seen[e.Slug]; dup {
			return nil, fmt.Errorf("%s: slug already used in entry %d", where, first+1)
		}
		seen[e.Slug] = i

		level := model.Level(e.Level)
		if !validLevels[level] {
			return nil, fmt.Errorf("%s: unknown level %q", where, e.Level)
		}
		if level != model.LevelBund && e.State == "" {
			return nil, fmt.Errorf("%s: state is required below federal level", where)
		}

		u, err := url.Parse(e.URL)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", where, err)
		}
		if u.Scheme != "https" || u.Host == "" {
			return nil, fmt.Errorf("%s: %q is not an https URL", where, e.URL)
		}
		if !strings.HasSuffix(u.Path, "/") {
			u.Path += "/"
		}

		out = append(out, model.Agency{
			Slug:     e.Slug,
			Name:     e.Name,
			URL:      u.String(),
			Level:    level,
			State:    e.State,
			Category: e.Category,
			Active:   true,
		})
	}
	return out, nil
}
