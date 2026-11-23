// Package version holds the build identity of the openjob binary. The
// variables are overridden at link time (see .goreleaser.yaml, Dockerfile and
// Makefile) with:
//
//	-ldflags "-X github.com/maicondmenezes/openjob/internal/version.Version=1.2.3 ..."
//
// A plain `go build` keeps the "dev" defaults, so local binaries are never
// mistaken for a release (CP11, docs/adr/0011).
package version

import "fmt"

// Overridden via -ldflags -X. Keep them plain string variables (not
// constants) or the linker cannot rewrite them.
var (
	// Version is the semantic version without the leading "v" (e.g. "1.2.3").
	Version = "dev"
	// Commit is the Git commit the binary was built from.
	Commit = "none"
	// Date is the build timestamp (RFC 3339, UTC).
	Date = "unknown"
)

// String renders the one-line identity printed by `openjob version`.
func String() string {
	return fmt.Sprintf("openjob %s (commit %s, built %s)", Version, Commit, Date)
}
