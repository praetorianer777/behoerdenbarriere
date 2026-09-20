package api

import (
	"context"
	"errors"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/thirdparty"
	"github.com/praetorianer777/behoerdenbarriere/internal/trend"
)

// fakeDB answers from fields, so the handlers can be exercised without a database.
type fakeDB struct {
	pingErr error

	agencies []store.AgencyListing
	total    int
	filter   store.AgencyFilter

	// calls counts every query, so a test can show that a rejected request never
	// reached the database.
	calls       int
	history     []trend.Point
	scanIDs     []int64
	scan        *store.ScanDetail
	rules       map[int64][]scoring.RuleSummary
	pages       []store.PageDetail
	pageResults []model.PageResult
	stats       *store.Stats
	states      []string
	queued      []int64
	usage       *store.UsageSummary
	usageDays   int
	seen        []thirdparty.Seen
	reach       []store.ContactReach
	failWith    error
}

func (f *fakeDB) Ping(context.Context) error { f.calls++; return f.pingErr }

func (f *fakeDB) ContactsForScan(context.Context, int64) ([]thirdparty.Seen, error) {
	f.calls++
	if f.failWith != nil {
		return nil, f.failWith
	}
	return f.seen, nil
}

func (f *fakeDB) ContactReach(context.Context, int) ([]store.ContactReach, error) {
	f.calls++
	if f.failWith != nil {
		return nil, f.failWith
	}
	return f.reach, nil
}

func (f *fakeDB) ListAgencies(_ context.Context, filter store.AgencyFilter) ([]store.AgencyListing, int, error) {
	f.calls++
	f.filter = filter
	if f.failWith != nil {
		return nil, 0, f.failWith
	}
	return f.agencies, f.total, nil
}

func (f *fakeDB) AgencyBySlug(_ context.Context, slug string) (*store.AgencyListing, error) {
	f.calls++
	if f.failWith != nil {
		return nil, f.failWith
	}
	for i := range f.agencies {
		if f.agencies[i].Slug == slug {
			return &f.agencies[i], nil
		}
	}
	return nil, store.ErrNotFound
}

func (f *fakeDB) AgencyIDBySlug(_ context.Context, slug string) (int64, error) {
	f.calls++
	if f.failWith != nil {
		return 0, f.failWith
	}
	for i := range f.agencies {
		if f.agencies[i].Slug == slug {
			return f.agencies[i].ID, nil
		}
	}
	return 0, store.ErrNotFound
}

func (f *fakeDB) ScanHistory(context.Context, int64, int) ([]trend.Point, error) {
	f.calls++
	return f.history, nil
}

func (f *fakeDB) LastScanIDs(context.Context, int64, int) ([]int64, error) {
	f.calls++
	return f.scanIDs, nil
}

func (f *fakeDB) LatestScanID(context.Context, int64) (int64, error) {
	f.calls++
	if len(f.scanIDs) == 0 {
		return 0, store.ErrNotFound
	}
	return f.scanIDs[0], nil
}

func (f *fakeDB) ScanByID(_ context.Context, id int64) (*store.ScanDetail, error) {
	f.calls++
	if f.scan == nil || f.scan.ID != id {
		return nil, store.ErrNotFound
	}
	return f.scan, nil
}

func (f *fakeDB) RulesForScan(_ context.Context, scanID int64) ([]scoring.RuleSummary, error) {
	f.calls++
	return f.rules[scanID], nil
}

func (f *fakeDB) PageResultsForScan(context.Context, int64) ([]model.PageResult, error) {
	f.calls++
	return f.pageResults, nil
}

func (f *fakeDB) PagesForScan(context.Context, int64) ([]store.PageDetail, error) {
	f.calls++
	return f.pages, nil
}

func (f *fakeDB) Stats(context.Context) (*store.Stats, error) {
	f.calls++
	if f.failWith != nil {
		return nil, f.failWith
	}
	if f.stats == nil {
		return &store.Stats{Grades: map[string]int{}}, nil
	}
	return f.stats, nil
}

func (f *fakeDB) States(context.Context) ([]string, error) { f.calls++; return f.states, nil }

func (f *fakeDB) EnqueueScan(_ context.Context, agencyID int64) error {
	f.calls++
	if f.failWith != nil {
		return f.failWith
	}
	f.queued = append(f.queued, agencyID)
	return nil
}

func (f *fakeDB) Usage(_ context.Context, days int) (*store.UsageSummary, error) {
	f.usageDays = days
	if f.failWith != nil {
		return nil, f.failWith
	}
	if f.usage == nil {
		return &store.UsageSummary{}, nil
	}
	return f.usage, nil
}

var errBoom = errors.New("database unreachable")
