package version_test

import (
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/version"
)

// Whatever the build did or did not record, the answer has to be something a person can
// read: a version that is silently empty is worse than one that says it does not know.
func TestCurrentIsNeverEmpty(t *testing.T) {
	if version.Current() == "" {
		t.Error("the version is empty")
	}
	if version.Short() == "" {
		t.Error("the short version is empty")
	}
}

func TestShortCutsToSevenCharacters(t *testing.T) {
	original := version.Revision
	t.Cleanup(func() { version.Revision = original })

	version.Revision = "8f2626f1d3c4b5a697887766554433221100aabb"
	if got := version.Short(); got != "8f2626f" {
		t.Errorf("short = %q, want 8f2626f", got)
	}

	// "unknown" is not a hash and must not be cut into something that looks like one.
	version.Revision = ""
	if got := version.Short(); got != "unknown" && len(got) != 7 {
		t.Errorf("short = %q", got)
	}
}
