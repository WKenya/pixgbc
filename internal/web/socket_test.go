package web

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/WKenya/pixgbc/internal/ioimg"
	"github.com/WKenya/pixgbc/internal/review"
)

func TestRenderSocketPublishesProgressAndDone(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(blockingEngine{}, ioimg.DefaultLimits(), store, ServerConfig{})
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	socketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?client_id=test-client"
	conn, _, err := websocket.Dial(ctx, socketURL, nil)
	if err != nil {
		t.Fatalf("websocket.Dial() error = %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	var ready RenderSocketEvent
	if err := wsjson.Read(ctx, conn, &ready); err != nil {
		t.Fatalf("wsjson.Read(ready) error = %v", err)
	}
	if ready.Type != "ready" {
		t.Fatalf("ready.Type = %q, want ready", ready.Type)
	}

	requestBody, contentType := renderMultipartBody(t, map[string]string{
		"client_id": "test-client",
	})
	responseCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/render", bytes.NewReader(requestBody))
		if err != nil {
			errCh <- err
			return
		}
		req.Header.Set("Content-Type", contentType)
		resp, err := server.Client().Do(req)
		if err != nil {
			errCh <- err
			return
		}
		responseCh <- resp
	}()

	sawProgress := false
	sawDone := false
	for !sawDone {
		var event RenderSocketEvent
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			t.Fatalf("wsjson.Read(event) error = %v", err)
		}
		if event.Type == "progress" {
			sawProgress = true
		}
		if event.Type == "done" {
			sawDone = true
			if event.Result == nil || event.Result.PreviewURL == "" {
				t.Fatalf("done event missing result payload: %#v", event)
			}
		}
	}

	select {
	case err := <-errCh:
		t.Fatalf("render request error = %v", err)
	case resp := <-responseCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("POST /api/render status = %d", resp.StatusCode)
		}
		var payload RenderResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("json.Decode(render response) error = %v", err)
		}
		if payload.PreviewURL == "" {
			t.Fatalf("payload missing preview url: %#v", payload)
		}
	case <-ctx.Done():
		t.Fatal("render request never completed")
	}

	if !sawProgress {
		t.Fatal("did not receive progress event")
	}
}

func renderMultipartBody(t *testing.T, fields map[string]string) ([]byte, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "socket.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField(%s) error = %v", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	return body.Bytes(), writer.FormDataContentType()
}
