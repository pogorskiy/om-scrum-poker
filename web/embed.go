package web

import "embed"

// DistFS embeds the built frontend assets from the dist directory.
// The directory must exist at compile time; a .gitkeep file ensures
// it's present even before the first build.
//
//go:embed dist
var DistFS embed.FS
