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

func BenchmarkBasic_Slog(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		logger.LogAttrs(
			context.Background(),
			slog.LevelInfo,
			"message",
			slogAttrs...,
		)
	}
}

func BenchmarkBasic_Humane(b *testing.B) {
	logger := slog.New(humane.NewHandler(io.Discard, nil))
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		logger.LogAttrs(
			context.Background(),
			slog.LevelInfo,
			"message",
			slogAttrs...,
		)
	}
}

func BenchmarkDeeplyNestedGroups_Slog(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		logger.Info("message",
			slog.Group("l1",
				slog.String("a", "1"),
				slog.Group("l2",
					slog.String("b", "2"),
					slog.Group("l3",
						slog.String("c", "3"),
						slog.Group("l4",
							slog.String("d", "4"),
							slog.Group("l5",
								slog.String("e", "5"),
							),
						),
					),
				),
			),
		)
	}
}

func BenchmarkDeeplyNestedGroups_Humane(b *testing.B) {
	logger := slog.New(humane.NewHandler(io.Discard, nil))

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		logger.Info("message",
			slog.Group("l1",
				slog.String("a", "1"),
				slog.Group("l2",
					slog.String("b", "2"),
					slog.Group("l3",
						slog.String("c", "3"),
						slog.Group("l4",
							slog.String("d", "4"),
							slog.Group("l5",
								slog.String("e", "5"),
							),
						),
					),
				),
			),
		)
	}
}

func BenchmarkWithGroupChaining_Slog(b *testing.B) {
	handler := slog.NewTextHandler(io.Discard, nil).
		WithGroup("g1").
		WithGroup("g2").
		WithGroup("g3").
		WithGroup("g4").
		WithGroup("g5")
	logger := slog.New(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		logger.Info("message", "key", "value")
	}
}

func BenchmarkWithGroupChaining_Humane(b *testing.B) {
	handler := humane.NewHandler(io.Discard, nil).
		WithGroup("g1").
		WithGroup("g2").
		WithGroup("g3").
		WithGroup("g4").
		WithGroup("g5")
	logger := slog.New(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		logger.Info("message", "key", "value")
	}
}

func BenchmarkWithAttrsChaining_Slog(b *testing.B) {
	handler := slog.NewTextHandler(io.Discard, nil).
		WithAttrs([]slog.Attr{slog.String("a", "1")}).
		WithAttrs([]slog.Attr{slog.String("b", "2")}).
		WithAttrs([]slog.Attr{slog.String("c", "3")}).
		WithAttrs([]slog.Attr{slog.String("d", "4")}).
		WithAttrs([]slog.Attr{slog.String("e", "5")})
	logger := slog.New(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		logger.Info("message", "key", "value")
	}
}

func BenchmarkWithAttrsChaining_Humane(b *testing.B) {
	handler := humane.NewHandler(io.Discard, nil).
		WithAttrs([]slog.Attr{slog.String("a", "1")}).
		WithAttrs([]slog.Attr{slog.String("b", "2")}).
		WithAttrs([]slog.Attr{slog.String("c", "3")}).
		WithAttrs([]slog.Attr{slog.String("d", "4")}).
		WithAttrs([]slog.Attr{slog.String("e", "5")})
	logger := slog.New(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		logger.Info("message", "key", "value")
	}
}

func BenchmarkMixedGroupsAndAttrs_Slog(b *testing.B) {
	handler := slog.NewTextHandler(io.Discard, nil).
		WithGroup("service").
		WithAttrs([]slog.Attr{slog.String("name", "api")}).
		WithGroup("request").
		WithAttrs([]slog.Attr{slog.String("id", "req-123")}).
		WithGroup("db")
	logger := slog.New(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		logger.Info("query", "table", "users", "rows", 42)
	}
}

func BenchmarkMixedGroupsAndAttrs_Humane(b *testing.B) {
	handler := humane.NewHandler(io.Discard, nil).
		WithGroup("service").
		WithAttrs([]slog.Attr{slog.String("name", "api")}).
		WithGroup("request").
		WithAttrs([]slog.Attr{slog.String("id", "req-123")}).
		WithGroup("db")
	logger := slog.New(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		logger.Info("query", "table", "users", "rows", 42)
	}
}

func BenchmarkWithReplaceAttr_Slog(b *testing.B) {
	replaceAttr := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == "sensitive" {
			return slog.String("sensitive", "REDACTED")
		}
		return a
	}

	opts := &slog.HandlerOptions{ReplaceAttr: replaceAttr}
	logger := slog.New(slog.NewTextHandler(io.Discard, opts))

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		logger.LogAttrs(
			context.Background(),
			slog.LevelInfo,
			"message",
			slogAttrs...,
		)
	}
}

func BenchmarkWithReplaceAttr_Humane(b *testing.B) {
	replaceAttr := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == "sensitive" {
			return slog.String("sensitive", "REDACTED")
		}
		return a
	}

	opts := &humane.Options{ReplaceAttr: replaceAttr}
	logger := slog.New(humane.NewHandler(io.Discard, opts))

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		logger.LogAttrs(
			context.Background(),
			slog.LevelInfo,
			"message",
			slogAttrs...,
		)
	}
}

func BenchmarkParallelLogging_Slog(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.LogAttrs(
				context.Background(),
				slog.LevelInfo,
				"message",
				slogAttrs...,
			)
		}
	})
}

func BenchmarkParallelLogging_Humane(b *testing.B) {
	logger := slog.New(humane.NewHandler(io.Discard, nil))

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.LogAttrs(
				context.Background(),
				slog.LevelInfo,
				"message",
				slogAttrs...,
			)
		}
	})
}

func BenchmarkParallelWithGroups_Slog(b *testing.B) {
	handler := slog.NewTextHandler(io.Discard, nil).
		WithGroup("service").
		WithGroup("request")
	logger := slog.New(handler)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("message", "key", "value", "count", 42)
		}
	})
}

func BenchmarkParallelWithGroups_Humane(b *testing.B) {
	handler := humane.NewHandler(io.Discard, nil).
		WithGroup("service").
		WithGroup("request")
	logger := slog.New(handler)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("message", "key", "value", "count", 42)
		}
	})
}

func BenchmarkParallelDeeplyNested_Slog(b *testing.B) {
	handler := slog.NewTextHandler(io.Discard, nil).
		WithGroup("g1").
		WithGroup("g2").
		WithGroup("g3").
		WithGroup("g4").
		WithGroup("g5")
	logger := slog.New(handler)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("message",
				slog.Group("inner",
					slog.String("key", "value"),
					slog.Int("count", 42),
				),
			)
		}
	})
}

func BenchmarkParallelDeeplyNested_Humane(b *testing.B) {
	handler := humane.NewHandler(io.Discard, nil).
		WithGroup("g1").
		WithGroup("g2").
		WithGroup("g3").
		WithGroup("g4").
		WithGroup("g5")
	logger := slog.New(handler)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("message",
				slog.Group("inner",
					slog.String("key", "value"),
					slog.Int("count", 42),
				),
			)
		}
	})
}
