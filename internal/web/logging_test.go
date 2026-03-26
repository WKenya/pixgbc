package web

import (
	"bytes"
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

func TestServerLogsRenderLifecycle(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	var logs bytes.Buffer
	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{
		LogOutput: &logs,
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "logged.png")
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

	bodyText := logs.String()
	for _, snippet := range []string{
		"render start mode=relaxed",
		"render done id=",
		"http POST /api/render status=200",
	} {
		if !strings.Contains(bodyText, snippet) {
			t.Fatalf("logs missing %q: %s", snippet, bodyText)
		}
	}
}
