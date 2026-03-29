package web

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WKenya/pixgbc/internal/ioimg"
	"github.com/WKenya/pixgbc/internal/render"
	"github.com/WKenya/pixgbc/internal/review"
)

func TestRenderAndFetchReview(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "smoke.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.WriteField("palette", "gbc-olive"); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	bodyBytes := append([]byte(nil), body.Bytes()...)

	request := httptest.NewRequest(http.MethodPost, "/api/render", bytes.NewReader(bodyBytes))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("POST /api/render status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var renderResponse RenderResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &renderResponse); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if renderResponse.ID == "" {
		t.Fatal("render response id empty")
	}
	if renderResponse.SourceURL == "" || renderResponse.CompareURL == "" || renderResponse.PreviewURL == "" || renderResponse.RecordURL == "" || renderResponse.ReviewURL == "" {
		t.Fatalf("render response missing urls: %#v", renderResponse)
	}

	recordRequest := httptest.NewRequest(http.MethodGet, renderResponse.RecordURL, nil)
	recordRecorder := httptest.NewRecorder()
	handler.ServeHTTP(recordRecorder, recordRequest)

	if recordRecorder.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, body = %s", renderResponse.RecordURL, recordRecorder.Code, recordRecorder.Body.String())
	}

	var record review.ReviewRecord
	if err := json.Unmarshal(recordRecorder.Body.Bytes(), &record); err != nil {
		t.Fatalf("json.Unmarshal(record) error = %v", err)
	}
	if record.ID != renderResponse.ID {
		t.Fatalf("record.ID = %q, want %q", record.ID, renderResponse.ID)
	}

	previewRequest := httptest.NewRequest(http.MethodGet, renderResponse.PreviewURL, nil)
	previewRecorder := httptest.NewRecorder()
	handler.ServeHTTP(previewRecorder, previewRequest)

	if previewRecorder.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d", renderResponse.PreviewURL, previewRecorder.Code)
	}
	if got := previewRecorder.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("preview content-type = %q, want image/png", got)
	}

	compareRequest := httptest.NewRequest(http.MethodGet, renderResponse.CompareURL, nil)
	compareRecorder := httptest.NewRecorder()
	handler.ServeHTTP(compareRecorder, compareRequest)
	if compareRecorder.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d", renderResponse.CompareURL, compareRecorder.Code)
	}
	if got := compareRecorder.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("compare content-type = %q, want image/png", got)
	}
}

func TestRenderCGBBGIncludesDebugArtifact(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "strict.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makeWidePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.WriteField("palette", "gbc-olive"); err != nil {
		t.Fatalf("WriteField(palette) error = %v", err)
	}
	if err := writer.WriteField("mode", "cgb-bg"); err != nil {
		t.Fatalf("WriteField(mode) error = %v", err)
	}
	if err := writer.WriteField("debug", "1"); err != nil {
		t.Fatalf("WriteField(debug) error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	bodyBytes := append([]byte(nil), body.Bytes()...)

	request := httptest.NewRequest(http.MethodPost, "/api/render", bytes.NewReader(bodyBytes))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("POST /api/render status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response RenderResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.DebugURL == "" {
		t.Fatalf("DebugURL empty: %#v", response)
	}

	debugRequest := httptest.NewRequest(http.MethodGet, response.DebugURL, nil)
	debugRecorder := httptest.NewRecorder()
	handler.ServeHTTP(debugRecorder, debugRequest)

	if debugRecorder.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d", response.DebugURL, debugRecorder.Code)
	}
}

