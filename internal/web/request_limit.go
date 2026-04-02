package web

import (
	"net/http"
	"time"
)

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	if s.limiter == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		now := time.Now()

		if suspiciousRequestPath(r) {
			if !s.limiter.allowProbeIP(ip, now) {
				http.Error(w, "probe rate limit exceeded; retry soon", http.StatusTooManyRequests)
				return
			}
			http.NotFound(w, r)
			return
		}

		if !s.limiter.allowRequestIP(ip, now) {
			http.Error(w, "request rate limit exceeded; retry soon", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
