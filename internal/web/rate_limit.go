package web

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type renderLimiter struct {
	window        time.Duration
	limit         int
	maxConcurrent int

	mu     sync.Mutex
	perIP  map[string]renderWindow
	active chan struct{}
}

type renderWindow struct {
	start time.Time
	count int
}

func newRenderLimiter(cfg ServerConfig) *renderLimiter {
	limiter := &renderLimiter{
		window:        cfg.RenderRateWindow,
		limit:         cfg.RenderRateLimit,
		maxConcurrent: cfg.MaxConcurrentRenders,
	}
	if cfg.RenderRateLimit > 0 {
		limiter.perIP = make(map[string]renderWindow)
	}
	if cfg.MaxConcurrentRenders > 0 {
		limiter.active = make(chan struct{}, cfg.MaxConcurrentRenders)
	}
	return limiter
}

func (l *renderLimiter) acquire(r *http.Request, now time.Time) (func(), int, string) {
	if l == nil {
		return func() {}, http.StatusOK, ""
	}
	if l.limit > 0 && !l.allowIP(clientIP(r), now) {
		return nil, http.StatusTooManyRequests, "render rate limit exceeded; retry soon"
	}
	if l.active == nil {
		return func() {}, http.StatusOK, ""
	}
	select {
	case l.active <- struct{}{}:
		return func() { <-l.active }, http.StatusOK, ""
	default:
		return nil, http.StatusTooManyRequests, "render queue full; retry soon"
	}
}

func (l *renderLimiter) allowIP(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.perIP[ip]
	if window.start.IsZero() || now.Sub(window.start) >= l.window {
		l.perIP[ip] = renderWindow{start: now, count: 1}
		return true
	}
	if window.count >= l.limit {
		return false
	}
	window.count++
	l.perIP[ip] = window
	return true
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if strings.TrimSpace(r.RemoteAddr) != "" {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return "unknown"
}
