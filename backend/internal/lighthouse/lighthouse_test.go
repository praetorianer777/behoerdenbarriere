package lighthouse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuditReadsTheScore(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audit" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		got = string(buf[:n])
		_, _ = w.Write([]byte(`{"score":97,"failed_audits":["color-contrast"],"lighthouse_version":"12.8.2"}`))
	}))
	defer srv.Close()

	result, err := New(srv.URL, time.Second).Audit(context.Background(), "https://www.bund.de")
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if result.Score != 97 || len(result.FailedAudits) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(got, "bund.de") {
		t.Errorf("the URL was not passed on: %s", got)
	}
}

// Without a configured service the client is nil, and a nil client answers "no
// result" — the Lighthouse number is an addition and must not cost us a scan.
func TestNilClientIsQuiet(t *testing.T) {
	result, err := New("", time.Second).Audit(context.Background(), "https://example.org")
	if err != nil || result != nil {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
}

func TestAuditReportsServiceFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"chrome ist abgestürzt"}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL, time.Second).Audit(context.Background(), "https://example.org")
	if err == nil {
		t.Fatal("no error")
	}
	if !strings.Contains(err.Error(), "abgestürzt") {
		t.Errorf("the cause is missing: %v", err)
	}
}

// A number outside the scale means the service answered something we do not
// understand, and a wrong score is worse than none.
func TestAuditRejectsAnImpossibleScore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"score":420}`))
	}))
	defer srv.Close()

	if _, err := New(srv.URL, time.Second).Audit(context.Background(), "https://example.org"); err == nil {
		t.Fatal("an impossible score was accepted")
	}
}

// A Lighthouse run takes seconds, sometimes longer. The caller's deadline has to end
// the wait, or a hanging service would hold the scan of an authority.
func TestAuditRespectsTheDeadline(t *testing.T) {
	slow := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-slow:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() {
		close(slow)
		srv.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	started := time.Now()
	if _, err := New(srv.URL, time.Minute).Audit(ctx, "https://example.org"); err == nil {
		t.Fatal("the call did not end with the context")
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("the call waited %v despite the deadline", elapsed)
	}
}
