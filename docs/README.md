# `humane`: a human-friendly (but still largely structured) `slog.Handler`

`humane` provides a slog.Handler for a human-friendly version of logfmt.  The
idea for this format comes from [Brandur Leach's original post about
logfmt][logfmt].  See the section [Human logfmt and best practices][details]
for details.  (To be very clear, Brandur Leach wrote that section in 2016, and
he has nothing to do with this project.  Any bad ideas are entirely my fault,
not his.)

[logfmt]: https://brandur.org/logfmt
[details]: https://brandur.org/logfmt#human

## Warning

[slog][slogdiscussion] has been accepted for Go 1.21, but it is still under
development.  This handler is new and also still being tweaked.  Let me know
if you have trouble with it, but be aware that API may change.

[slogdiscussion]: https://github.com/golang/go/issues/56345

## The Format

Briefly, the format is as follows.

```
LEVEL | Message text | [foo=bar...] time="2023-04-02T10:50.09 EDT"
```

The level and message Attrs appear as is without `key=value` structure or
quoting.  Then the rest of the Attrs appear as `key=value` pairs.  A time Attr
will be added by default to the third section.  (See below for how to change
the format of this Attr or omit it entirely.)  The three sections of the log
message are separated by a pipe character (`|`).  The pipes should make it easy
to parse out the sections of the message with (e.g.) `cut` or `awk`, but no
attempt is made to check for that character anywhere else in the log.  Thus, if
pipes appear elsewhere, all bets are off.  (This seems like a reasonable
trade-off to me since the format is meant for humans to scan rather than for
other programs to parse.  If you want something fully structured, you should
probably use JSON or another format.)

## Installation

```
go get github.com/telemachus/humane
```

## Usage

```go
// Create a logger with default options.  See below for more on available
// options.
logger := slog.New(humane.NewHandler(os.Stdout))
logger.Info("My informative message", "foo", "bar", "bizz", "buzz")
logger.Error("Ooops", slog.Any("error", err))
// Output:
// INFO | My informative message | foo=bar bizz=buzz time=2023-04-02T10:50.09 EDT
// ERROR | Ooops | error="error message" time=2023-04-02T10:50.09 EDT

// You can also set options.  Again, see the next section for more details.
ho := humane.Options{
    Level: slog.LevelError,
    TimeFormat: time.RFC3339
}
logger := slog.New(ho.NewHandler(os.Stderr))
logger.Info("This message will not be written")
```

## Options

+ `Level slog.Leveler`: Level defaults to slog.Info.  You can use
  a [slog.Level](https://pkg.go.dev/golang.org/x/exp/slog#Level) to change
  the default.  If you want something more complex, you can also implement
  a [slog.Leveler](https://pkg.go.dev/golang.org/x/exp/slog#Leveler).
+ `ReplaceAttr func(groups []string, a slog.Attr)`: As in slog itself, this
  function is applied to each Attr in a given Record during handling.  This
  allows you to, e.g., omit or edit Attrs in creative ways.  See [slog's
  documentation and tests for further examples](slog).  Note that the
  ReplaceAttr function is **not** applied to the level or message Attrs since
  they receive specific formatting by this handler.  (However, I am open to
  reconsidering that.  Please open an [issue][issue] to discuss it.)  In order
  to make the time and source Attrs easier to test for, they use constants
  defined by slog for their keys: `slog.TimeKey` and `slog.SourceKey`.
+ `TimeFormat string`: The time format defaults to "2006-01-02T03:04.05 MST".
  You can use this option to set some other time format.  (You can also tweak
  the time Attr via a ReplaceAttr function, but this is easier for simple
  format changes.)  The time Attr uses `slog.TimeKey` as its key value by
  default.
+ `AddSource bool`: This option defaults to false.  If you set it to true,
  then an Attr containing `source=/path/to/source:line` will be added to each
  record.  If source Attr is present, it uses `slog.SourceKey` as its default
  key value.

A common need (e.g., for testing) is to remove the time Attr altogether.
Here's a simple way to do that.

```go
func removeTime(_ []string, a slog.Attr) slog.Attr {
    if a.Key == slog.TimeKey {
        return slog.Attr{}
    }
    return a
}
ho := humane.Options{ReplaceAttr: removeTime}
logger := slog.New(ho.NewHandler(os.Stdout))
```

[slog]: https://pkg.go.dev/golang.org/x/exp/slog
[issue]: https://github.com/telemachus/humane/issues

## Bugs and Limitations

I'm not aware of any bugs yet, but I'm sure there in here.  Please [let me
know][issue] if you find any.

One limitation concerns the source Attr.  If you use the logger in a helper
function or a wrapper, then the source information will likely be wrong.  See
[slog's documentation][sourceproblem] for a discussion and workaround.  There
is also [an open issue][sourceissue] that proposes more support in slog for
writing helper functions.  (There's another [open issue][pcissue] that
proposes other ways to give users of slog more help with the source Attr.)

[sourceproblem]: https://pkg.go.dev/golang.org/x/exp/slog#hdr-Wrapping_output_methods
[sourceissue]: https://github.com/golang/go/issues/59145
[pcissue]: https://github.com/golang/go/issues/59280


## Acknowledgements

I'm using quite a lot of code from slog itself as well as from the [slog
extras repository][slogextras].  Thanks to Jonathan Amsterdam for both.  I've
also taken ideas and code from sources on [Go's wiki][wiki] as well as several
blog posts about slog.  See below for a list of resources. (Note that some of
the resources are more or less out of date since slog and its API have changed
over time.)

+ [Proposal: Structured Logging][proposal]
+ [`slog`: Golang's official structured logging package][sobyte]
+ [Structured logging in Go][mrkaran]
+ [A Comprehensive Guide to Structured Logging in Go][betterstack]

[slogextras]: https://github.com/jba/slog
[wiki]: https://github.com/golang/go/wiki/Resources-for-slog
[proposal]: https://go.googlesource.com/proposal/+/master/design/56345-structured-logging.md
[sobyte]: https://www.sobyte.net/post/2022-10/go-slog/
[mrkaran]: https://mrkaran.dev/posts/structured-logging-in-go-with-slog/
[betterstack]: https://betterstack.com/community/guides/logging/logging-in-go/
