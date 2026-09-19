package crawler

// Target is a page waiting to be checked.
type Target struct {
	URL      string
	Depth    int
	IsEntry  bool
	Priority bool
}

// frontier holds the pages still to visit. Pages that matter legally — the
// accessibility statement, contact, forms — are pulled forward, because a crawl that
// runs into its page budget should have spent it on those and not on the tenth press
// release.
type frontier struct {
	priority []Target
	normal   []Target
	seen     map[string]bool
}

func newFrontier() *frontier {
	return &frontier{seen: map[string]bool{}}
}

// push adds a target unless its URL has been seen before. It reports whether the
// target was taken.
func (f *frontier) push(t Target) bool {
	if t.URL == "" || f.seen[t.URL] {
		return false
	}
	f.seen[t.URL] = true
	if t.Priority || t.IsEntry {
		f.priority = append(f.priority, t)
	} else {
		f.normal = append(f.normal, t)
	}
	return true
}

func (f *frontier) pop() (Target, bool) {
	if len(f.priority) > 0 {
		t := f.priority[0]
		f.priority = f.priority[1:]
		return t, true
	}
	if len(f.normal) > 0 {
		t := f.normal[0]
		f.normal = f.normal[1:]
		return t, true
	}
	return Target{}, false
}

func (f *frontier) empty() bool { return len(f.priority) == 0 && len(f.normal) == 0 }
