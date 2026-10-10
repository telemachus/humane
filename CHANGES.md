# humane version history

## v0.8.0

+ Require Go 1.26 or later.
+ Remove the files for go:build now that Go 1.24 is unsupported.
+ Modernize tests and benchmarks.
  E.g., `wg.Go`, `b.Loop`, `t.Context`, `slogtest.Run`, and `slices.Clone`.
+ Update the default time format to `"2006-01-02 15:04:05 MST"`.
  The old format used a 12-hour clock without AM or PM.
+ Follow slog and quote empty keys and values (`""=v`, `e=""`).
+ Bug fix: quote keys and values that contain control characters.
  E.g., prevent `\n` from breaking a record across lines.
+ Bug fix: follow slog and do not crash when a value's `MarshalText` or
  `Error` method panics.
  Print `<nil>` for a nil pointer; otherwise, fall back to `fmt`.
+ Bug fix: resolve and expand what `ReplaceAttr` returns.
  Previously a returned group or `LogValuer` was printed as a single value.
+ Bug fix: do not apply `ReplaceAttr` to a zero time, as documented.
+ Bug fix: quote the time based on the formatted output rather than the
  format string.

## v0.7.0

+ There are no changes to the public API.
+ Pre-format groups and attributes to avoid repeatedly formatting shared groups
  and attributes.
+ Use a pool for groups slice; put all pool-related code in internal/pooled.
+ Use slog's Record.Source method for Go 1.25+; continue using my version for Go
  1.24. (Once I drop support for 1.24, I can remove the go:build distinction.)
+ Expand test coverage and benchmarks.
+ Improve and tweak documentation. In particular, document in docs/README.md
  that humane supports only the latest two major versions of Go.

## v0.6.0

+ Fix a bug (found thanks to a user): display the level of the record not the
  handler (!).

## v0.5.0

+ Switch from `exp/slog` to `log/slog` and from `exp/slices` to `slices` now
  that Go 1.21 has been released.
+ Adjust the `NewHandler` function to match the latest `slog` API.
+ Use a pointer to `sync.Mutex` rather than `sync.Mutex`. See this discussion
  in the guide to writing `slog` handlers for why: <https://bit.ly/3s2KrOG>.
+ Add `testing/slogtest`.
+ Fix a bug (found thanks to `testing/slogtest`): move the test for
  `Attr.Empty` to catch all empty attrs.
+ Fix a bug (found thanks to `testing/slogtest`): make sure to call `Resolve`
  on all attribute values.
