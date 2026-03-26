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
	"strconv"
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
	cfg    ServerConfig
}

type ServerConfig struct {
	Token string
}

func NewServer(engine core.Engine, limits ioimg.Limits) (http.Handler, error) {
	return NewServerWithConfig(engine, limits, ServerConfig{}, 7*24*time.Hour)
}

func NewServerWithConfig(engine core.Engine, limits ioimg.Limits, cfg ServerConfig, ttl time.Duration) (http.Handler, error) {
	store, err := review.NewTempStore("", ttl)
	if err != nil {
		return nil, err
	}
	return NewServerWithStore(engine, limits, store, cfg), nil
}

func NewServerWithStore(engine core.Engine, limits ioimg.Limits, store review.Store, cfg ServerConfig) http.Handler {
	server := &Server{engine: engine, limits: limits, store: store, cfg: cfg}
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

func (s *Server) handlePalettes(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
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
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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

	cfg, err := parseRenderConfig(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
		ReviewURL:  s.reviewURL(record.ID, s.requestTokenQuery(r)),
		RecordURL:  s.recordURL(record.ID, s.requestTokenQuery(r)),
		PreviewURL: s.artifactURL(record.ID, record.Artifacts.PreviewPNG, s.requestTokenQuery(r)),
		FinalURL:   s.artifactURL(record.ID, record.Artifacts.FinalPNG, s.requestTokenQuery(r)),
	}
	if record.Artifacts.DebugPNG != "" {
		response.DebugURL = s.artifactURL(record.ID, record.Artifacts.DebugPNG, s.requestTokenQuery(r))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetRecord(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
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
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
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
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
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
		debugURL = s.artifactURL(record.ID, record.Artifacts.DebugPNG, s.requestTokenQuery(r))
	}

	page, err := renderReviewPage(
		record,
		s.recordURL(record.ID, s.requestTokenQuery(r)),
		s.artifactURL(record.ID, record.Artifacts.PreviewPNG, s.requestTokenQuery(r)),
		s.artifactURL(record.ID, record.Artifacts.FinalPNG, s.requestTokenQuery(r)),
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

func (s *Server) recordURL(id string, token string) string {
	return withTokenQuery("/api/renders/"+id, token)
}

func (s *Server) artifactURL(id, name string, token string) string {
	return withTokenQuery(path.Join("/api/renders", id, "artifacts", name), token)
}

func (s *Server) reviewURL(id string, token string) string {
	return withTokenQuery("/renders/"+id, token)
}

func (s *Server) authorized(r *http.Request) bool {
	if s.cfg.Token == "" {
		return true
	}
	if r.URL.Query().Get("token") == s.cfg.Token {
		return true
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") && strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")) == s.cfg.Token {
		return true
	}
	return false
}

func (s *Server) requestTokenQuery(r *http.Request) string {
	if s.cfg.Token == "" {
		return ""
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == s.cfg.Token {
		return token
	}
	return ""
}

func withTokenQuery(urlPath string, token string) string {
	if token == "" {
		return urlPath
	}
	return urlPath + "?token=" + token
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

func parseRenderConfig(r *http.Request) (core.Config, error) {
	defaults := core.DefaultConfig()
	cfg := core.Config{
		Mode:            core.Mode(formValueDefault(r, "mode", string(defaults.Mode))),
		PaletteStrategy: core.PaletteStrategy(formValueDefault(r, "palette_mode", string(defaults.PaletteStrategy))),
		PalettePreset:   formValueDefault(r, "palette", defaults.PalettePreset),
		Dither:          core.DitherMode(formValueDefault(r, "dither", string(defaults.Dither))),
		CropMode:        core.CropMode(formValueDefault(r, "crop", string(defaults.CropMode))),
		AlphaMode:       core.AlphaMode(formValueDefault(r, "alpha_mode", string(defaults.AlphaMode))),
		EmitDebug:       formBool(r, "debug"),
	}

	width, err := formIntDefault(r, "width", defaults.TargetWidth)
	if err != nil {
		return core.Config{}, fmt.Errorf("invalid width: %w", err)
	}
	height, err := formIntDefault(r, "height", defaults.TargetHeight)
	if err != nil {
		return core.Config{}, fmt.Errorf("invalid height: %w", err)
	}
	previewScale, err := formIntDefault(r, "preview_scale", defaults.PreviewScale)
	if err != nil {
		return core.Config{}, fmt.Errorf("invalid preview_scale: %w", err)
	}
	tileSize, err := formIntDefault(r, "tile_size", defaults.TileSize)
	if err != nil {
		return core.Config{}, fmt.Errorf("invalid tile_size: %w", err)
	}
	colorsPerTile, err := formIntDefault(r, "colors_per_tile", defaults.ColorsPerTile)
	if err != nil {
		return core.Config{}, fmt.Errorf("invalid colors_per_tile: %w", err)
	}
	maxPalettes, err := formIntDefault(r, "max_palettes", defaults.MaxPalettes)
	if err != nil {
		return core.Config{}, fmt.Errorf("invalid max_palettes: %w", err)
	}
	backgroundColor, err := formHexColorDefault(r, "bg_color", defaults.BackgroundColor)
	if err != nil {
		return core.Config{}, fmt.Errorf("invalid bg_color: %w", err)
	}

	cfg.TargetWidth = width
	cfg.TargetHeight = height
	cfg.PreviewScale = previewScale
	cfg.TileSize = tileSize
	cfg.ColorsPerTile = colorsPerTile
	cfg.MaxPalettes = maxPalettes
	cfg.BackgroundColor = backgroundColor

	return cfg, nil
}

func formValueDefault(r *http.Request, key, fallback string) string {
	value := strings.TrimSpace(r.FormValue(key))
	if value == "" {
		return fallback
	}
	return value
}

func formIntDefault(r *http.Request, key string, fallback int) (int, error) {
	raw := strings.TrimSpace(r.FormValue(key))
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

func formBool(r *http.Request, key string) bool {
	switch strings.ToLower(strings.TrimSpace(r.FormValue(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func formHexColorDefault(r *http.Request, key string, fallback color.NRGBA) (color.NRGBA, error) {
	raw := strings.TrimSpace(r.FormValue(key))
	if raw == "" {
		return fallback, nil
	}
	if strings.HasPrefix(raw, "#") {
		raw = raw[1:]
	}
	if len(raw) != 6 {
		return color.NRGBA{}, fmt.Errorf("want #RRGGBB")
	}

	parse := func(pair string) (uint8, error) {
		value, err := strconv.ParseUint(pair, 16, 8)
		if err != nil {
			return 0, err
		}
		return uint8(value), nil
	}

	rv, err := parse(raw[0:2])
	if err != nil {
		return color.NRGBA{}, err
	}
	gv, err := parse(raw[2:4])
	if err != nil {
		return color.NRGBA{}, err
	}
	bv, err := parse(raw[4:6])
	if err != nil {
		return color.NRGBA{}, err
	}

	return color.NRGBA{R: rv, G: gv, B: bv, A: 0xFF}, nil
}
