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
	renderLimit   int
	requestLimit  int
	probeLimit    int
	maxConcurrent int

	mu           sync.Mutex
	renderPerIP  map[string]renderWindow
	requestPerIP map[string]renderWindow
	probePerIP   map[string]renderWindow
	active       chan struct{}
}

type renderWindow struct {
	start time.Time
	count int
}

func newRenderLimiter(cfg ServerConfig) *renderLimiter {
	limiter := &renderLimiter{
		window:        cfg.RenderRateWindow,
		renderLimit:   cfg.RenderRateLimit,
		requestLimit:  cfg.RequestRateLimit,
		probeLimit:    cfg.ProbeRateLimit,
		maxConcurrent: cfg.MaxConcurrentRenders,
	}
	if cfg.RenderRateLimit > 0 {
		limiter.renderPerIP = make(map[string]renderWindow)
	}
	if cfg.RequestRateLimit > 0 {
		limiter.requestPerIP = make(map[string]renderWindow)
	}
	if cfg.ProbeRateLimit > 0 {
		limiter.probePerIP = make(map[string]renderWindow)
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
	if l.renderLimit > 0 && !l.allowRenderIP(clientIP(r), now) {
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

func (l *renderLimiter) allowRequestIP(ip string, now time.Time) bool {
	if l == nil || l.requestLimit <= 0 {
		return true
	}
	return l.allowWindow(l.requestPerIP, ip, now, l.requestLimit)
}

func (l *renderLimiter) allowProbeIP(ip string, now time.Time) bool {
	if l == nil || l.probeLimit <= 0 {
		return true
	}
	return l.allowWindow(l.probePerIP, ip, now, l.probeLimit)
}

func (l *renderLimiter) allowRenderIP(ip string, now time.Time) bool {
	if l == nil || l.renderLimit <= 0 {
		return true
	}
	return l.allowWindow(l.renderPerIP, ip, now, l.renderLimit)
}

func (l *renderLimiter) allowWindow(windows map[string]renderWindow, ip string, now time.Time, limit int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	window := windows[ip]
	if window.start.IsZero() || now.Sub(window.start) >= l.window {
		windows[ip] = renderWindow{start: now, count: 1}
		return true
	}
	if window.count >= limit {
		return false
	}
	window.count++
	windows[ip] = window
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
