package web

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/ioimg"
	"github.com/WKenya/pixgbc/internal/review"
)

func TestRenderRateLimitByIP(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(blockingEngine{}, ioimg.DefaultLimits(), store, ServerConfig{
		RenderRateLimit:  1,
		RenderRateWindow: time.Minute,
	})

	request := makeRenderRequest(t, "/api/render", "1.2.3.4:1234")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("first POST /api/render status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	request = makeRenderRequest(t, "/api/render", "1.2.3.4:9999")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("second POST /api/render status = %d, want 429", recorder.Code)
	}

	request = makeRenderRequest(t, "/api/render", "5.6.7.8:7777")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("third POST /api/render other ip status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRenderConcurrencyLimit(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	block := make(chan struct{})
	handler := NewServerWithStore(blockingEngine{wait: block}, ioimg.DefaultLimits(), store, ServerConfig{
		MaxConcurrentRenders: 1,
	})

	firstDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, makeRenderRequest(t, "/api/render", "1.2.3.4:1111"))
		firstDone <- recorder
	}()

	time.Sleep(50 * time.Millisecond)

	secondRecorder := httptest.NewRecorder()
	handler.ServeHTTP(secondRecorder, makeRenderRequest(t, "/api/render", "1.2.3.4:2222"))
	if secondRecorder.Code != http.StatusTooManyRequests {
		t.Fatalf("second POST /api/render status = %d, want 429", secondRecorder.Code)
	}

	close(block)

	select {
	case firstRecorder := <-firstDone:
		if firstRecorder.Code != http.StatusOK {
			t.Fatalf("first POST /api/render status = %d, body = %s", firstRecorder.Code, firstRecorder.Body.String())
		}
	case <-time.After(time.Second):
		t.Fatal("first render never completed")
	}
}

type blockingEngine struct {
	wait chan struct{}
}

func (e blockingEngine) Run(ctx context.Context, src core.Source, _ core.Config) (*core.Result, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-e.waitOrClosed():
	}

	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xFF})
	return &core.Result{
		FinalImage:      img,
		PreviewImage:    img,
		NormalizedImage: img,
		SourceMeta:      src.Meta(),
	}, nil
}

func (e blockingEngine) waitOrClosed() <-chan struct{} {
	if e.wait != nil {
		return e.wait
	}
	ch := make(chan struct{})
	close(ch)
	return ch
}

func makeRenderRequest(t *testing.T, target, remoteAddr string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "limit.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.RemoteAddr = remoteAddr
	return request
}
