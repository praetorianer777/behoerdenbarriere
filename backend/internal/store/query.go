package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/statement"
)

// AgencyListing is an authority as the ranking shows it: master data plus its latest
// score and the one before, so a change can be shown without a second query.
type AgencyListing struct {
	model.Agency
	Score                                         *float64
	Grade                                         string
	ScannedAt                                     *time.Time
	Pages                                         int
	PrevScore                                     *float64
	Perceivable, Operable, Understandable, Robust *float64
	LighthouseScore                               *float64
}

// AgencyFilter narrows the ranking.
type AgencyFilter struct {
	Query   string
	Level   string
	State   string
	Grade   string
	Sort    string
	Page    int
	PerPage int
}

const maxPerPage = 200

func (f AgencyFilter) normalize() AgencyFilter {
	if f.Page < 1 {
		f.Page = 1
	}
	switch {
	case f.PerPage <= 0:
		f.PerPage = 50
	case f.PerPage > maxPerPage:
		f.PerPage = maxPerPage
	}
	return f
}

// orderBy translates the sort parameter. The column names are chosen here and never
// taken from the request, so no input reaches the SQL.
//
// Authorities without a scan sort last in either direction: "not yet checked" is not
// the same as "checked and bad", and it should not head the table either way.
// orderBy maps the requested sort onto SQL. Everything that can be missing is sorted
// with NULLS LAST in both directions: an authority that was never checked has no date
// and no change, and putting it first because "nothing" sorts below every number would
// answer a question nobody asked. Never checked is not "checked long ago", and no
// change is not a change of zero.
//
// The level is not sorted alphabetically but from the federal government downwards,
// which is the order the levels are actually thought of in.
func orderBy(sort string) string {
	switch sort {
	case "score_asc":
		return "s.score ASC NULLS LAST, a.name ASC"
	case "name":
		return "a.name ASC"
	case "name_desc":
		return "a.name DESC"
	case "level":
		return levelOrder + " ASC, a.name ASC"
	case "level_desc":
		return levelOrder + " DESC, a.name ASC"
	case "scanned":
		return "s.finished_at DESC NULLS LAST, a.name ASC"
	case "scanned_asc":
		return "s.finished_at ASC NULLS LAST, a.name ASC"
	case "delta":
		return deltaOrder + " DESC NULLS LAST, a.name ASC"
	case "delta_asc":
		return deltaOrder + " ASC NULLS LAST, a.name ASC"
	default:
		return "s.score DESC NULLS LAST, a.name ASC"
	}
}

// levelOrder sorts by rank, not by name: bund, land, kreis, kommune.
const levelOrder = `CASE a.level
	WHEN 'bund' THEN 1 WHEN 'land' THEN 2 WHEN 'kreis' THEN 3 ELSE 4 END`

// deltaOrder is the change against the previous scan. It exists only where both scans
// do; where one is missing the row has no change, not a change of zero.
const deltaOrder = `(s.score - p.score)`

const agencyColumns = `
	a.id, a.slug, a.name, a.url, a.level, coalesce(a.state, ''), coalesce(a.category, ''),
	a.active, a.created_at,
	s.score, coalesce(s.grade, ''), s.finished_at, coalesce(s.pages_scanned, 0),
	p.score,
	s.score_perceivable, s.score_operable, s.score_understandable, s.score_robust,
	s.lighthouse_score`

// latestScans attaches the newest finished scan and the one before it.
const latestScans = `
	LEFT JOIN LATERAL (
		SELECT score, grade, finished_at, pages_scanned,
		       score_perceivable, score_operable, score_understandable, score_robust,
		       lighthouse_score
		FROM scans WHERE agency_id = a.id AND status = 'done'
		ORDER BY finished_at DESC LIMIT 1
	) s ON true
	LEFT JOIN LATERAL (
		SELECT score FROM scans WHERE agency_id = a.id AND status = 'done'
		ORDER BY finished_at DESC OFFSET 1 LIMIT 1
	) p ON true`

