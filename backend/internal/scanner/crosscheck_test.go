package scanner

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// Compares our findings with the reference implementation: the same axe version, the
// same tags, the same page, run through @axe-core/cli in a browser of its own.
//
// The point is not the score — nobody else computes that — but the layer underneath.
// If the injection, the tag filter or the settling period were wrong, we would report
// fewer violations than the reference and every score would come out too kind. That is
// the failure this test is here to catch, so missing rules fail it while extra ones are
// only logged.
//
// It downloads a package and drives a second browser, so it runs on request only:
// AXE_CLI_CROSSCHECK=1 go test ./internal/scanner/ -run CrossCheck
func TestCrossCheckAgainstAxeCLI(t *testing.T) {
	if os.Getenv("AXE_CLI_CROSSCHECK") == "" {
		t.Skip("AXE_CLI_CROSSCHECK not set")
	}
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not available")
	}

	scanner := newTestScanner(t)
	url := servePage(t, brokenPage)

	ours := map[string]bool{}
	for _, v := range scanner.Scan(context.Background(), url).Result.Violations {
		ours[v.RuleID] = true
	}
	theirs := axeCLIRules(t, url)

	if len(theirs) == 0 {
		t.Fatal("the reference found nothing — then the comparison proves nothing")
	}

	var missing []string
	for rule := range theirs {
		if !ours[rule] {
			missing = append(missing, rule)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("the reference reports rules we miss: %v (ours: %v)", missing, sorted(ours))
	}

	var extra []string
	for rule := range ours {
		if !theirs[rule] {
			extra = append(extra, rule)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Logf("we report rules the reference does not: %v", extra)
	}
}

func axeCLIRules(t *testing.T, url string) map[string]bool {
	t.Helper()
	// The CLI resolves --save relative to its working directory and mangles an
	// absolute path, so it is handed a bare file name and run inside the temp dir.
	dir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	args := []string{"-y", "@axe-core/cli@4.10.2", url,
		"--tags", "wcag2a,wcag2aa,wcag21a,wcag21aa", "--save", "axe.json"}
	if driver := os.Getenv("AXE_CHROMEDRIVER"); driver != "" {
		args = append(args, "--chromedriver-path", driver)
	}

	cmd := exec.CommandContext(ctx, "npx", args...)
	cmd.Dir = dir
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("@axe-core/cli could not be run: %v\n%s", err, combined)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "axe.json"))
	if err != nil {
		t.Fatalf("reading the reference result: %v", err)
	}
	var report []struct {
		Violations []struct {
			ID string `json:"id"`
		} `json:"violations"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("reference result: %v", err)
	}

	rules := map[string]bool{}
	for _, page := range report {
		for _, v := range page.Violations {
			rules[v.ID] = true
		}
	}
	return rules
}

func sorted(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
