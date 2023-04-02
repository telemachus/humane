// Package humane provides a slog.Handler for a human-friendly version of logfmt.
// The idea for this format comes from Brandur Leach in his original post about
// logfmt. See https://brandur.org/logfmt#human for the inspiration.
//
// Examples:
//
// 1. Get a slog logger using humane's handler with default options:
//
//	logger := slog.New(humane.NewHandler(os.Stdout))
//	logger.Info("Message", "foo", "bar", "bizz", "buzz")
//
// 2. Get a slog logger using humane's handler with customized options:
//
//	func trimSource(_ []string, a slog.Attr) slog.Attr {
//		if a.Key == slog.SourceKey {
//			return slog.String(
//				slog.SourceKey,
//				filepath.Base(a.Value.String()),
//			)
//		}
//		return a
//	}
//
//	func main() {
//		ho := humane.Options{
//			Level:       slog.LevelError,
//			ReplaceAttr: trimSource,
//			TimeFormat:  time.Kitchen,
//			AddSource:   true,
//		}
//		logger := slog.New(ho.NewHandler(os.Stdout))
//		// ... later
//		logger.Error("Message", "error", err, "response", respStatus)
//	}
//
// [this section of the post]: https://brandur.org/logfmt#human
package humane
