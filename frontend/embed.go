package frontend

import "embed"

//go:generate npm run build
//go:embed all:dist
var Content embed.FS
