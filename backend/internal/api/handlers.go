package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/praetorianer777/behoerdenbarriere/internal/maildns"
	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/thirdparty"
	"github.com/praetorianer777/behoerdenbarriere/internal/trend"
)

// Queries is what the API needs from the database. An interface, so the handlers can
// be tested without one.
type Queries interface {
	Ping(ctx context.Context) error
	ListAgencies(ctx context.Context, f store.AgencyFilter) ([]store.AgencyListing, int, error)
	AgencyBySlug(ctx context.Context, slug string) (*store.AgencyListing, error)
	AgencyIDBySlug(ctx context.Context, slug string) (int64, error)
	ScanHistory(ctx context.Context, agencyID int64, limit int) ([]trend.Point, error)
	LastScanIDs(ctx context.Context, agencyID int64, limit int) ([]int64, error)
	LatestScanID(ctx context.Context, agencyID int64) (int64, error)
	ScanByID(ctx context.Context, id int64) (*store.ScanDetail, error)
	RulesForScan(ctx context.Context, scanID int64) ([]scoring.RuleSummary, error)
	PagesForScan(ctx context.Context, scanID int64) ([]store.PageDetail, error)
	PageResultsForScan(ctx context.Context, scanID int64) ([]model.PageResult, error)
	ContactsForScan(ctx context.Context, scanID int64) ([]thirdparty.Seen, error)
	ContactReach(ctx context.Context, limit int) ([]store.ContactReach, error)
	LastFailure(ctx context.Context, agencyID int64) (*store.Failure, error)
	MailForAgency(ctx context.Context, agencyID int64) (*maildns.Record, error)
	MailOverview(ctx context.Context) (*store.MailSummary, error)
	Stats(ctx context.Context) (*store.Stats, error)
	States(ctx context.Context) ([]string, error)
	EnqueueScan(ctx context.Context, agencyID int64) error
	Usage(ctx context.Context, days int) (*store.UsageSummary, error)
}

