package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/ioimg"
	"github.com/WKenya/pixgbc/internal/palette"
	"github.com/WKenya/pixgbc/internal/review"
	"github.com/WKenya/pixgbc/internal/source"
	webui "github.com/WKenya/pixgbc/web"
)

type Server struct {
	engine core.Engine
	limits ioimg.Limits
	store  review.Store
}

func NewServer(engine core.Engine, limits ioimg.Limits) (http.Handler, error) {
	store, err := review.NewTempStore("", 7*24*time.Hour)
	if err != nil {
		return nil, err
	}
	return NewServerWithStore(engine, limits, store), nil
}

func NewServerWithStore(engine core.Engine, limits ioimg.Limits, store review.Store) http.Handler {
	server := &Server{engine: engine, limits: limits, store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.handleHealth)
	mux.HandleFunc("GET /api/palettes", server.handlePalettes)
	mux.HandleFunc("POST /api/render", server.handleRender)
	mux.HandleFunc("GET /api/renders/{id}", server.handleGetRecord)
	mux.HandleFunc("GET /api/renders/{id}/artifacts/{name}", server.handleGetArtifact)
	mux.HandleFunc("GET /renders/{id}", server.handleReviewPage)
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

	inputBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("read upload: %v", err), http.StatusBadRequest)
		return
	}

	decoded, err := ioimg.DecodeImage(bytes.NewReader(inputBytes), s.limits)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := core.Config{
		PalettePreset: r.FormValue("palette"),
		Mode:          core.Mode(r.FormValue("mode")),
		EmitDebug:     r.FormValue("debug") == "1" || r.FormValue("debug") == "true",
	}
	if cfg.PalettePreset == "" {
		cfg.PalettePreset = core.DefaultConfig().PalettePreset
	}
	if cfg.Mode == "" {
		cfg.Mode = core.ModeRelaxed
	}

	result, err := s.engine.Run(r.Context(), source.NewSingleImage(decoded.Image, decoded.Meta), cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	record, err := review.SaveResult(r.Context(), s.store, inputBytes, cfg, result)
	if err != nil {
		http.Error(w, fmt.Sprintf("save review: %v", err), http.StatusInternalServerError)
		return
	}

	response := RenderResponse{
		ID:         record.ID,
		ReviewURL:  s.reviewURL(record.ID),
		RecordURL:  s.recordURL(record.ID),
		PreviewURL: s.artifactURL(record.ID, record.Artifacts.PreviewPNG),
		FinalURL:   s.artifactURL(record.ID, record.Artifacts.FinalPNG),
	}
	if record.Artifacts.DebugPNG != "" {
		response.DebugURL = s.artifactURL(record.ID, record.Artifacts.DebugPNG)
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetRecord(w http.ResponseWriter, r *http.Request) {
	record, err := s.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if err == core.ErrReviewNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func (s *Server) handleGetArtifact(w http.ResponseWriter, r *http.Request) {
	reader, err := s.store.OpenArtifact(r.Context(), r.PathValue("id"), r.PathValue("name"))
	if err != nil {
		status := http.StatusInternalServerError
		if err == core.ErrReviewNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", contentTypeForArtifact(r.PathValue("name")))
	http.ServeContent(w, r, r.PathValue("name"), time.Time{}, reader)
}

func (s *Server) handleReviewPage(w http.ResponseWriter, r *http.Request) {
	record, err := s.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if err == core.ErrReviewNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	debugURL := ""
	if record.Artifacts.DebugPNG != "" {
		debugURL = s.artifactURL(record.ID, record.Artifacts.DebugPNG)
	}

	page, err := renderReviewPage(
		record,
		s.recordURL(record.ID),
		s.artifactURL(record.ID, record.Artifacts.PreviewPNG),
		s.artifactURL(record.ID, record.Artifacts.FinalPNG),
		debugURL,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(page)
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

func (s *Server) recordURL(id string) string {
	return "/api/renders/" + id
}

func (s *Server) artifactURL(id, name string) string {
	return path.Join("/api/renders", id, "artifacts", name)
}

func (s *Server) reviewURL(id string) string {
	return "/renders/" + id
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func colorHex(c color.NRGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func contentTypeForArtifact(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".png":
		return "image/png"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
