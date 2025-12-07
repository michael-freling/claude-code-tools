package templates

import (
	"embed"
)

//go:embed all:prompts all:workflow
var FS embed.FS
