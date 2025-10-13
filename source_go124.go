//go:build !go1.25

package humane

import (
	"log/slog"
	"runtime"
)

func source(r slog.Record) *slog.Source {
	if r.PC == 0 {
		return nil
	}
	fs := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := fs.Next()
	return &slog.Source{
		Function: f.Function,
		File:     f.File,
		Line:     f.Line,
	}
}