func TestRenderPersistsFormConfig(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "config.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	fields := map[string]string{
		"mode":            "relaxed",
		"palette_mode":    "extract",
		"palette":         "lcd-cool",
		"width":           "96",
		"height":          "80",
		"crop":            "fit",
		"dither":          "none",
		"brightness":      "0.1",
		"contrast":        "-0.2",
		"gamma":           "1.25",
		"alpha_mode":      "reserve-color0",
		"bg_color":        "#112233",
		"preview_scale":   "3",
		"tile_size":       "12",
		"colors_per_tile": "3",
		"max_palettes":    "6",
		"debug":           "1",
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField(%s) error = %v", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	bodyBytes := append([]byte(nil), body.Bytes()...)

	request := httptest.NewRequest(http.MethodPost, "/api/render", bytes.NewReader(bodyBytes))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("POST /api/render status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response RenderResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	recordRequest := httptest.NewRequest(http.MethodGet, response.RecordURL, nil)
	recordRecorder := httptest.NewRecorder()
	handler.ServeHTTP(recordRecorder, recordRequest)

	if recordRecorder.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, body = %s", response.RecordURL, recordRecorder.Code, recordRecorder.Body.String())
	}

	var record review.ReviewRecord
	if err := json.Unmarshal(recordRecorder.Body.Bytes(), &record); err != nil {
		t.Fatalf("json.Unmarshal(record) error = %v", err)
	}
	if record.Config.TargetWidth != 96 || record.Config.TargetHeight != 80 {
		t.Fatalf("record.Config size = %dx%d, want 96x80", record.Config.TargetWidth, record.Config.TargetHeight)
	}
	if record.Config.CropMode != "fit" {
		t.Fatalf("record.Config.CropMode = %q, want fit", record.Config.CropMode)
	}
	if record.Config.Dither != "none" {
		t.Fatalf("record.Config.Dither = %q, want none", record.Config.Dither)
	}
	if record.Config.Brightness != 0.1 || record.Config.Contrast != -0.2 || record.Config.Gamma != 1.25 {
		t.Fatalf("record.Config tone = %v/%v/%v, want 0.1/-0.2/1.25", record.Config.Brightness, record.Config.Contrast, record.Config.Gamma)
	}
	if record.Config.PaletteStrategy != "extract" {
		t.Fatalf("record.Config.PaletteStrategy = %q, want extract", record.Config.PaletteStrategy)
	}
	if record.Config.PreviewScale != 3 {
		t.Fatalf("record.Config.PreviewScale = %d, want 3", record.Config.PreviewScale)
	}
	if record.Config.AlphaMode != "reserve-color0" {
		t.Fatalf("record.Config.AlphaMode = %q, want reserve-color0", record.Config.AlphaMode)
	}
	if record.Config.BackgroundColor.R != 0x11 || record.Config.BackgroundColor.G != 0x22 || record.Config.BackgroundColor.B != 0x33 {
		t.Fatalf("record.Config.BackgroundColor = %#v, want #112233", record.Config.BackgroundColor)
	}
	if record.Config.TileSize != 12 {
		t.Fatalf("record.Config.TileSize = %d, want 12", record.Config.TileSize)
	}
	if record.Config.ColorsPerTile != 3 {
		t.Fatalf("record.Config.ColorsPerTile = %d, want 3", record.Config.ColorsPerTile)
	}
	if record.Config.MaxPalettes != 6 {
		t.Fatalf("record.Config.MaxPalettes = %d, want 6", record.Config.MaxPalettes)
	}
	if !record.Config.EmitDebug {
		t.Fatal("record.Config.EmitDebug = false, want true")
	}
}