// ListAgencies returns one page of the ranking and the total number of matches.
func (s *Store) ListAgencies(ctx context.Context, f AgencyFilter) ([]AgencyListing, int, error) {
	f = f.normalize()

	where := `WHERE a.active
		AND ($1 = '' OR a.name ILIKE '%' || $1 || '%' OR a.slug ILIKE '%' || $1 || '%')
		AND ($2 = '' OR a.level::text = $2)
		AND ($3 = '' OR a.state = $3)
		AND ($4 = '' OR s.grade = $4)`

	var total int
	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM agencies a `+latestScans+` `+where,
		f.Query, f.Level, f.State, f.Grade).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count authorities: %w", err)
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT `+agencyColumns+` FROM agencies a `+latestScans+` `+where+
			` ORDER BY `+orderBy(f.Sort)+` LIMIT $5 OFFSET $6`,
		f.Query, f.Level, f.State, f.Grade, f.PerPage, (f.Page-1)*f.PerPage)
	if err != nil {
		return nil, 0, fmt.Errorf("list authorities: %w", err)
	}
	defer rows.Close()

	listings, err := scanListings(rows)
	return listings, total, err
}

// AgencyBySlug returns one authority with its latest score.
func (s *Store) AgencyBySlug(ctx context.Context, slug string) (*AgencyListing, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+agencyColumns+` FROM agencies a `+latestScans+` WHERE a.slug = $1`, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	listings, err := scanListings(rows)
	if err != nil {
		return nil, err
	}
	if len(listings) == 0 {
		return nil, ErrNotFound
	}
	return &listings[0], nil
}

// ErrNotFound means the authority or scan does not exist.
var ErrNotFound = errors.New("not found")

