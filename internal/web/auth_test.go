package web

import (
	"bytes"
	"encoding/json"
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

func TestSessionLoginUnlocksProtectedRoutes(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{
		Token:      "secret-token",
		SessionTTL: 6 * time.Hour,
	})

	statusRequest := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	statusRecorder := httptest.NewRecorder()
	handler.ServeHTTP(statusRecorder, statusRequest)
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("GET /api/session status = %d, want 200", statusRecorder.Code)
	}

	var status SessionStatusResponse
	if err := json.Unmarshal(statusRecorder.Body.Bytes(), &status); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if !status.AuthRequired || status.Authenticated {
		t.Fatalf("status = %#v, want protected+logged-out", status)
	}

	palettesRequest := httptest.NewRequest(http.MethodGet, "/api/palettes", nil)
	palettesRecorder := httptest.NewRecorder()
	handler.ServeHTTP(palettesRecorder, palettesRequest)
	if palettesRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/palettes status = %d, want 401", palettesRecorder.Code)
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/session/login", strings.NewReader(`{"token":"secret-token"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	handler.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("POST /api/session/login status = %d, body = %s", loginRecorder.Code, loginRecorder.Body.String())
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != sessionCookieName {
		t.Fatalf("login cookies = %#v, want %q", cookies, sessionCookieName)
	}

	palettesRequest = httptest.NewRequest(http.MethodGet, "/api/palettes", nil)
	palettesRequest.AddCookie(cookies[0])
	palettesRecorder = httptest.NewRecorder()
	handler.ServeHTTP(palettesRecorder, palettesRequest)
	if palettesRecorder.Code != http.StatusOK {
		t.Fatalf("GET /api/palettes with cookie status = %d, body = %s", palettesRecorder.Code, palettesRecorder.Body.String())
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "session.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(makePNG(t)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	renderRequest := httptest.NewRequest(http.MethodPost, "/api/render", bytes.NewReader(body.Bytes()))
	renderRequest.Header.Set("Content-Type", writer.FormDataContentType())
	renderRequest.AddCookie(cookies[0])
	renderRecorder := httptest.NewRecorder()
	handler.ServeHTTP(renderRecorder, renderRequest)
	if renderRecorder.Code != http.StatusOK {
		t.Fatalf("POST /api/render with cookie status = %d, body = %s", renderRecorder.Code, renderRecorder.Body.String())
	}

	var response RenderResponse
	if err := json.Unmarshal(renderRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal(render) error = %v", err)
	}
	for _, value := range []string{response.ReviewURL, response.RecordURL, response.PreviewURL, response.FinalURL} {
		if strings.Contains(value, "token=") {
			t.Fatalf("cookie-auth response leaked query token in %q", value)
		}
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/session/logout", nil)
	logoutRequest.AddCookie(cookies[0])
	logoutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(logoutRecorder, logoutRequest)
	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("POST /api/session/logout status = %d", logoutRecorder.Code)
	}

	palettesRequest = httptest.NewRequest(http.MethodGet, "/api/palettes", nil)
	palettesRequest.AddCookie(logoutRecorder.Result().Cookies()[0])
	palettesRecorder = httptest.NewRecorder()
	handler.ServeHTTP(palettesRecorder, palettesRequest)
	if palettesRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/palettes after logout status = %d, want 401", palettesRecorder.Code)
	}
}

func TestSessionLoginRejectsBadToken(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{
		Token: "secret-token",
	})

	request := httptest.NewRequest(http.MethodPost, "/api/session/login", strings.NewReader(`{"token":"wrong"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("POST /api/session/login status = %d, want 401", recorder.Code)
	}
}
