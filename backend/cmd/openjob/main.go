// openJob — local-first job application copilot.
//
// One static binary (CGO off, SPA embedded via //go:embed) with four commands:
// `openjob` (desktop), `openjob serve` (headless), `openjob version` and
// `openjob healthcheck`. Process settings come from flags or OPENJOB_*
// environment variables (flag > env > default); everything functional is
// configured later in the web app (CP18/CP19). See docs/plan/11-package-release.md
// and docs/adr/0011, 0012.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := newApp(os.Stdout, os.Stderr).run(ctx, os.Args[1:])
	stop()
	os.Exit(code)
}
