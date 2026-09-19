package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
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

	history, err := s.db.ScanHistory(r.Context(), agency.ID, 200)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	detail := agencyDetailDTO{
		agencyDTO: toAgencyDTO(*agency),
		Subscores: subscoresDTO{agency.Perceivable, agency.Operable, agency.Understandable, agency.Robust},
		Trend:     trend.Summarize(history, time.Now()),
		History:   history,
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

	dto := toScanDTO(*detail)
	dto.Rules = toRuleDTOs(rules)
	dto.Pages = toPageDTOs(pages)
	return &dto, nil
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

	out := statsDTO{
		Agencies: stats.Agencies, Scanned: stats.Scanned, AvgScore: stats.AvgScore,
		Grades: stats.Grades, States: states, UpdatedAt: stats.UpdatedAt,
		ByLevel: groups(stats.ByLevel), ByState: groups(stats.ByState),
		TopRules: make([]ruleCountDTO, 0, len(stats.TopRules)),
	}
	if out.Grades == nil {
		out.Grades = map[string]int{}
	}
	for _, r := range stats.TopRules {
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
	out := make([]ruleCountDTO, 0, len(stats.TopRules))
	for _, r := range stats.TopRules {
		out = append(out, ruleCountDTO{
			RuleID: r.RuleID, Impact: r.Impact, Agencies: r.Agencies, Pages: r.Pages,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// handleRescan queues an authority. It changes something and costs the authority
// traffic, so it needs the API key; queueing twice is harmless, the queue is
// idempotent.
func (s *Server) handleRescan(w http.ResponseWriter, r *http.Request) {
	if s.apiKey == "" || r.Header.Get("X-API-Key") != s.apiKey {
		writeError(w, http.StatusUnauthorized, "api key missing or wrong")
		return
	}
	agencyID, err := s.db.AgencyIDBySlug(r.Context(), chi.URLParam(r, "slug"))
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

func groups(in []store.GroupScore) []groupDTO {
	out := make([]groupDTO, 0, len(in))
	for _, g := range in {
		out = append(out, groupDTO{Name: g.Name, Agencies: g.Agencies, AvgScore: g.AvgScore})
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