func scanListings(rows pgx.Rows) ([]AgencyListing, error) {
	var out []AgencyListing
	for rows.Next() {
		var a AgencyListing
		if err := rows.Scan(
			&a.ID, &a.Slug, &a.Name, &a.URL, &a.Level, &a.State, &a.Category,
			&a.Active, &a.CreatedAt,
			&a.Score, &a.Grade, &a.ScannedAt, &a.Pages, &a.PrevScore,
			&a.Perceivable, &a.Operable, &a.Understandable, &a.Robust,
			&a.LighthouseScore,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ScanDetail is a finished scan with its numbers.
type ScanDetail struct {
	ID                                            int64
	AgencyID                                      int64
	AgencySlug                                    string
	AgencyName                                    string
	AgencyURL                                     string
	Status                                        string
	StartedAt                                     time.Time
	FinishedAt                                    *time.Time
	Error                                         string
	Score                                         *float64
	Grade                                         string
	Perceivable, Operable, Understandable, Robust *float64
	PagesScanned                                  int
	PagesFailed                                   int
	PagesBlocked                                  int
	LighthouseScore                               *float64
	LighthouseFailed                              []string
	Statement                                     *statement.Result
}

func (s *Store) ScanByID(ctx context.Context, id int64) (*ScanDetail, error) {
	var d ScanDetail
	err := s.Pool.QueryRow(ctx, `
		SELECT sc.id, sc.agency_id, a.slug, a.name, a.url, sc.status, sc.started_at, sc.finished_at,
		       coalesce(sc.error, ''), sc.score, coalesce(sc.grade, ''),
		       sc.score_perceivable, sc.score_operable, sc.score_understandable, sc.score_robust,
		       sc.pages_scanned, sc.pages_failed, sc.pages_blocked,
		       sc.lighthouse_score, coalesce(sc.lighthouse_failed, '{}'), sc.statement
		FROM scans sc JOIN agencies a ON a.id = sc.agency_id
		WHERE sc.id = $1`, id,
	).Scan(&d.ID, &d.AgencyID, &d.AgencySlug, &d.AgencyName, &d.AgencyURL, &d.Status, &d.StartedAt, &d.FinishedAt,
		&d.Error, &d.Score, &d.Grade, &d.Perceivable, &d.Operable, &d.Understandable, &d.Robust,
		&d.PagesScanned, &d.PagesFailed, &d.PagesBlocked,
		&d.LighthouseScore, &d.LighthouseFailed, &d.Statement)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// LatestScanID returns the newest finished scan of an authority.
func (s *Store) LatestScanID(ctx context.Context, agencyID int64) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx, `
		SELECT id FROM scans WHERE agency_id = $1 AND status = 'done'
		ORDER BY finished_at DESC LIMIT 1`, agencyID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

// PageDetail is a checked page of a scan.
type PageDetail struct {
	URL        string
	Title      string
	Depth      int
	IsEntry    bool
	Priority   bool
	HTTPStatus int
	DOMNodes   int
	Score      *float64
	LoadMS     int
	Error      string
	Consent    string
	Violations int
}

func (s *Store) PagesForScan(ctx context.Context, scanID int64) ([]PageDetail, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT p.url, coalesce(p.title, ''), p.depth, p.is_entry, p.priority,
		       coalesce(p.http_status, 0), p.dom_nodes, p.page_score, coalesce(p.load_ms, 0),
		       coalesce(p.error, ''), p.consent::text, count(v.id)
		FROM pages p
		LEFT JOIN violations v ON v.page_id = p.id
		WHERE p.scan_id = $1
		GROUP BY p.id
		ORDER BY p.is_entry DESC, p.priority DESC, p.page_score ASC NULLS LAST`, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PageDetail
	for rows.Next() {
		var p PageDetail
		if err := rows.Scan(&p.URL, &p.Title, &p.Depth, &p.IsEntry, &p.Priority,
			&p.HTTPStatus, &p.DOMNodes, &p.Score, &p.LoadMS, &p.Error, &p.Consent,
			&p.Violations); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Stats is the nationwide picture.
type Stats struct {
	Agencies  int
	Scanned   int
	AvgScore  *float64
	Grades    map[string]int
	ByLevel   []GroupScore
	ByState   []GroupScore
	TopRules  []RuleCount
	UpdatedAt *time.Time
}

type GroupScore struct {
	Name     string
	Agencies int
	// Scanned is how many of Agencies the average rests on. A column that pairs a
	// count of all authorities with an average over the checked ones reads as one
	// statement, and that statement is false: Saarland was shown as eight authorities
	// averaging zero when one had been checked, and its only page was a block page.
	Scanned  int
	AvgScore *float64
}

type RuleCount struct {
	RuleID   string
	Impact   string
	Agencies int
	Pages    int
}

func (s *Store) Stats(ctx context.Context) (*Stats, error) {
	stats := &Stats{Grades: map[string]int{}}

	if err := s.Pool.QueryRow(ctx, `
		SELECT count(*),
		       count(s.score),
		       avg(s.score),
		       max(s.finished_at)
		FROM agencies a `+latestScans+`
		WHERE a.active`,
	).Scan(&stats.Agencies, &stats.Scanned, &stats.AvgScore, &stats.UpdatedAt); err != nil {
		return nil, fmt.Errorf("overview: %w", err)
	}

	if err := s.eachRow(ctx, `
		SELECT coalesce(s.grade, ''), count(*)
		FROM agencies a `+latestScans+`
		WHERE a.active AND s.grade IS NOT NULL
		GROUP BY s.grade`, func(rows pgx.Rows) error {
		var grade string
		var count int
		if err := rows.Scan(&grade, &count); err != nil {
			return err
		}
		stats.Grades[grade] = count
		return nil
	}); err != nil {
		return nil, fmt.Errorf("grades: %w", err)
	}

	groups := func(column string, into *[]GroupScore) error {
		return s.eachRow(ctx, `
			SELECT `+column+`, count(*), count(s.score), avg(s.score)
			FROM agencies a `+latestScans+`
			WHERE a.active AND `+column+` IS NOT NULL
			GROUP BY 1 ORDER BY 1`, func(rows pgx.Rows) error {
			var g GroupScore
			if err := rows.Scan(&g.Name, &g.Agencies, &g.Scanned, &g.AvgScore); err != nil {
				return err
			}
			*into = append(*into, g)
			return nil
		})
	}
	if err := groups("a.level::text", &stats.ByLevel); err != nil {
		return nil, fmt.Errorf("by level: %w", err)
	}
	if err := groups("a.state", &stats.ByState); err != nil {
		return nil, fmt.Errorf("by state: %w", err)
	}

	// Which barriers are the most widespread: counted per authority, not per element,
	// so that one page with a thousand images does not decide the list.
	if err := s.eachRow(ctx, `
		SELECT v.rule_id, v.impact::text, count(DISTINCT a.id), count(DISTINCT pg.id)
		FROM agencies a `+latestScans+`
		JOIN scans sc ON sc.agency_id = a.id AND sc.finished_at = s.finished_at
		JOIN pages pg ON pg.scan_id = sc.id
		JOIN violations v ON v.page_id = pg.id
		WHERE a.active
		GROUP BY v.rule_id, v.impact
		ORDER BY count(DISTINCT a.id) DESC, count(DISTINCT pg.id) DESC
		LIMIT 20`, func(rows pgx.Rows) error {
		var r RuleCount
		if err := rows.Scan(&r.RuleID, &r.Impact, &r.Agencies, &r.Pages); err != nil {
			return err
		}
		stats.TopRules = append(stats.TopRules, r)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("rules: %w", err)
	}
	return stats, nil
}

// States returns the states that occur, for the filter in the ranking.
func (s *Store) States(ctx context.Context) ([]string, error) {
	var out []string
	err := s.eachRow(ctx,
		`SELECT DISTINCT state FROM agencies WHERE active AND state IS NOT NULL ORDER BY state`,
		func(rows pgx.Rows) error {
			var state string
			if err := rows.Scan(&state); err != nil {
				return err
			}
			out = append(out, state)
			return nil
		})
	return out, err
}

func (s *Store) eachRow(ctx context.Context, sql string, fn func(pgx.Rows) error, args ...any) error {
	rows, err := s.Pool.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// AgencyIDBySlug is what the rescan endpoint needs.
func (s *Store) AgencyIDBySlug(ctx context.Context, slug string) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx, `SELECT id FROM agencies WHERE slug = $1 AND active`, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

// PageResultsForScan rebuilds the checked pages with their findings, so the scoring
// package can take the score apart again exactly as it put it together.
func (s *Store) PageResultsForScan(ctx context.Context, scanID int64) ([]model.PageResult, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT p.id, p.url, coalesce(p.title, ''), p.depth, p.is_entry, p.priority,
		       coalesce(p.http_status, 0), p.dom_nodes, coalesce(p.error, '')
		FROM pages p WHERE p.scan_id = $1
		ORDER BY p.is_entry DESC, p.priority DESC, p.page_score ASC NULLS LAST`, scanID)
	if err != nil {
		return nil, fmt.Errorf("pages of the scan: %w", err)
	}
	defer rows.Close()

	pages := make([]model.PageResult, 0)
	byID := map[int64]int{}
	for rows.Next() {
		var id int64
		var page model.PageResult
		if err := rows.Scan(&id, &page.URL, &page.Title, &page.Depth, &page.IsEntry,
			&page.Priority, &page.HTTPStatus, &page.DOMNodes, &page.Err); err != nil {
			return nil, err
		}
		byID[id] = len(pages)
		pages = append(pages, page)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return pages, nil
	}

	violations, err := s.Pool.Query(ctx, `
		SELECT v.page_id, v.rule_id, v.impact, v.principle, coalesce(v.help, ''),
		       coalesce(v.help_url, ''), v.node_count, coalesce(v.sample_html, ''),
		       coalesce(v.sample_target, '')
		FROM violations v
		JOIN pages p ON p.id = v.page_id
		WHERE p.scan_id = $1`, scanID)
	if err != nil {
		return nil, fmt.Errorf("findings of the scan: %w", err)
	}
	defer violations.Close()

	for violations.Next() {
		var pageID int64
		var v model.Violation
		if err := violations.Scan(&pageID, &v.RuleID, &v.Impact, &v.Principle, &v.Help,
			&v.HelpURL, &v.NodeCount, &v.SampleHTML, &v.SampleTarget); err != nil {
			return nil, err
		}
		if index, ok := byID[pageID]; ok {
			pages[index].Violations = append(pages[index].Violations, v)
		}
	}
	return pages, violations.Err()
}

// Failure is why the last attempt at an authority came to nothing.
type Failure struct {
	Reason string
	At     time.Time
}

// LastFailure returns the newest failed scan of an authority, if the attempt after it
// did not succeed. An authority that failed last month and has a score since does not
// need to be told about it.
//
// It exists because a refusal must not look like an absence: bot protection takes about
// twenty authorities out of the ranking, and "not checked" without a reason tells
// nobody anything.
func (s *Store) LastFailure(ctx context.Context, agencyID int64) (*Failure, error) {
	var failure Failure
	err := s.Pool.QueryRow(ctx, `
		SELECT coalesce(error, ''), finished_at
		FROM scans
		WHERE agency_id = $1 AND finished_at IS NOT NULL
		ORDER BY finished_at DESC
		LIMIT 1`, agencyID).Scan(&failure.Reason, &failure.At)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("last failure: %w", err)
	}
	// Der letzte Lauf war erfolgreich: Dann gibt es nichts zu erklären.
	if failure.Reason == "" {
		return nil, ErrNotFound
	}
	return &failure, nil
}
