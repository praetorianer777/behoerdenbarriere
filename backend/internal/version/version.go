// Package version says which build is running.
//
// A container tagged "latest" says nothing about what is inside it: the tag moves with
// every release, and an installation that kept an older copy looks exactly like a
// current one. That is not a theoretical worry — it cost an afternoon of looking for a
// tool in an image that had been published an hour earlier.
package version

import "runtime/debug"

// Revision is set at build time:
//
//	go build -ldflags "-X …/internal/version.Revision=$(git rev-parse HEAD)"
//
// Left unset, the build information Go records itself is used — that works for a local
// build from a checkout and not in a container, where the source arrives without its
// history.
var Revision = ""

// Revision returns the commit this binary was built from, or "unknown".
func Current() string {
	if Revision != "" {
		return Revision
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			return setting.Value
		}
	}
	return "unknown"
}

// Short is the revision in the length people actually compare — the first seven
// characters, the same as git and GitHub show.
func Short() string {
	revision := Current()
	if len(revision) > 7 && revision != "unknown" {
		return revision[:7]
	}
	return revision
}
