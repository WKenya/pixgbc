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

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store)

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

	request := httptest.NewRequest(http.MethodPost, "/api/render", &body)
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
	if renderResponse.PreviewURL == "" || renderResponse.RecordURL == "" || renderResponse.ReviewURL == "" {
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
