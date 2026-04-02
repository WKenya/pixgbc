package web

import (
	"net/http"
	"net/url"
	"strings"
)

const defaultContentSecurityPolicy = "default-src 'self'; base-uri 'self'; connect-src 'self' ws: wss:; form-action 'self'; frame-ancestors 'none'; img-src 'self' data: blob:; object-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'"

func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		headers.Set("Content-Security-Policy", defaultContentSecurityPolicy)
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Cross-Origin-Resource-Policy", "same-origin")
		headers.Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func suspiciousRequestPath(r *http.Request) bool {
	rawPath := r.URL.EscapedPath()
	if rawPath == "" {
		rawPath = r.URL.Path
	}
	lowerRaw := strings.ToLower(rawPath)
	if strings.Contains(lowerRaw, "%2f") || strings.Contains(lowerRaw, "%5c") {
		return true
	}

	decodedPath, err := url.PathUnescape(rawPath)
	if err != nil {
		return true
	}
	if strings.Contains(decodedPath, "\\") {
		return true
	}

	segments := strings.Split(decodedPath, "/")
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		if segment == "." || segment == ".." || strings.HasPrefix(segment, ".") {
			return true
		}
	}

	return false
}
