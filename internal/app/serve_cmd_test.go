package app

import (
	"bytes"
	"context"
	"testing"
)

func TestIsLocalListen(t *testing.T) {
	for _, value := range []string{"127.0.0.1:8080", "localhost:8080", "[::1]:8080", ":8080"} {
		if !isLocalListen(value) {
			t.Fatalf("isLocalListen(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"0.0.0.0:8080", "192.168.1.2:8080", "[2001:db8::1]:8080"} {
		if isLocalListen(value) {
			t.Fatalf("isLocalListen(%q) = true, want false", value)
		}
	}
}

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		raw  string
		want int64
	}{
		{"10MB", 10 << 20},
		{"512KB", 512 << 10},
		{"42", 42},
	}
	for _, test := range tests {
		got, err := parseByteSize(test.raw)
		if err != nil {
			t.Fatalf("parseByteSize(%q) error = %v", test.raw, err)
		}
		if got != test.want {
			t.Fatalf("parseByteSize(%q) = %d, want %d", test.raw, got, test.want)
		}
	}

	if _, err := parseByteSize("0MB"); err == nil {
		t.Fatal("parseByteSize(0MB) error = nil, want error")
	}
}

func TestRequiresAccessToken(t *testing.T) {
	tests := []struct {
		name            string
		addr            string
		token           string
		allowOpenAccess bool
		want            bool
	}{
		{name: "localhost no token", addr: "127.0.0.1:8080", want: false},
		{name: "remote token", addr: "0.0.0.0:8080", token: "secret", want: false},
		{name: "remote open access", addr: "0.0.0.0:8080", allowOpenAccess: true, want: false},
		{name: "remote requires token", addr: "0.0.0.0:8080", want: true},
	}

	for _, test := range tests {
		if got := requiresAccessToken(test.addr, test.token, test.allowOpenAccess); got != test.want {
			t.Fatalf("%s: requiresAccessToken(%q, %q, %t) = %t, want %t", test.name, test.addr, test.token, test.allowOpenAccess, got, test.want)
		}
	}
}

func TestServeRateFlagsMustBeNonNegative(t *testing.T) {
	app := New(&bytes.Buffer{}, &bytes.Buffer{})

	for _, args := range [][]string{
		{"serve", "--request-rate-per-minute", "-1"},
		{"serve", "--probe-rate-per-minute", "-1"},
		{"serve", "--render-rate-per-minute", "-1"},
	} {
		if code := app.Run(context.Background(), args); code != 2 {
			t.Fatalf("Run(%v) = %d, want 2", args, code)
		}
	}
}
