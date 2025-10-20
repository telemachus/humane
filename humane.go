package humane

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/telemachus/humane/internal/pooled"
)

var (
	defaultLevel      = slog.LevelInfo
	defaultTimeFormat = "2006-01-02T03:04.05 MST"
	levelValues       = map[slog.Level]string{
		slog.LevelDebug: "DEBUG |",
		slog.LevelInfo:  " INFO |",
		slog.LevelWarn:  " WARN |",
		slog.LevelError: "ERROR |",
	}
)

type handler struct {
	w           io.Writer
	level       slog.Leveler
	mu          *sync.Mutex
	replaceAttr func(groups []string, a slog.Attr) slog.Attr
	attrs       string
	groupPrefix string
	timeFormat  string
	groups      []string
	addSource   bool
}

// Options are options for Humane's [log/slog.Handler].
//
// Level sets the minimum level to log. Humane uses [log/slog.LevelInfo] as its
// default. In order to set a different level, use one of the built-in choices
// for [log/slog.Level] or implement a [log/slog.Leveler].
//
// ReplaceAttr is a user-defined function that receives each non-group Attr
// before it is logged. The first argument is a slice of groups that contain
// the Attr. This slice is read-only; do not retain or modify it. By default,
// ReplaceAttr is nil, and no changes are made to Attrs. Note: Humane's handler
// does not apply ReplaceAttr to the level or message Attrs because the handler
// already formats these items in a specific way. However, Humane does apply
// ReplaceAttr to the time Attr (unless it's zero) and to the source Attr if
// AddSource is true.
//
// TimeFormat defaults to "2006-01-02T03:04.05 MST". Set a format option to
// customize the presentation of the time. (See [time.Time.Format] for details
// about the format string.)
//
// AddSource defaults to false. If AddSource is true, the handler adds to each
// log event an Attr with a key of [log/slog.SourceKey] and a value of
// "/path/to/file:line".
type Options struct {
	Level       slog.Leveler
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
	TimeFormat  string
	AddSource   bool
}

// NewHandler returns a [log/slog.Handler] using the receiver's options.
// Default options are used if opts is nil.
func NewHandler(w io.Writer, opts *Options) slog.Handler {
	if opts == nil {
		opts = &Options{}
	}
	h := &handler{
		w:           w,
		mu:          &sync.Mutex{},
		level:       opts.Level,
		timeFormat:  opts.TimeFormat,
		replaceAttr: opts.ReplaceAttr,
		addSource:   opts.AddSource,
		groups:      make([]string, 0, 10),
	}
	if opts.Level == nil {
		h.level = defaultLevel
	}
	if h.timeFormat == "" {
		h.timeFormat = defaultTimeFormat
	}
	return h
}

// Users should not call the following methods directly on a handler. Instead,
// users should create a logger and call methods on the logger. The logger will
// create a record and invoke the handler's methods.

// Enabled indicates whether the receiver logs at the given level.
func (h *handler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

// Handle formats a record in a human-friendly but largely structured way.
//
// Typical lines will look like the following:
//
//	 INFO | Request processed | sku=24A2 branch=manhattan time="2023-04-02T10:50.09 EDT"
//	ERROR | Connection failed | time=2024-01-23T17:14:03Z
//
// More abstractly each line has three sections that are separated by " | ".
//
//  1. A level: DEBUG, INFO, WARN, or ERROR.
//  2. A message that the handler does not format in any way.
//  3. Zero or more key=value pairs. By default, a time attribute appears at
//     the end. The format of the time can be changed via Options.TimeFormat,
//     and the time attribute can be removed using Options.ReplaceAttr. Both
//     time and source (if AddSource is true) appear at the top level and are
//     not affected by WithGroup().
func (h *handler) Handle(_ context.Context, r slog.Record) error {
	buf := pooled.NewBuffer()
	defer buf.Free()
	// If ReplaceAttr is nil, groups can be nil.
	var groups *pooled.StringSlice
	hasReplaceAttr := h.replaceAttr != nil
	if hasReplaceAttr {
		groups = pooled.NewStringSlice()
		defer groups.Free()
		groups.Append(h.groups...)
	}

	appendLevel(buf, r.Level)
	buf.WriteByte(' ')
	buf.WriteString(r.Message)
	buf.WriteString(" |")
	if h.attrs != "" {
		buf.WriteString(h.attrs)
	}
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(buf, a, h.groupPrefix, groups)
		return true
	})
	if h.addSource {
		src := source(r)
		if src != nil && (src.File != "" || src.Line != 0) {
			sourceBuf := pooled.NewBuffer()
			defer sourceBuf.Free()
			sourceBuf.WriteString(src.File)
			sourceBuf.WriteByte(':')
			sourceBuf.WriteString(strconv.Itoa(src.Line))
			sourceAttr := slog.String(slog.SourceKey, sourceBuf.String())
			h.appendAttr(buf, sourceAttr, "", nil)
		}
	}
	timeAttr := slog.Time(slog.TimeKey, r.Time)
	if hasReplaceAttr {
		// Pass nil since we format time outside of groups.
		timeAttr = h.replaceAttr(nil, timeAttr)
	}
	if !r.Time.IsZero() && !timeAttr.Equal(slog.Attr{}) {
		appendKey(buf, "", timeAttr.Key)
		h.appendVal(buf, timeAttr.Value)
	}
	buf.WriteByte('\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(*buf)
	return err
}

