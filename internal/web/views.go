package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/WKenya/pixgbc/internal/review"
)

type reviewRow struct {
	Label string
	Value string
}

type reviewPaletteBank struct {
	Label  string
	Colors []string
}

type reviewBankSummary struct {
	Label     string
	TileCount int
	Percent   int
	Colors    []string
}

type reviewPageData struct {
	Record          review.ReviewRecord
	RecordURL       string
	PreviewURL      string
	FinalURL        string
	DebugURL        string
	RecordJSON      string
	MetadataJSON    string
	SourceRows      []reviewRow
	ConfigRows      []reviewRow
	FingerprintRows []reviewRow
	GlobalPalette   []string
	PaletteBanks    []reviewPaletteBank
	BankSummaries   []reviewBankSummary
}

var reviewTemplate = template.Must(template.New("review").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>pixgbc review {{.Record.ID}}</title>
    <style>
      :root { color-scheme: light; }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        font-family: "Iowan Old Style", Georgia, serif;
        background:
          radial-gradient(circle at top, rgba(119, 141, 69, 0.14), transparent 34%),
          linear-gradient(180deg, #f4efe4 0%, #ece4d2 100%);
        color: #1d2417;
      }
      main { width: min(1180px, calc(100vw - 32px)); margin: 24px auto 48px; }
      .panel {
        background: rgba(250, 246, 238, .96);
        border: 2px solid #28331f;
        padding: 18px;
        margin-bottom: 20px;
        box-shadow: 10px 10px 0 rgba(40, 51, 31, .08);
      }
      html[data-debug-ui="off"] .debug-only { display: none !important; }
      h1, h2, h3, p { margin-top: 0; }
      h1 { margin-bottom: 10px; }
      h2 { margin-bottom: 14px; }
      .lede { color: #4b5a37; }
      .quick {
        display: grid;
        gap: 12px;
        grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
        margin: 18px 0;
      }
      .stat {
        background: #f0e7d5;
        border: 1px solid #7d8c5b;
        padding: 12px;
      }
      .stat strong, .stat span { display: block; }
      .stat span { font-size: 12px; text-transform: uppercase; letter-spacing: .08em; color: #5b6e1d; }
      .links { display: flex; flex-wrap: wrap; gap: 10px; }
      .links a { color: #5b6e1d; }
      .images { display: grid; gap: 20px; }
      .images img, .debug img {
        width: 100%;
        image-rendering: pixelated;
        border: 2px solid #28331f;
        background: #fff;
      }
      .meta-grid {
        display: grid;
        gap: 18px;
      }
      .kv {
        display: grid;
        grid-template-columns: minmax(110px, 180px) 1fr;
        gap: 8px 12px;
        margin: 0;
      }
      .kv dt {
        font-size: 12px;
        letter-spacing: .08em;
        text-transform: uppercase;
        color: #5b6e1d;
      }
      .kv dd { margin: 0; word-break: break-word; }
      .swatches {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
      }
      .swatch {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        border: 1px solid #7d8c5b;
        background: #f6f0e5;
        padding: 6px 8px;
      }
      .chip {
        width: 22px;
        height: 22px;
        border: 1px solid rgba(0,0,0,.35);
        background: var(--swatch);
      }
      .bank-list {
        display: grid;
        gap: 14px;
      }
      .bank-card {
        border: 1px solid #7d8c5b;
        background: #f3ead9;
        padding: 12px;
      }
      .bank-head {
        display: flex;
        justify-content: space-between;
        gap: 12px;
        align-items: baseline;
        margin-bottom: 10px;
      }
      .bank-meter {
        width: 100%;
        height: 12px;
        background: #d8cfbf;
        border: 1px solid #7d8c5b;
        margin-bottom: 10px;
      }
      .bank-meter > span {
        display: block;
        height: 100%;
        background: linear-gradient(90deg, #5b6e1d 0%, #8ba652 100%);
      }
      pre {
        overflow: auto;
        white-space: pre-wrap;
        word-break: break-word;
        background: #efe5d3;
        border: 1px solid #7d8c5b;
        padding: 12px;
      }
      @media (min-width: 860px) {
        .images { grid-template-columns: 1fr 1fr; }
        .meta-grid { grid-template-columns: 1fr 1fr 1fr; }
      }
    </style>
  </head>
  <body>
    <main>
      <section class="panel">
        <h1>Render {{.Record.ID}}</h1>
        <p class="lede">Mode {{.Record.Mode}}. {{.Record.OutputWidth}}x{{.Record.OutputHeight}} output. Created {{.Record.CreatedAt.Format "2006-01-02 15:04:05 MST"}}.</p>
        <div class="quick">
          <div class="stat"><span>Source</span><strong>{{.Record.Source.Width}}x{{.Record.Source.Height}}</strong></div>
          <div class="stat"><span>Format</span><strong>{{.Record.Source.Format}}</strong></div>
          <div class="stat"><span>Palette Banks</span><strong>{{len .Record.PaletteBanks}}</strong></div>
          <div class="stat"><span>Tiles</span><strong>{{len .Record.TileAssignments}}</strong></div>
        </div>
        <p class="links"><a href="{{.PreviewURL}}">preview.png</a> <a href="{{.FinalURL}}">final.png</a> <a class="debug-only" href="{{.RecordURL}}">record.json</a>{{if .DebugURL}} <a class="debug-only" href="{{.DebugURL}}">debug.png</a>{{end}}</p>
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
      <section class="panel debug debug-only">
        <h2>Debug Sheet</h2>
        <img src="{{.DebugURL}}" alt="Debug sheet">
      </section>
      {{end}}

      <section class="panel meta-grid">
        <div>
          <h2>Source</h2>
          <dl class="kv">
            {{range .SourceRows}}
            <dt>{{.Label}}</dt><dd>{{.Value}}</dd>
            {{end}}
          </dl>
        </div>
        <div>
          <h2>Config</h2>
          <dl class="kv">
            {{range .ConfigRows}}
            <dt>{{.Label}}</dt><dd>{{.Value}}</dd>
            {{end}}
          </dl>
        </div>
        <div class="debug-only">
          <h2>Fingerprints</h2>
          <dl class="kv">
            {{range .FingerprintRows}}
            <dt>{{.Label}}</dt><dd>{{.Value}}</dd>
            {{end}}
          </dl>
        </div>
      </section>

      {{if .GlobalPalette}}
      <section class="panel">
        <h2>Global Palette</h2>
        <div class="swatches">
          {{range .GlobalPalette}}
          <div class="swatch"><span class="chip" style="--swatch: {{.}}"></span><code>{{.}}</code></div>
          {{end}}
        </div>
      </section>
      {{end}}

      {{if .PaletteBanks}}
      <section class="panel">
        <h2>Palette Banks</h2>
        <div class="bank-list">
          {{range .PaletteBanks}}
          <div class="bank-card">
            <div class="bank-head"><h3>{{.Label}}</h3></div>
            <div class="swatches">
              {{range .Colors}}
              <div class="swatch"><span class="chip" style="--swatch: {{.}}"></span><code>{{.}}</code></div>
              {{end}}
            </div>
          </div>
          {{end}}
        </div>
      </section>
      {{end}}

      {{if .BankSummaries}}
      <section class="panel">
        <h2>Tile Bank Distribution</h2>
        <div class="bank-list">
          {{range .BankSummaries}}
          <div class="bank-card">
            <div class="bank-head">
              <strong>{{.Label}}</strong>
              <span>{{.TileCount}} tiles</span>
            </div>
            <div class="bank-meter"><span style="width: {{.Percent}}%;"></span></div>
            <div class="swatches">
              {{range .Colors}}
              <div class="swatch"><span class="chip" style="--swatch: {{.}}"></span><code>{{.}}</code></div>
              {{end}}
            </div>
          </div>
          {{end}}
        </div>
      </section>
      {{end}}

      {{if .MetadataJSON}}
      <section class="panel debug-only">
        <h2>Metadata</h2>
        <pre>{{.MetadataJSON}}</pre>
      </section>
      {{end}}

      <section class="panel debug-only">
        <h2>Record JSON</h2>
        <pre>{{.RecordJSON}}</pre>
      </section>
    </main>
    <script>
      (function () {
        const storageKey = "pixgbc.debug-ui";
        const host = window.location.hostname;
        const loopback =
          host === "localhost" ||
          host === "127.0.0.1" ||
          host === "::1" ||
          host === "[::1]" ||
          host === "0.0.0.0" ||
          host.endsWith(".localhost");

        function enabled() {
          return loopback || window.localStorage.getItem(storageKey) === "1";
        }

        function sync() {
          document.documentElement.dataset.debugUi = enabled() ? "on" : "off";
        }

        document.addEventListener("keydown", function (event) {
          if (event.repeat || event.metaKey || event.ctrlKey || !event.altKey || !event.shiftKey) {
            return;
          }
          if (event.key.toLowerCase() !== "d") {
            return;
          }
          event.preventDefault();
          if (!loopback) {
            if (enabled()) {
              window.localStorage.removeItem(storageKey);
            } else {
              window.localStorage.setItem(storageKey, "1");
            }
          }
          sync();
        });

        sync();
      })();
    </script>
  </body>
</html>`))

func renderReviewPage(record review.ReviewRecord, recordURL, previewURL, finalURL, debugURL string) ([]byte, error) {
	jsonBytes, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, err
	}

	var metadataJSON string
	if len(record.Metadata) > 0 {
		metadataBytes, err := json.MarshalIndent(record.Metadata, "", "  ")
		if err != nil {
			return nil, err
		}
		metadataJSON = string(metadataBytes)
	}

	var buf bytes.Buffer
	if err := reviewTemplate.Execute(&buf, reviewPageData{
		Record:          record,
		RecordURL:       recordURL,
		PreviewURL:      previewURL,
		FinalURL:        finalURL,
		DebugURL:        debugURL,
		RecordJSON:      string(jsonBytes),
		MetadataJSON:    metadataJSON,
		SourceRows:      buildSourceRows(record),
		ConfigRows:      buildConfigRows(record),
		FingerprintRows: buildFingerprintRows(record),
		GlobalPalette:   append([]string(nil), record.GlobalPalette...),
		PaletteBanks:    buildPaletteBanks(record),
		BankSummaries:   buildBankSummaries(record),
	}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func buildSourceRows(record review.ReviewRecord) []reviewRow {
	return []reviewRow{
		{Label: "Width", Value: fmt.Sprintf("%d", record.Source.Width)},
		{Label: "Height", Value: fmt.Sprintf("%d", record.Source.Height)},
		{Label: "Format", Value: record.Source.Format},
		{Label: "File Size", Value: fmt.Sprintf("%d", record.Source.FileSize)},
		{Label: "Frames", Value: fmt.Sprintf("%d", record.Source.FrameCount)},
		{Label: "Alpha", Value: fmt.Sprintf("%t", record.Source.HasAlpha)},
	}
}

func buildConfigRows(record review.ReviewRecord) []reviewRow {
	cfg := record.Config
	return []reviewRow{
		{Label: "Mode", Value: record.Mode},
		{Label: "Size", Value: fmt.Sprintf("%dx%d", cfg.TargetWidth, cfg.TargetHeight)},
		{Label: "Palette", Value: string(cfg.PaletteStrategy) + " / " + cfg.PalettePreset},
		{Label: "Dither", Value: string(cfg.Dither)},
		{Label: "Crop", Value: string(cfg.CropMode)},
		{Label: "Alpha", Value: string(cfg.AlphaMode)},
		{Label: "BG", Value: fmt.Sprintf("#%02x%02x%02x", cfg.BackgroundColor.R, cfg.BackgroundColor.G, cfg.BackgroundColor.B)},
		{Label: "Tone", Value: fmt.Sprintf("b=%0.2f c=%0.2f g=%0.2f", cfg.Brightness, cfg.Contrast, cfg.Gamma)},
		{Label: "Preview", Value: fmt.Sprintf("%dx", cfg.PreviewScale)},
		{Label: "Tile Size", Value: fmt.Sprintf("%d", cfg.TileSize)},
		{Label: "Colors/Tile", Value: fmt.Sprintf("%d", cfg.ColorsPerTile)},
		{Label: "Max Palettes", Value: fmt.Sprintf("%d", cfg.MaxPalettes)},
	}
}

func buildFingerprintRows(record review.ReviewRecord) []reviewRow {
	return []reviewRow{
		{Label: "Input", Value: record.Fingerprints.InputSHA256},
		{Label: "Config", Value: record.Fingerprints.ConfigSHA256},
		{Label: "Output", Value: record.Fingerprints.OutputSHA256},
	}
}

func buildPaletteBanks(record review.ReviewRecord) []reviewPaletteBank {
	out := make([]reviewPaletteBank, 0, len(record.PaletteBanks))
	for i, colors := range record.PaletteBanks {
		out = append(out, reviewPaletteBank{
			Label:  fmt.Sprintf("Bank %d", i),
			Colors: append([]string(nil), colors...),
		})
	}
	return out
}

func buildBankSummaries(record review.ReviewRecord) []reviewBankSummary {
	if len(record.PaletteBanks) == 0 || len(record.TileAssignments) == 0 {
		return nil
	}

	counts := make([]int, len(record.PaletteBanks))
	for _, assignment := range record.TileAssignments {
		if assignment.PaletteBank < 0 || assignment.PaletteBank >= len(counts) {
			continue
		}
		counts[assignment.PaletteBank]++
	}

	total := len(record.TileAssignments)
	out := make([]reviewBankSummary, 0, len(record.PaletteBanks))
	for i, colors := range record.PaletteBanks {
		out = append(out, reviewBankSummary{
			Label:     fmt.Sprintf("Bank %d", i),
			TileCount: counts[i],
			Percent:   percent(counts[i], total),
			Colors:    append([]string(nil), colors...),
		})
	}
	return out
}

func percent(count, total int) int {
	if total <= 0 {
		return 0
	}
	return int(float64(count) / float64(total) * 100)
}
