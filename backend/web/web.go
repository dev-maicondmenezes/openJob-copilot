// Package web embeds the built SPA assets (backend/web/dist) so the openjob
// binary is fully self-contained — the Docker runtime stage ships only the
// static binary (spec §5, //go:embed).
//
// In Checkpoint 2 the dist folder is produced by the SvelteKit build
// (adapter-static, fallback: index.html); for Checkpoint 1 it holds the
// foundation placeholder page.
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
