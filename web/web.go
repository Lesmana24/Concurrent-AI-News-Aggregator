package web

import "embed"

// DistFS embeds the production-built React + Vite SPA assets.
//
//go:embed all:dist
var DistFS embed.FS

