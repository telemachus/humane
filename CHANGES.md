# humane version history

# v0.5.0

+ Switch from `exp/slog` to `log/slog` and from `exp/slices` to `slices` now
  that Go 1.21 has been released.
+ Adjust the `NewHandler` function to match the latest `slog` API.
+ Use a pointer to `sync.Mutex` rather than `sync.Mutex`. See this discussion
  in the guide to writing `slog` handlers for why: https://bit.ly/3s2KrOG.
+ Add `testing/slogtest`.
+ Fix a bug (found thanks to `testing/slogtest`): move the test for
  `Attr.Empty` to catch all empty attrs.
+ Fix a bug (found thanks to `testing/slogtest`): make sure to call `Resolve`
  on all attribute values.
