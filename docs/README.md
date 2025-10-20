# `humane`: a human-friendly (but still largely structured) `slog.Handler`

`humane` provides a [slog][slog] handler for a human-friendly version of logfmt.
The idea for this format comes from [Brandur Leach's original post about
logfmt][logfmt]. See the section [Human logfmt and best practices][details] for
details. (To be very clear, Brandur Leach wrote that in 2016, and he has nothing
to do with this project. Any bad ideas are entirely my fault.)

[slog]: https://pkg.go.dev/log/slog
[logfmt]: https://brandur.org/logfmt
[details]: https://brandur.org/logfmt#human

## The Format

Handle formats a record in a human-friendly but largely structured way.

Typical output will look like the following.

```shell
 INFO | Request processed | sku=24A2 branch=manhattan time="2023-04-02T10:50.09 EDT"
ERROR | Connection failed | time=2024-01-23T17:14:03Z
```

Each line of output has three sections that are separated by " | ".

1. A level: DEBUG, INFO, WARN, or ERROR.
2. A message that the handler does not format in any way.
3. Zero or more key=value pairs. By default, a time attribute appears at the
   end. If `Options.AddSource` is true, then a source attribute appears before
   the time attribute. The time format can be changed via `Options.TimeFormat`,
   and both the time and source attributes can be modified or removed using
   `Options.ReplaceAttr`. Both time and source (if `AddSource` is true) appear
   at the top level and are not affected by `WithGroup()`.

The level and message sections appear as is without `key=value` structure or
quoting. The attributes appear as `key=value` pairs, with quotes added when
values contain whitespace or special characters.

The three sections of the log line are separated by " | ". This should make it
easy to parse out the sections of the message with (e.g.) `cut` or `awk`, but no
attempt is made to check for the pipe character anywhere else in the log line.
Thus, if pipes appear elsewhere, all bets are off. (This seems like a reasonable
trade-off to me since the format is meant for humans to scan rather than for
other programs to parse. If you want something fully structured, you should use
a different handler.)

## Supported Go Versions

`humane` follows [Go's release policy][release-policy]. It supports the latest
two major versions of Go. Currently, that means Go 1.25 and Go 1.24. When Go
1.26 is released, support for Go 1.24 will be dropped.

[release-policy]: https://go.dev/doc/devel/release#policy

## Installation

```shell
go get github.com/telemachus/humane
```

## Usage

```go
// Create and use a logger with default options.
logger := slog.New(humane.NewHandler(os.Stdout, nil))
logger.Info("My informative message", "foo", "bar", "bizz", "buzz")
logger.Error("Ooops", slog.Any("error", err))

// Output:
//  INFO | My informative message | foo=bar bizz=buzz time="2023-04-02T10:50.09 EDT"
// ERROR | Ooops | error="error message" time="2023-04-02T10:50.09 EDT"

// Use Options to change defaults. See the next section for more details.
opts := &humane.Options{
    Level: slog.LevelError,
    TimeFormat: time.RFC3339,
}
logger := slog.New(humane.NewHandler(os.Stderr, opts))
logger.Info("This message will not be written because the level is too low.")
```

## Options

+ `Level slog.Leveler`: Level defaults to slog.Info. You can use
  a [slog.Level](https://pkg.go.dev/log/slog#Level) to change the default. If
  you want something more complex, you can also implement
  a [slog.Leveler](https://pkg.go.dev/log/slog#Leveler).
+ `ReplaceAttr func(groups []string, a slog.Attr)`: As in slog itself, this
  function is applied to each attribute in a given Record during handling. This
  allows you to, e.g., edit attributes or omit them altogether. See [slog's
  documentation and tests for further examples](https://pkg.go.dev/log/slog).
  The `groups` slice is read-only; do not retain or modify it. Note that the
  ReplaceAttr function is **not** applied to the level or message, but it **is**
  applied to time and source attributes. These two attributes always appear at
  the top level; thus, they ignore groups. In order to make the time and source
  attributes easy to test for, they use constants defined by slog for their
  keys: `slog.TimeKey` and `slog.SourceKey`.
+ `TimeFormat string`: The time format defaults to `"2006-01-02T03:04.05 MST"`.
  You can use this option to set some other time format. (You can also tweak the
  time format via a ReplaceAttr function, but setting this option is easier for
  simple format changes.) The time attribute uses `slog.TimeKey` as its key.
+ `AddSource bool`: This option defaults to false. If you set it to true, then
  a source attribute will be added to each record before the time. The source
  attribute uses `slog.SourceKey` as its key and "/path/to/file:line" as its
  value. In practice, this will look like `source=/path/to/file:line`. If you
  want to edit this format (e.g., to only print the filename rather than its
  full path), you can use a ReplaceAttr function.

A common need (e.g., for testing) is to remove the time attribute entirely.
Here's a simple way to do that.

```go
func removeTime(_ []string, a slog.Attr) slog.Attr {
    if a.Key == slog.TimeKey {
        // Since slog does not show empty attributes, this removes the time.
        return slog.Attr{}
    }
    return a
}
opts := &humane.Options{ReplaceAttr: removeTime}
logger := slog.New(humane.NewHandler(os.Stdout, opts))
```

[issue]: https://github.com/telemachus/humane/issues

## Bugs and Limitations

Please [create an issue][issue] if you find any bugs.

One limitation concerns the source Attr. If you use the logger in a helper
function or a wrapper, then the source information may be wrong. See
[slog's documentation][sourceproblem] for a discussion and workaround.

[sourceproblem]: https://pkg.go.dev/log/slog#hdr-Wrapping_output_methods

## Acknowledgments

I'm using quite a lot of code from slog itself as well as from the [slog extras
repository][slogextras]. The [guide to writing `slog` handlers][guide] was also
very useful. Thanks to Jonathan Amsterdam for for all three of these. I've also
taken ideas and code from sources on [Go's wiki][wiki] as well as several blog
posts about slog. See below for a list of resources. (Note that some of the
resources are more or less out of date since slog and its API have changed over
time.)

+ [A Guide to Writing `slog` Handlers][guide]
+ [Proposal: Structured Logging][proposal]
+ [`slog`: Golang's official structured logging package][sobyte]
+ [Structured logging in Go][mrkaran]
+ [A Comprehensive Guide to Structured Logging in Go][betterstack]

[slogextras]: https://github.com/jba/slog
[guide]: https://github.com/golang/example/tree/master/slog-handler-guide
[wiki]: https://github.com/golang/go/wiki/Resources-for-slog
[proposal]: https://go.googlesource.com/proposal/+/master/design/56345-structured-logging.md
[sobyte]: https://www.sobyte.net/post/2022-10/go-slog/
[mrkaran]: https://mrkaran.dev/posts/structured-logging-in-go-with-slog/
[betterstack]: https://betterstack.com/community/guides/logging/logging-in-go/