func (s *Server) handleAgencies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := store.AgencyFilter{
		Query:   q.Get("q"),
		Level:   q.Get("level"),
		State:   q.Get("state"),
		Grade:   q.Get("grade"),
		Sort:    q.Get("sort"),
		Page:    atoi(q.Get("page"), 1),
		PerPage: atoi(q.Get("per_page"), 50),
	}

	items, total, err := s.db.ListAgencies(r.Context(), filter)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	out := listDTO{Items: make([]agencyDTO, 0, len(items)), Total: total,
		Page: max(filter.Page, 1), PerPage: filter.PerPage}
	for _, a := range items {
		out.Items = append(out.Items, toAgencyDTO(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAgency(w http.ResponseWriter, r *http.Request) {
	agency, err := s.db.AgencyBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		s.fail(w, r, err)
		return
	}

	history, err := s.db.ScanHistory(r.Context(), agency.ID, s.limits.HistoryPoints)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	history = clip(history, s.limits.HistoryPoints)
	detail := agencyDetailDTO{
		agencyDTO: toAgencyDTO(*agency),
		Subscores: subscoresDTO{agency.Perceivable, agency.Operable, agency.Understandable, agency.Robust},
		Trend:     trend.Summarize(history, time.Now()),
		History:   history,
	}
	// Warum die letzte Prüfung nichts ergeben hat. Ohne diese Angabe sähe eine
	// abgewiesene Behörde aus wie eine, die noch niemand angefasst hat.
	if failure, err := s.db.LastFailure(r.Context(), agency.ID); err == nil {
		detail.Failure = &failureDTO{Reason: failure.Reason, At: failure.At}
	} else if !errors.Is(err, store.ErrNotFound) {
		s.fail(w, r, err)
		return
	}

	// What the domain publishes about its email. Not every authority has been looked
	// up, and a missing record is not a statement about the authority.
	if mail, err := s.db.MailForAgency(r.Context(), agency.ID); err == nil {
		detail.Mail = toMailDTO(mail)
	} else if !errors.Is(err, store.ErrNotFound) {
		s.fail(w, r, err)
		return
	}
	if id, err := s.db.LatestScanID(r.Context(), agency.ID); err == nil {
		detail.LatestID = id
	} else if !errors.Is(err, store.ErrNotFound) {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// handleLatestScan answers with the newest finished scan of an authority, including
// what changed against the one before it.
func (s *Server) handleLatestScan(w http.ResponseWriter, r *http.Request) {
	agencyID, err := s.db.AgencyIDBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	ids, err := s.db.LastScanIDs(r.Context(), agencyID, 2)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if len(ids) == 0 {
		s.fail(w, r, store.ErrNotFound)
		return
	}

	dto, err := s.scanDetail(r.Context(), ids[0])
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if len(ids) > 1 {
		previous, err := s.db.RulesForScan(r.Context(), ids[1])
		if err != nil {
			s.fail(w, r, err)
			return
		}
		current, err := s.db.RulesForScan(r.Context(), ids[0])
		if err != nil {
			s.fail(w, r, err)
			return
		}
		change := trend.CompareRules(previous, current)
		dto.Changes = &change
	}
	writeJSON(w, http.StatusOK, dto)
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "scan id is not a number")
		return
	}
	dto, err := s.scanDetail(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (s *Server) scanDetail(ctx context.Context, id int64) (*scanDTO, error) {
	detail, err := s.db.ScanByID(ctx, id)
	if err != nil {
		return nil, err
	}
	rules, err := s.db.RulesForScan(ctx, id)
	if err != nil {
		return nil, err
	}
	pages, err := s.db.PagesForScan(ctx, id)
	if err != nil {
		return nil, err
	}

	// The explanation is rebuilt from the stored findings rather than stored itself:
	// the arithmetic belongs in one place, and a score taken apart by an older formula
	// than the one that produced it would be a second truth.
	results, err := s.db.PageResultsForScan(ctx, id)
	if err != nil {
		return nil, err
	}

	dto := toScanDTO(*detail)
	dto.Rules = toRuleDTOs(clip(rules, s.limits.ListItems))
	dto.Pages = toPageDTOs(clip(pages, s.limits.ListItems))

	explanation := scoring.ExplainSite(clip(results, s.limits.ListItems))
	explanation.Improvements = clip(explanation.Improvements, s.limits.ListItems)
	dto.Explanation = &explanation

	// A scan without recorded contacts is not a scan without third parties: it may
	// predate the recording. The list is therefore left out entirely rather than shown
	// as an empty one, which would read as "contacts none".
	seen, err := s.db.ContactsForScan(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(seen) > 0 {
		dto.ThirdParties = clip(thirdparty.Describe(seen, hostOf(detail.AgencyURL)), s.limits.ListItems)
	}
	return &dto, nil
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// handleThirdParties answers with the nationwide picture: which outside host was seen
// on how many authorities, kept apart by the phase of the visit it appeared in.
func (s *Server) handleThirdParties(w http.ResponseWriter, r *http.Request) {
	reach, err := s.db.ContactReach(r.Context(), s.limits.ListItems)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	stats, err := s.db.Stats(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}

	out := thirdPartyListDTO{Items: make([]thirdPartyDTO, 0, len(reach)), Scanned: stats.Scanned}
	for _, c := range reach {
		out.Items = append(out.Items, thirdPartyDTO{
			Host:       c.Host,
			Domain:     thirdparty.Registrable(c.Host),
			Group:      string(thirdparty.Classify(c.Host)),
			PublicBody: thirdparty.PublicBody(c.Host),
			Phase:      string(c.Phase),
			Agencies:   c.Agencies,
			Pages:      c.Pages,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// clip keeps a response from growing with the database. A scan of a large authority
// can carry thousands of pages, and nobody reads them in one answer.
//
// An empty list comes back as an empty list, never as nil: a nil slice marshals to
// null, and a reader that expects an array gets an exception instead of a page. That
// happened — an authority without scans blanked the whole site.
func clip[T any](items []T, limit int) []T {
	if items == nil {
		return []T{}
	}
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	return items
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.Stats(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	states, err := s.db.States(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}

	topRules := clip(stats.TopRules, s.limits.ListItems)
	out := statsDTO{
		Agencies: stats.Agencies, Scanned: stats.Scanned, AvgScore: stats.AvgScore,
		Grades: stats.Grades, States: clip(states, s.limits.ListItems), UpdatedAt: stats.UpdatedAt,
		ByLevel:  groups(clip(stats.ByLevel, s.limits.ListItems)),
		ByState:  groups(clip(stats.ByState, s.limits.ListItems)),
		TopRules: make([]ruleCountDTO, 0, len(topRules)),
	}
	if out.Grades == nil {
		out.Grades = map[string]int{}
	}
	for _, r := range topRules {
		out.TopRules = append(out.TopRules, ruleCountDTO{
			RuleID: r.RuleID, Impact: r.Impact, Agencies: r.Agencies, Pages: r.Pages,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.Stats(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	rules := clip(stats.TopRules, s.limits.ListItems)
	out := make([]ruleCountDTO, 0, len(rules))
	for _, r := range rules {
		out = append(out, ruleCountDTO{
			RuleID: r.RuleID, Impact: r.Impact, Agencies: r.Agencies, Pages: r.Pages,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// handleRescan queues an authority. It changes something and costs the authority
// traffic, so it needs the API key; queueing twice is harmless, the queue is
// idempotent.
//
// The per-authority limit is counted for the authority, not for the caller: the
// traffic lands on that authority's servers however many keys ask for it.
func (s *Server) handleRescan(w http.ResponseWriter, r *http.Request) {
	if !s.hasKey(r.Header.Get("X-API-Key")) {
		writeError(w, http.StatusUnauthorized, "api key missing or wrong")
		return
	}
	slug := chi.URLParam(r, "slug")
	if s.limits.RescanPerAgency > 0 {
		d := s.rescanLimiter.allow("rescan|"+slug, rate{
			perSecond: 1 / s.limits.RescanPerAgency.Seconds(), burst: 1,
		})
		writeLimitHeaders(w, d)
		if !d.ok {
			w.Header().Set("Retry-After", strconv.Itoa(seconds(d.retryAfter)))
			writeError(w, http.StatusTooManyRequests, "this authority was queued recently")
			return
		}
	}
	agencyID, err := s.db.AgencyIDBySlug(r.Context(), slug)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.db.EnqueueScan(r.Context(), agencyID); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

// handleMail answers with the nationwide picture of where authority mail is received.
// Counts and raw material side by side: the classification is ours, the records are
// the authorities' own.
func (s *Server) handleMail(w http.ResponseWriter, r *http.Request) {
	summary, err := s.db.MailOverview(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}

	out := mailSummaryDTO{
		Total:      summary.Total,
		CheckedAt:  summary.CheckedAt,
		ByProvider: mailCounts(clip(summary.ByProvider, s.limits.ListItems)),
		ByState:    mailCounts(clip(summary.ByState, s.limits.ListItems)),
		ByLevel:    mailCounts(clip(summary.ByLevel, s.limits.ListItems)),
	}
	writeJSON(w, http.StatusOK, out)
}

func mailCounts(in []store.MailCount) []mailCountDTO {
	out := make([]mailCountDTO, 0, len(in))
	for _, count := range in {
		out = append(out, mailCountDTO{
			Name: count.Name, Provider: count.Provider, Agencies: count.Agencies,
			US:     maildns.USBased[maildns.Provider(count.Provider)],
			Filter: maildns.Filters[maildns.Provider(count.Provider)],
		})
	}
	return out
}

func groups(in []store.GroupScore) []groupDTO {
	out := make([]groupDTO, 0, len(in))
	for _, g := range in {
		out = append(out, groupDTO{Name: g.Name, Agencies: g.Agencies, Scanned: g.Scanned, AvgScore: g.AvgScore})
	}
	return out
}

func atoi(raw string, def int) int {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}
