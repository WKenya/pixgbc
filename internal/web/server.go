package web

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"io/fs"
	"net/http"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/ioimg"
	"github.com/WKenya/pixgbc/internal/palette"
	"github.com/WKenya/pixgbc/internal/source"
	webui "github.com/WKenya/pixgbc/web"
)

type Server struct {
	engine core.Engine
	limits ioimg.Limits
}

func NewServer(engine core.Engine, limits ioimg.Limits) http.Handler {
	server := &Server{engine: engine, limits: limits}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", server.handleHealth)
	mux.HandleFunc("/api/palettes", server.handlePalettes)
	mux.HandleFunc("/api/render", server.handleRender)
	mux.Handle("/", server.staticHandler())
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	_, _ = io.WriteString(w, "ok\n")
}

func (s *Server) handlePalettes(w http.ResponseWriter, _ *http.Request) {
	type response struct {
		Key         string   `json:"key"`
		DisplayName string   `json:"display_name"`
		Description string   `json:"description"`
		Colors      []string `json:"colors"`
	}

	presets := palette.AllPresets()
	out := make([]response, 0, len(presets))
	for _, preset := range presets {
		colors := make([]string, 0, len(preset.Colors))
		for _, c := range preset.Colors {
			colors = append(colors, colorHex(c))
		}
		out = append(out, response{
			Key:         preset.Key,
			DisplayName: preset.DisplayName,
			Description: preset.Description,
			Colors:      colors,
		})
	}

	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.limits.MaxFileBytes)
	if err := r.ParseMultipartForm(s.limits.MaxFileBytes); err != nil {
		http.Error(w, fmt.Sprintf("parse upload: %v", err), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing form file field 'file'", http.StatusBadRequest)
		return
	}
	defer file.Close()

	decoded, err := ioimg.DecodeImage(file, s.limits)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := core.Config{
		PalettePreset: r.FormValue("palette"),
	}
	if cfg.PalettePreset == "" {
		cfg.PalettePreset = core.DefaultConfig().PalettePreset
	}

	result, err := s.engine.Run(r.Context(), source.NewSingleImage(decoded.Image, decoded.Meta), cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	if err := ioimg.EncodePNG(w, result.PreviewImage); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) staticHandler() http.Handler {
	sub, err := fs.Sub(webui.FS, ".")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		})
	}

	return http.FileServer(http.FS(sub))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func colorHex(c color.NRGBA) string {
	buf := []byte{c.R, c.G, c.B}
	return "#" + hex.EncodeToString(buf)
}
