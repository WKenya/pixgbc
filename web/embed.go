package webui

import "embed"

//go:embed index.html app.js browser-mode.js server-mode.js styles.css wasm-client.js wasm-worker.js wasm_exec.js
var FS embed.FS
