package web

import (
	"fmt"
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
