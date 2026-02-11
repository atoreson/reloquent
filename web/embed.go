package web

import "embed"

// DistFS contains the built React app files.
// The dist/ directory is populated by `npm run build` in the web/ directory.
//
//go:embed all:dist
var DistFS embed.FS
