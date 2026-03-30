package core

import "context"

type ProgressUpdate struct {
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
	Message string `json:"message"`
}

type progressContextKey struct{}

func WithProgressReporter(ctx context.Context, fn func(ProgressUpdate)) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, progressContextKey{}, fn)
}

func ReportProgress(ctx context.Context, stage string, percent int, message string) {
	fn, ok := ctx.Value(progressContextKey{}).(func(ProgressUpdate))
	if !ok || fn == nil {
		return
	}
	fn(ProgressUpdate{
		Stage:   stage,
		Percent: percent,
		Message: message,
	})
}
