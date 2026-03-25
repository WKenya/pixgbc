package webui

import "embed"

//go:embed index.html app.js styles.css
var FS embed.FS
