package app

import "testing"

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
