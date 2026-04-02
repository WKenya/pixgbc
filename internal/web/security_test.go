package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WKenya/pixgbc/internal/ioimg"
	"github.com/WKenya/pixgbc/internal/render"
	"github.com/WKenya/pixgbc/internal/review"
)

func TestSecurityHeadersPresent(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	headers := recorder.Header()
	if got := headers.Get("Content-Security-Policy"); got != defaultContentSecurityPolicy {
		t.Fatalf("Content-Security-Policy = %q, want %q", got, defaultContentSecurityPolicy)
	}
	for key, want := range map[string]string{
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Permissions-Policy":           "camera=(), geolocation=(), microphone=()",
		"Referrer-Policy":              "no-referrer",
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
	} {
		if got := headers.Get(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestSecurityMiddlewareBlocksDotfileAndTraversalPaths(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{})

	for _, target := range []string{
		"/.aws/credentials",
		"/.ssh/config",
		"/%2eenv",
		"/api/renders/%2e%2e/artifacts/final.png",
		"/api/%2fetc%2fpasswd",
	} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want 404", target, recorder.Code)
		}
	}
}

func TestSecurityMiddlewareAllowsNormalPaths(t *testing.T) {
	store, err := review.NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	handler := NewServerWithStore(render.NewEngine(), ioimg.DefaultLimits(), store, ServerConfig{})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
