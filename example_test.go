package humane_test

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/telemachus/humane"
)

func ExampleInfo() {
	opts := &humane.Options{ReplaceAttr: removeTime, Level: slog.LevelDebug}
	logger := slog.New(humane.NewHandler(os.Stdout, opts))
	logger.Debug("Foo")
	logger.Info("Bar")
	logger.Warn("Fizz")
	logger.Error("Buzz")
	logger.Error("Error", slog.Any("error", fmt.Errorf("xxxx")))
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
