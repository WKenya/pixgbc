package web

import (
	"bytes"
	"encoding/json"
	"html/template"

	"github.com/WKenya/pixgbc/internal/review"
)

var reviewTemplate = template.Must(template.New("review").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>pixgbc review {{.Record.ID}}</title>
    <style>
      :root { color-scheme: light; }
      body { margin: 0; font-family: "Iowan Old Style", Georgia, serif; background: #f4efe4; color: #1d2417; }
      main { width: min(1100px, calc(100vw - 32px)); margin: 24px auto 48px; }
      .panel { background: rgba(250,246,238,.96); border: 2px solid #28331f; padding: 18px; margin-bottom: 20px; box-shadow: 10px 10px 0 rgba(40,51,31,.08); }
      h1, h2, p { margin-top: 0; }
      .images { display: grid; gap: 20px; }
      .images img { width: 100%; image-rendering: pixelated; border: 2px solid #28331f; background: #fff; }
      pre { overflow: auto; white-space: pre-wrap; word-break: break-word; }
      a { color: #5b6e1d; }
      @media (min-width: 860px) { .images { grid-template-columns: 1fr 1fr; } }
    </style>
  </head>
  <body>
    <main>
      <section class="panel">
        <h1>Render {{.Record.ID}}</h1>
        <p>Mode: {{.Record.Mode}}. Output: {{.Record.OutputWidth}}x{{.Record.OutputHeight}}.</p>
        <p><a href="{{.PreviewURL}}">preview.png</a> · <a href="{{.FinalURL}}">final.png</a> · <a href="{{.RecordURL}}">record.json</a></p>
      </section>
      <section class="panel images">
        <div>
          <h2>Preview</h2>
          <img src="{{.PreviewURL}}" alt="Preview image">
        </div>
        <div>
          <h2>Final</h2>
          <img src="{{.FinalURL}}" alt="Final image">
        </div>
      </section>
      {{if .DebugURL}}
      <section class="panel">
        <h2>Debug Sheet</h2>
        <img src="{{.DebugURL}}" alt="Debug sheet" style="width: 100%; image-rendering: pixelated; border: 2px solid #28331f; background: #fff;">
      </section>
      {{end}}
      <section class="panel">
        <h2>Metadata</h2>
        <pre>{{.RecordJSON}}</pre>
      </section>
    </main>
  </body>
</html>`))

type reviewPageData struct {
	Record     review.ReviewRecord
	RecordURL  string
	PreviewURL string
	FinalURL   string
	DebugURL   string
	RecordJSON string
}

func renderReviewPage(record review.ReviewRecord, recordURL, previewURL, finalURL, debugURL string) ([]byte, error) {
	jsonBytes, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := reviewTemplate.Execute(&buf, reviewPageData{
		Record:     record,
		RecordURL:  recordURL,
		PreviewURL: previewURL,
		FinalURL:   finalURL,
		DebugURL:   debugURL,
		RecordJSON: string(jsonBytes),
	}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
