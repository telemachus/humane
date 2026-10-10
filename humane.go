package humane

import (
	"cmp"
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

const (
	defaultLevel      = slog.LevelInfo
	defaultTimeFormat = "2006-01-02 15:04:05 MST"
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
// TimeFormat defaults to "2006-01-02 15:04:05 MST". Set a format option to
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

// NewHandler returns a [log/slog.Handler] using the given options.
// Default options are used if opts is nil.
func NewHandler(w io.Writer, opts *Options) slog.Handler {
	if opts == nil {
		opts = new(Options)
	}

	return &handler{
		w:           w,
		mu:          new(sync.Mutex),
		level:       cmp.Or[slog.Leveler](opts.Level, defaultLevel),
		timeFormat:  cmp.Or(opts.TimeFormat, defaultTimeFormat),
		replaceAttr: opts.ReplaceAttr,
		addSource:   opts.AddSource,
	}
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
//	INFO | Request processed | sku=24A2 branch=manhattan time="2023-04-02 10:50:09 EDT"
//	ERROR | Connection failed | time="2024-01-23 17:14:03 UTC"
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
	if h.replaceAttr != nil {
		groups = pooled.NewStringSlice()
		defer groups.Free()
		groups.Append(h.groups...)
	}

	appendLevel(buf, r.Level)
	buf.WriteByte(' ')
	buf.WriteString(r.Message)
	buf.WriteString(" |")
	buf.WriteString(h.attrs)
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(buf, a, h.groupPrefix, groups)
		return true
	})
	if h.addSource {
		src := r.Source()
		if src != nil && (src.File != "" || src.Line != 0) {
			sourceAttr := slog.String(slog.SourceKey, src.File+":"+strconv.Itoa(src.Line))
			h.appendAttr(buf, sourceAttr, "", nil)
		}
	}
	if !r.Time.IsZero() {
		h.appendAttr(buf, slog.Time(slog.TimeKey, r.Time), "", nil)
	}
	buf.WriteByte('\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(*buf)

	return err //nolint:wrapcheck // Do as slog does and pass this unwrapped.
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
	switch level {
	case slog.LevelDebug:
		buf.WriteString("DEBUG |")
	case slog.LevelInfo:
		buf.WriteString(" INFO |")
	case slog.LevelWarn:
		buf.WriteString(" WARN |")
	case slog.LevelError:
		buf.WriteString("ERROR |")
	default:
		buf.WriteByte(' ')
		buf.WriteString(level.String())
		buf.WriteString(" |")
	}
}

func (h *handler) appendAttr(buf *pooled.Buffer, a slog.Attr, groupPrefix string, groups *pooled.StringSlice) {
	a.Value = a.Value.Resolve()
	if h.replaceAttr != nil && a.Value.Kind() != slog.KindGroup {
		var gs []string
		if groups != nil {
			gs = *groups
		}
		a = h.replaceAttr(gs, a)
		a.Value = a.Value.Resolve()
	}
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() == slog.KindGroup {
		h.appendGroup(buf, a, groupPrefix, groups)
		return
	}
	appendKey(buf, groupPrefix, a.Key)
	h.appendVal(buf, a.Value)
}

func (h *handler) appendGroup(buf *pooled.Buffer, a slog.Attr, groupPrefix string, groups *pooled.StringSlice) {
	attrs := a.Value.Group()
	if len(attrs) == 0 {
		return
	}
	newGroupPrefix := groupPrefix
	if a.Key != "" {
		if groupPrefix == "" {
			newGroupPrefix = a.Key
		} else {
			newGroupPrefix = groupPrefix + "." + a.Key
		}
		if groups != nil {
			groups.Append(a.Key)
		}
	}
	for _, child := range attrs {
		h.appendAttr(buf, child, newGroupPrefix, groups)
	}
	if a.Key != "" && groups != nil {
		*groups = (*groups)[:groups.Len()-1]
	}
}

func appendKey(buf *pooled.Buffer, groupPrefix, key string) {
	buf.WriteByte(' ')
	var fullKey string
	if groupPrefix != "" {
		fullKey = groupPrefix + "." + key
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
		// Quote based on what the format produced, not on the format itself.
		start := len(*buf)
		*buf = val.Time().AppendFormat(*buf, h.timeFormat)
		if s := string((*buf)[start:]); needsQuoting(s) {
			*buf = (*buf)[:start]
			appendQuoted(buf, s)
		}
	case slog.KindGroup, slog.KindLogValuer, slog.KindAny:
		appendAny(buf, val.Any())
	}
}

func appendAny(buf *pooled.Buffer, v any) {
	defer func() {
		if recover() != nil {
			// fmt recovers from nil-receiver panics and prints <nil>.
			appendString(buf, fmt.Sprint(v))
		}
	}()

	if tm, ok := v.(encoding.TextMarshaler); ok {
		data, err := tm.MarshalText()
		if err != nil {
			appendString(buf, "!ERROR:"+err.Error())
			return
		}
		appendString(buf, string(data))

		return
	}
	if err, ok := v.(error); ok {
		appendString(buf, err.Error())
		return
	}
	appendString(buf, fmt.Sprint(v))
}

func appendString(buf *pooled.Buffer, s string) {
	if needsQuoting(s) {
		appendQuoted(buf, s)
	} else {
		buf.WriteString(s)
	}
}

func appendQuoted(buf *pooled.Buffer, s string) {
	if needsEscaping(s) {
		*buf = strconv.AppendQuote(*buf, s)
		return
	}
	buf.WriteByte('"')
	buf.WriteString(s)
	buf.WriteByte('"')
}

func needsEscaping(s string) bool {
	for i := range len(s) {
		if c := s[i]; c < 0x20 || c >= 0x7f || c == '"' || c == '\\' {
			return true
		}
	}

	return false
}

func needsQuoting(s string) bool {
	if s == "" {
		return true
	}
	for i := 0; i < len(s); {
		b := s[i]
		// Handle ASCII characters quickly.
		if b < utf8.RuneSelf {
			if mustQuoteChars[b] {
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
// mustQuoteChars reports, for each ASCII byte, whether a logfmt key or value
// that contains it must be quoted. needsQuoting handles non-ASCII runes.
//
// Note that a map is far slower.
var mustQuoteChars = func() [utf8.RuneSelf]bool {
	var t [utf8.RuneSelf]bool
	for b := range byte(0x20) {
		t[b] = true
	}
	t[0x7f], t[' '], t['"'], t['='] = true, true, true, true

	return t
}()
