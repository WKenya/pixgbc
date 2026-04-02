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
	engine  core.Engine
	limits  ioimg.Limits
	store   review.Store
	cfg     ServerConfig
	limiter *renderLimiter
	sockets *socketHub
}

type ServerConfig struct {
	Token                string
	LogOutput            io.Writer
	SessionTTL           time.Duration
	RenderRateLimit      int
	RequestRateLimit     int
	ProbeRateLimit       int
	RenderRateWindow     time.Duration
	MaxConcurrentRenders int
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
	cfg = normalizeServerConfig(cfg)
	server := &Server{
		engine:  engine,
		limits:  limits,
		store:   store,
		cfg:     cfg,
		limiter: newRenderLimiter(cfg),
		sockets: newSocketHub(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.handleHealth)
	mux.HandleFunc("GET /ws", server.handleSocket)
	mux.HandleFunc("GET /api/session", server.handleSessionStatus)
	mux.HandleFunc("POST /api/session/login", server.handleSessionLogin)
	mux.HandleFunc("POST /api/session/logout", server.handleSessionLogout)
	mux.HandleFunc("GET /api/palettes", server.handlePalettes)
	mux.HandleFunc("GET /api/renders", server.handleListRenders)
	mux.HandleFunc("POST /api/render", server.handleRender)
	mux.HandleFunc("GET /api/renders/{id}", server.handleGetRecord)
	mux.HandleFunc("GET /api/renders/{id}/artifacts/{name}", server.handleGetArtifact)
	mux.HandleFunc("GET /renders/{id}", server.handleReviewPage)
	mux.Handle("/", server.staticHandler())
	return server.loggingMiddleware(server.securityHeadersMiddleware(server.rateLimitMiddleware(mux)))
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
	release, status, message := s.limiter.acquire(r, time.Now())
	if release == nil {
		s.logf("render reject status=%d reason=%q ip=%s", status, message, clientIP(r))
		http.Error(w, message, status)
		return
	}
	defer release()

	r.Body = http.MaxBytesReader(w, r.Body, s.limits.MaxFileBytes)
	if err := r.ParseMultipartForm(s.limits.MaxFileBytes); err != nil {
		http.Error(w, fmt.Sprintf("parse upload: %v", err), http.StatusBadRequest)
		return
	}
	clientID := strings.TrimSpace(r.FormValue("client_id"))
	s.publishProgress(clientID, "upload", 6, "upload received")

	file, _, err := r.FormFile("file")
	if err != nil {
		s.publishError(clientID, "missing form file field 'file'")
		http.Error(w, "missing form file field 'file'", http.StatusBadRequest)
		return
	}
	defer file.Close()

	inputBytes, err := io.ReadAll(file)
	if err != nil {
		s.publishError(clientID, fmt.Sprintf("read upload: %v", err))
		http.Error(w, fmt.Sprintf("read upload: %v", err), http.StatusBadRequest)
		return
	}
	s.publishProgress(clientID, "decode", 16, "decoding source image")

	decoded, err := ioimg.DecodeImage(bytes.NewReader(inputBytes), s.limits)
	if err != nil {
		s.publishError(clientID, err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.publishProgress(clientID, "config", 24, "reading render controls")
	cfg, err := parseRenderConfig(r)
	if err != nil {
		s.publishError(clientID, err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.logf(
		"render start mode=%s palette_mode=%s palette=%s source=%dx%d bytes=%d",
		cfg.Mode,
		cfg.PaletteStrategy,
		cfg.PalettePreset,
		decoded.Meta.Width,
		decoded.Meta.Height,
		len(inputBytes),
	)

	resultCtx := core.WithProgressReporter(r.Context(), func(update core.ProgressUpdate) {
		s.publishProgress(clientID, update.Stage, update.Percent, update.Message)
	})
	result, err := s.engine.Run(resultCtx, source.NewSingleImage(decoded.Image, decoded.Meta), cfg)
	if err != nil {
		s.publishError(clientID, err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.publishProgress(clientID, "save", 96, "saving review bundle")
	record, err := review.SaveResult(r.Context(), s.store, inputBytes, cfg, result)
	if err != nil {
		s.publishError(clientID, fmt.Sprintf("save review: %v", err))
		http.Error(w, fmt.Sprintf("save review: %v", err), http.StatusInternalServerError)
		return
	}

	response := RenderResponse{
		ID:         record.ID,
		ReviewURL:  s.reviewURL(record.ID, s.requestTokenQuery(r)),
		RecordURL:  s.recordURL(record.ID, s.requestTokenQuery(r)),
		SourceURL:  s.artifactURL(record.ID, record.Artifacts.SourcePNG, s.requestTokenQuery(r)),
		PreviewURL: s.artifactURL(record.ID, record.Artifacts.PreviewPNG, s.requestTokenQuery(r)),
		FinalURL:   s.artifactURL(record.ID, record.Artifacts.FinalPNG, s.requestTokenQuery(r)),
		CompareURL: s.artifactURL(record.ID, record.Artifacts.ComparePNG, s.requestTokenQuery(r)),
	}
	if record.Artifacts.DebugPNG != "" {
		response.DebugURL = s.artifactURL(record.ID, record.Artifacts.DebugPNG, s.requestTokenQuery(r))
	}
	s.logf(
		"render done id=%s mode=%s output=%dx%d review=%s",
		record.ID,
		record.Mode,
		record.OutputWidth,
		record.OutputHeight,
		response.ReviewURL,
	)
	s.publishProgress(clientID, "done", 100, "render complete")
	s.publishResult(clientID, response)
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleListRenders(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	limit, err := formIntDefault(r, "limit", 20)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid limit: %v", err), http.StatusBadRequest)
		return
	}
	records, err := s.store.List(r.Context(), limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("list renders: %v", err), http.StatusInternalServerError)
		return
	}

	token := s.requestTokenQuery(r)
	out := make([]RenderHistoryItem, 0, len(records))
	for _, record := range records {
		item := RenderHistoryItem{
			ID:         record.ID,
			CreatedAt:  record.CreatedAt.Format(time.RFC3339),
			Mode:       record.Mode,
			Width:      record.OutputWidth,
			Height:     record.OutputHeight,
			SourceURL:  s.artifactURL(record.ID, record.Artifacts.SourcePNG, token),
			PreviewURL: s.artifactURL(record.ID, record.Artifacts.PreviewPNG, token),
			ReviewURL:  s.reviewURL(record.ID, token),
			RecordURL:  s.recordURL(record.ID, token),
			FinalURL:   s.artifactURL(record.ID, record.Artifacts.FinalPNG, token),
			CompareURL: s.artifactURL(record.ID, record.Artifacts.ComparePNG, token),
		}
		if record.Artifacts.DebugPNG != "" {
			item.DebugURL = s.artifactURL(record.ID, record.Artifacts.DebugPNG, token)
		}
		out = append(out, item)
	}

	writeJSON(w, http.StatusOK, out)
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
		s.artifactURL(record.ID, record.Artifacts.SourcePNG, s.requestTokenQuery(r)),
		s.artifactURL(record.ID, record.Artifacts.PreviewPNG, s.requestTokenQuery(r)),
		s.artifactURL(record.ID, record.Artifacts.FinalPNG, s.requestTokenQuery(r)),
		s.artifactURL(record.ID, record.Artifacts.ComparePNG, s.requestTokenQuery(r)),
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

func (s *Server) publishProgress(clientID, stage string, percent int, message string) {
	if clientID == "" {
		return
	}
	s.sockets.send(clientID, RenderSocketEvent{
		Type:    "progress",
		Stage:   stage,
		Percent: percent,
		Message: message,
	})
}

func (s *Server) publishResult(clientID string, result RenderResponse) {
	if clientID == "" {
		return
	}
	s.sockets.send(clientID, RenderSocketEvent{
		Type:    "done",
		Stage:   "done",
		Percent: 100,
		Message: "render complete",
		Result:  &result,
	})
}

func (s *Server) publishError(clientID, message string) {
	if clientID == "" {
		return
	}
	s.sockets.send(clientID, RenderSocketEvent{
		Type:    "error",
		Percent: 100,
		Message: message,
	})
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
	brightness, err := formFloatDefault(r, "brightness", 0)
	if err != nil {
		return core.Config{}, fmt.Errorf("invalid brightness: %w", err)
	}
	contrast, err := formFloatDefault(r, "contrast", 0)
	if err != nil {
		return core.Config{}, fmt.Errorf("invalid contrast: %w", err)
	}
	gamma, err := formFloatDefault(r, "gamma", defaults.Gamma)
	if err != nil {
		return core.Config{}, fmt.Errorf("invalid gamma: %w", err)
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
	cfg.Brightness = brightness
	cfg.Contrast = contrast
	cfg.Gamma = gamma
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

func formFloatDefault(r *http.Request, key string, fallback float64) (float64, error) {
	raw := strings.TrimSpace(r.FormValue(key))
	if raw == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(raw, 64)
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
	return core.ParseHexColor(raw)
}

func normalizeServerConfig(cfg ServerConfig) ServerConfig {
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 12 * time.Hour
	}
	if cfg.RenderRateWindow <= 0 {
		cfg.RenderRateWindow = time.Minute
	}
	if cfg.RenderRateLimit < 0 {
		cfg.RenderRateLimit = 0
	}
	if cfg.RequestRateLimit < 0 {
		cfg.RequestRateLimit = 0
	}
	if cfg.ProbeRateLimit < 0 {
		cfg.ProbeRateLimit = 0
	}
	if cfg.MaxConcurrentRenders < 0 {
		cfg.MaxConcurrentRenders = 0
	}
	return cfg
}
