package humane_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/telemachus/humane"
)

var (
	errTester  = errors.New("random error")
	timeTester = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
)

var slogAttrs = []slog.Attr{
	slog.Any("error", errTester),
	slog.Bool("bool", true),
	slog.Duration("duration", 5*time.Second),
	slog.Float64("float64", 3.14),
	slog.Group("group", slog.Int("d", 4), slog.Duration("a", 5*time.Second)),
	slog.Int("int", 3),
	slog.Int64("int64", 4),
	slog.String("string", "random string"),
	slog.Time("time", timeTester),
	slog.Uint64("uint64", 0),
}

func BenchmarkSlog(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.LogAttrs(
			context.Background(),
			slog.LevelInfo,
			"message",
			slogAttrs...,
		)
	}
}

func BenchmarkHumane(b *testing.B) {
	logger := slog.New(humane.NewHandler(io.Discard, nil))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.LogAttrs(
			context.Background(),
			slog.LevelInfo,
			"message",
			slogAttrs...,
		)
	}
}
