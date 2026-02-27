package frontend

import "embed"

//go:embed all:out
var StaticFiles embed.FS