func TestListRendersReturnsNewestFirst(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}
	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{})

	for _, name := range []string{"one.png", "two.png"} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", name)
		if err != nil {
			t.Fatalf("CreateFormFile() error = %v", err)
		}
		if _, err := part.Write(makePNG(t)); err != nil {
			t.Fatalf("part.Write() error = %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("writer.Close() error = %v", err)
		}

		request := httptest.NewRequest(http.MethodPost, "/api/render", bytes.NewReader(body.Bytes()))
		request.Header.Set("Content-Type", writer.FormDataContentType())
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("POST /api/render status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/api/renders?limit=10", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/renders status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var items []RenderHistoryItem
	if err := json.Unmarshal(recorder.Body.Bytes(), &items); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID == items[1].ID {
		t.Fatalf("items ids repeated: %#v", items)
	}
	if items[0].SourceURL == "" || items[0].CompareURL == "" || items[0].PreviewURL == "" || items[0].ReviewURL == "" {
		t.Fatalf("history item missing urls: %#v", items[0])
	}
}

func TestRenderRejectsInvalidBGColor(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "bad-bg.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.WriteField("bg_color", "#12"); err != nil {
		t.Fatalf("WriteField(bg_color) error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	bodyBytes := append([]byte(nil), body.Bytes()...)

	request := httptest.NewRequest(http.MethodPost, "/api/render", bytes.NewReader(bodyBytes))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/render status = %d, want 400", recorder.Code)
	}
}

func TestRenderRejectsInvalidGamma(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "bad-gamma.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.WriteField("gamma", "oops"); err != nil {
		t.Fatalf("WriteField(gamma) error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	bodyBytes := append([]byte(nil), body.Bytes()...)

	request := httptest.NewRequest(http.MethodPost, "/api/render", bytes.NewReader(bodyBytes))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/render status = %d, want 400", recorder.Code)
	}
}

func TestRenderRequiresTokenWhenConfigured(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{
		Token: "secret-token",
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "auth.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	bodyBytes := append([]byte(nil), body.Bytes()...)

	request := httptest.NewRequest(http.MethodPost, "/api/render", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("POST /api/render status = %d, want 401", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/render?token=secret-token", bytes.NewReader(bodyBytes))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("POST /api/render with token status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRenderRejectsOversizeBodyByConfiguredLimit(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	limits := ioimg.DefaultLimits()
	limits.MaxFileBytes = 32
	handler := NewServerWithStore(render.NewEngine(), limits, store, ServerConfig{})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "too-big.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makeWidePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/render", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/render status = %d, want 400, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRenderResponseURLsPreserveTokenQuery(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{
		Token: "secret-token",
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "auth-links.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/render?token=secret-token", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("POST /api/render status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response RenderResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	for _, value := range []string{response.ReviewURL, response.RecordURL, response.SourceURL, response.PreviewURL, response.FinalURL, response.CompareURL} {
		if !strings.Contains(value, "token=secret-token") {
			t.Fatalf("url %q missing propagated token", value)
		}
	}
}

func TestReviewPagePreservesTokenLinks(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{
		Token: "secret-token",
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "review-links.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/render?token=secret-token", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("POST /api/render status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response RenderResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	reviewRequest := httptest.NewRequest(http.MethodGet, response.ReviewURL, nil)
	reviewRecorder := httptest.NewRecorder()
	handler.ServeHTTP(reviewRecorder, reviewRequest)

	if reviewRecorder.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, body = %s", response.ReviewURL, reviewRecorder.Code, reviewRecorder.Body.String())
	}
	if !strings.Contains(reviewRecorder.Body.String(), "token=secret-token") {
		t.Fatalf("review page missing tokenized links: %s", reviewRecorder.Body.String())
	}
}

func makePNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0x20, G: 0x40, B: 0x60, A: 0xFF})
	img.SetNRGBA(1, 0, color.NRGBA{R: 0x80, G: 0xA0, B: 0xC0, A: 0xFF})
	img.SetNRGBA(0, 1, color.NRGBA{R: 0xD0, G: 0xC0, B: 0x80, A: 0xFF})
	img.SetNRGBA(1, 1, color.NRGBA{R: 0xF0, G: 0xE0, B: 0xD0, A: 0xFF})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	return buf.Bytes()
}

func makeWidePNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 16, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(0x10 + x*2), G: uint8(0x20 + y*2), B: 0x18, A: 0xFF})
		}
		for x := 8; x < 16; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(0x80 + x), G: uint8(0x30 + y*3), B: 0x20, A: 0xFF})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	return buf.Bytes()
}
