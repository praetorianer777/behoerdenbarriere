package crawler_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/crawler"
)

// Jedes Bundesministerium liegt unter bund.de: bmi.bund.de und bmf.bund.de sind für uns
// zwei Behörden und für den Betreiber eine Maschine. Ein Limiter je Lauf würde bei zwei
// gleichzeitigen Prüfungen unbemerkt den doppelten Takt erzeugen — und unser eigenes
// Versprechen im README brechen.
func TestHostLimiterIsSharedAcrossScans(t *testing.T) {
	limiter := crawler.NewHostLimiter()
	ctx := context.Background()
	interval := 50 * time.Millisecond

	// Der erste Zugriff ist frei (der Eimer ist voll), die folgenden kosten je ein
	// Intervall. Vier Anfragen an denselben Host aus zwei „Läufen" brauchen also
	// mindestens drei Intervalle.
	started := time.Now()
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 2 {
				if err := limiter.Wait(ctx, "www.bmi.bund.de", interval); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()

	if elapsed := time.Since(started); elapsed < 3*interval {
		t.Errorf("four requests to one host took %v, want at least %v", elapsed, 3*interval)
	}
}

// Zwei verschiedene Hosts warten nicht aufeinander: Sie sind in aller Regel zwei
// Server, und alle auszubremsen wäre Rücksicht, um die niemand gebeten hat.
func TestHostLimiterDoesNotSlowDownOtherHosts(t *testing.T) {
	limiter := crawler.NewHostLimiter()
	ctx := context.Background()

	started := time.Now()
	for _, host := range []string{"a.de", "b.de", "c.de", "d.de"} {
		if err := limiter.Wait(ctx, host, time.Second); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(started); elapsed > 200*time.Millisecond {
		t.Errorf("four different hosts took %v — they were waiting for each other", elapsed)
	}
}

// robots.txt kann einen langsameren Takt verlangen. Wer den einmal gelesen hat, darf
// ihn nicht dadurch verlieren, dass ein anderer Lauf denselben Host schneller anfragt.
func TestHostLimiterKeepsTheSlowestInterval(t *testing.T) {
	limiter := crawler.NewHostLimiter()
	ctx := context.Background()

	// Erst langsam eintragen, dann schnell anfragen.
	if err := limiter.Wait(ctx, "rki.de", 120*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if err := limiter.Wait(ctx, "rki.de", time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed < 100*time.Millisecond {
		t.Errorf("the slower interval was lost: waited %v", elapsed)
	}
}

func TestHostLimiterStopsWithTheContext(t *testing.T) {
	limiter := crawler.NewHostLimiter()
	ctx, cancel := context.WithCancel(context.Background())

	if err := limiter.Wait(ctx, "a.de", time.Hour); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := limiter.Wait(ctx, "a.de", time.Hour); err == nil {
		t.Error("a cancelled scan must not keep waiting for an hour")
	}
}
