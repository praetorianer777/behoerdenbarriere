package crawler

import "testing"

func TestFrontierServesPriorityFirst(t *testing.T) {
	f := newFrontier()
	f.push(Target{URL: "https://a.de/", IsEntry: true})
	f.push(Target{URL: "https://a.de/presse"})
	f.push(Target{URL: "https://a.de/barrierefreiheit", Priority: true})
	f.push(Target{URL: "https://a.de/aktuelles"})

	var order []string
	for {
		target, ok := f.pop()
		if !ok {
			break
		}
		order = append(order, target.URL)
	}

	// Entry page and the legally relevant pages come before the press releases: if the
	// page budget runs out, it should have gone to those.
	want := []string{
		"https://a.de/", "https://a.de/barrierefreiheit",
		"https://a.de/presse", "https://a.de/aktuelles",
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

func TestFrontierDeduplicates(t *testing.T) {
	f := newFrontier()
	if !f.push(Target{URL: "https://a.de/x"}) {
		t.Fatal("first push refused")
	}
	if f.push(Target{URL: "https://a.de/x"}) {
		t.Fatal("duplicate accepted")
	}
	if f.push(Target{URL: ""}) {
		t.Fatal("empty URL accepted")
	}

	if _, ok := f.pop(); !ok {
		t.Fatal("nothing to pop")
	}
	if _, ok := f.pop(); ok {
		t.Fatal("the duplicate came back")
	}
}

// A page that has been served must not come back through a link from another page;
// otherwise a footer link would keep the crawl going in circles.
func TestFrontierRemembersServedTargets(t *testing.T) {
	f := newFrontier()
	f.push(Target{URL: "https://a.de/impressum"})
	f.pop()
	if f.push(Target{URL: "https://a.de/impressum"}) {
		t.Fatal("an already served page was queued again")
	}
	if !f.empty() {
		t.Fatal("frontier is not empty")
	}
}