// WithAttrs returns a new [log/slog.Handler] that has the receiver's
// attributes plus attrs.
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := h.clone()
	buf := pooled.NewBuffer()
	defer buf.Free()

	var groups *pooled.StringSlice
	if h.replaceAttr != nil {
		groups = pooled.NewStringSlice()
		defer groups.Free()
		groups.Append(h.groups...)
	}

	for _, a := range attrs {
		h2.appendAttr(buf, a, h.groupPrefix, groups)
	}
	h2.attrs += string(*buf)
	return h2
}

// WithGroup returns a new [log/slog.Handler] with name appended to the
// receiver's groups.
func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	if h.groupPrefix == "" {
		h2.groupPrefix = name
	} else {
		h2.groupPrefix = h.groupPrefix + "." + name
	}
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *handler) clone() *handler {
	return &handler{
		w:           h.w,
		mu:          h.mu,
		level:       h.level,
		groupPrefix: h.groupPrefix,
		// Force a new backing array as soon as h.groups grows.
		groups:      slices.Clip(h.groups),
		attrs:       h.attrs,
		timeFormat:  h.timeFormat,
		replaceAttr: h.replaceAttr,
		addSource:   h.addSource,
	}
}

func appendLevel(buf *pooled.Buffer, level slog.Level) {
	if lVal, ok := levelValues[level.Level()]; ok {
		buf.WriteString(lVal)
		return
	}
	buf.WriteByte(' ')
	buf.WriteString(level.Level().String())
	buf.WriteString(" |")
}

//nolint:cyclop // This function simply *is* complex.
func (h *handler) appendAttr(buf *pooled.Buffer, a slog.Attr, groupPrefix string, groups *pooled.StringSlice) {
	a.Value = a.Value.Resolve()
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return
		}
		var newGroupPrefix string

		if a.Key != "" {
			if groupPrefix == "" {
				newGroupPrefix = a.Key
			} else {
				newGroupPrefix = groupPrefix + "." + a.Key
			}
			if groups != nil {
				groups.Append(a.Key)
			}
		} else {
			newGroupPrefix = groupPrefix
		}

		for _, a := range attrs {
			h.appendAttr(buf, a, newGroupPrefix, groups)
		}

		if a.Key != "" && groups != nil {
			*groups = (*groups)[:groups.Len()-1]
		}
		return
	}

	var groupsSlice []string
	if groups != nil {
		groupsSlice = *groups
	}
	if h.replaceAttr != nil {
		a = h.replaceAttr(groupsSlice, a)
	}
	if !a.Equal(slog.Attr{}) {
		appendKey(buf, groupPrefix, a.Key)
		h.appendVal(buf, a.Value)
	}
}

func appendKey(buf *pooled.Buffer, groups, key string) {
	buf.WriteByte(' ')
	var fullKey string
	if groups != "" {
		fullKey = groups + "." + key
	} else {
		fullKey = key
	}
	if needsQuoting(fullKey) {
		*buf = strconv.AppendQuote(*buf, fullKey)
	} else {
		buf.WriteString(fullKey)
	}
	buf.WriteByte('=')
}

//nolint:cyclop // This function simply *is* complex.
func (h *handler) appendVal(buf *pooled.Buffer, val slog.Value) {
	switch val.Kind() {
	case slog.KindString:
		appendString(buf, val.String())
	case slog.KindInt64:
		*buf = strconv.AppendInt(*buf, val.Int64(), 10)
	case slog.KindUint64:
		*buf = strconv.AppendUint(*buf, val.Uint64(), 10)
	case slog.KindFloat64:
		*buf = strconv.AppendFloat(*buf, val.Float64(), 'g', -1, 64)
	case slog.KindBool:
		*buf = strconv.AppendBool(*buf, val.Bool())
	case slog.KindDuration:
		appendString(buf, val.Duration().String())
	case slog.KindTime:
		// This is crude: if timeFormat needs quoting, we simply quote
		// the entire formatted time string.
		//
		// If the user must have a time with quotes, they should use
		// ReplaceAttr to change the Kind to slog.String.
		quoteTime := needsQuoting(h.timeFormat)
		if quoteTime {
			buf.WriteByte('"')
		}
		*buf = val.Time().AppendFormat(*buf, h.timeFormat)
		if quoteTime {
			buf.WriteByte('"')
		}
	case slog.KindAny, slog.KindGroup, slog.KindLogValuer:
		if tm, ok := val.Any().(encoding.TextMarshaler); ok {
			data, err := tm.MarshalText()
			if err != nil {
				appendString(buf, fmt.Sprintf("!ERROR:%v", err))
				return
			}
			appendString(buf, string(data))
			return
		}
		appendString(buf, fmt.Sprint(val.Any()))
	}
}

func appendString(buf *pooled.Buffer, s string) {
	if needsQuoting(s) {
		*buf = strconv.AppendQuote(*buf, s)
	} else {
		buf.WriteString(s)
	}
}

func needsQuoting(s string) bool {
	for i := 0; i < len(s); {
		b := s[i]
		// Handle ASCII characters quickly.
		if b < utf8.RuneSelf {
			if unsafe[b] {
				return true
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return true
		}
		i += size
	}
	return false
}

// Adapted from log/slog/json_handler.go which copied the original from
// encoding/json/tables.go.
//
// unsafe holds the value true if the ASCII character requires a logfmt key or
// value to be quoted.
//
// All values are safe except for ' ', '"', and '='. Note that a map is far slower.
var unsafe = [utf8.RuneSelf]bool{
	' ': true,
	'"': true,
	'=': true,
}
