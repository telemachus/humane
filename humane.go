package humane

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/telemachus/humane/internal/buffer"
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
	mu          *sync.Mutex
	level       slog.Leveler
	groups      []string
	attrs       string
	timeFormat  string
	replaceAttr func(groups []string, a slog.Attr) slog.Attr
	addSource   bool
}

// Options are options for Humane's [log/slog.Handler].
//
// Level reports the minimum level to log. Humane uses [log/slog.LevelInfo] as
// its default level. In order to set a different level, use one of the
// built-in choices for [log/slog.Level] or implement a [log/slog.Leveler].
//
// ReplaceAttr is a user-defined function that receives each non-group Attr
// before it is logged. By default, ReplaceAttr is nil, and no changes are made
// to Attrs. Note: Humane's handler does not apply ReplaceAttr to the level or
// message Attrs because the handler already formats these items in a specific
// way. However, Humane does apply ReplaceAttr to the time Attr (unless it's
// zero) and to the source Attr if AddSource is true.
//
// TimeFormat defaults to "2006-01-02T03:04.05 MST". Set a format option to
// customize the presentation of the time. (See [time.Time.Format] for details
// about the format string.)
//
// AddSource defaults to false. If AddSource is true, the handler adds to each
// log event an Attr with [log/slog.SourceKey] as the key and "file:line" as
// the value.
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
	}
	h.groups = make([]string, 0, 10)
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

// Handle formats a given record in a human-friendly but still largely
// structured way.
func (h *handler) Handle(_ context.Context, r slog.Record) error {
	buf := buffer.New()
	defer buf.Free()
	h.appendLevel(buf, r.Level)
	buf.WriteByte(' ')
	buf.WriteString(r.Message)
	buf.WriteString(" |")
	if h.attrs != "" {
		buf.WriteString(h.attrs)
	}
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(buf, a)
		return true
	})
	if h.addSource && r.PC != 0 {
		sourceAttr := h.newSourceAttr(r.PC)
		h.appendAttr(buf, sourceAttr)
	}
	timeAttr := slog.Time(slog.TimeKey, r.Time)
	if h.replaceAttr != nil {
		timeAttr = h.replaceAttr(nil, timeAttr)
	}
	if !r.Time.IsZero() && !timeAttr.Equal(slog.Attr{}) {
		appendKey(buf, nil, timeAttr.Key)
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
	buf := buffer.New()
	defer buf.Free()
	for _, a := range attrs {
		h2.appendAttr(buf, a)
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
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *handler) clone() *handler {
	return &handler{
		w:           h.w,
		mu:          h.mu,
		level:       h.level,
		groups:      slices.Clip(h.groups),
		attrs:       h.attrs,
		timeFormat:  h.timeFormat,
		replaceAttr: h.replaceAttr,
		addSource:   h.addSource,
	}
}

func (h *handler) appendLevel(buf *buffer.Buffer, level slog.Level) {
	if lVal, ok := levelValues[level.Level()]; ok {
		buf.WriteString(lVal)
		return
	}
	buf.WriteByte(' ')
	buf.WriteString(level.Level().String())
	buf.WriteString(" |")
}

func (h *handler) appendAttr(buf *buffer.Buffer, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return
		}
		if a.Key != "" {
			h.groups = append(h.groups, a.Key)
		}
		for _, a := range attrs {
			h.appendAttr(buf, a)
		}
		if a.Key != "" {
			h.groups = h.groups[:len(h.groups)-1]
		}
		return
	}
	if h.replaceAttr != nil {
		a = h.replaceAttr(h.groups, a)
	}
	if !a.Equal(slog.Attr{}) {
		appendKey(buf, h.groups, a.Key)
		h.appendVal(buf, a.Value)
	}
}

func appendKey(buf *buffer.Buffer, groups []string, key string) {
	buf.WriteByte(' ')
	if len(groups) > 0 {
		key = strings.Join(groups, ".") + "." + key
	}
	if needsQuoting(key) {
		*buf = strconv.AppendQuote(*buf, key)
	} else {
		buf.WriteString(key)
	}
	buf.WriteByte('=')
}

func (h *handler) appendVal(buf *buffer.Buffer, val slog.Value) {
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
		// If fmt contains any quote characters, this won't
		// properly quote it. But alternative versions run far slower.
		// If the user must have a time with quotes, they can use
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
				// TODO: should this append an error?
				return
			}
			appendString(buf, string(data))
			return
		}
		appendString(buf, fmt.Sprint(val.Any()))
	}
}

func appendString(buf *buffer.Buffer, s string) {
	if needsQuoting(s) {
		*buf = strconv.AppendQuote(*buf, s)
	} else {
		buf.WriteString(s)
	}
}

func (h *handler) newSourceAttr(pc uintptr) slog.Attr {
	source := frame(pc)
	info := fmt.Sprintf("%s:%d", source.File, source.Line)
	return slog.String(slog.SourceKey, info)
}

func frame(pc uintptr) runtime.Frame {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	return f
}

func needsQuoting(s string) bool {
	for i := 0; i < len(s); {
		b := s[i]
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
