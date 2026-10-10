package humane_test

import (
	"errors"
	"log/slog"
	"os"

	"github.com/telemachus/humane"
)

func ExampleNewHandler() {
	opts := &humane.Options{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}

			return a
		},
	}
	logger := slog.New(humane.NewHandler(os.Stdout, opts))
	logger.Debug("Foo")
	logger.Info("Bar")
	logger.Warn("Fizz")
	logger.Error("Buzz")
	logger.Error("Error", slog.Any("error", errors.New("xxxx")))
	logger.Warn("Warn", "foo", "bar")
	logger.Info("Info", "fizz", "buzz")
	logger.Debug("Debug", "status", "hello, world")
	// Output:
	// DEBUG | Foo |
	//  INFO | Bar |
	//  WARN | Fizz |
	// ERROR | Buzz |
	// ERROR | Error | error=xxxx
	//  WARN | Warn | foo=bar
	//  INFO | Info | fizz=buzz
	// DEBUG | Debug | status="hello, world"
}
