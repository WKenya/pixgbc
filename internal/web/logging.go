package web

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (s *Server) logf(format string, args ...any) {
	if s.cfg.LogOutput == nil {
		return
	}
	_, _ = fmt.Fprintf(s.cfg.LogOutput, format+"\n", args...)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	if s.cfg.LogOutput == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		next.ServeHTTP(recorder, r)
		s.logf("http %s %s status=%d duration=%s", r.Method, r.URL.Path, recorder.status, time.Since(start).Round(time.Millisecond))
	})
}
