// Package web embeds the built frontend (this directory's dist/, produced by
// `npm run build`) into the Go binary. go:embed patterns can't contain "..",
// so this file has to live inside web/ itself — a file under cmd/serve/
// cannot reach up into a sibling directory. See cmd/serve/main.go for how
// Dist is consumed.
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
