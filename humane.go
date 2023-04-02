// Package humane provides a slog.Handler for a human-friendly version of logfmt.
// The idea for this format comes from Brandur Leach in his original post about
// logfmt. See https://brandur.org/logfmt for my inspiration.
package humane

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/telemachus/humane/internal/withsupport"
	"golang.org/x/exp/slog"
)

var (
	defaultLevel      = slog.LevelInfo
	defaultTimeFormat = "2006-01-02T03:04.05 MST"
)

type handler struct {
	w           io.Writer
	s           strings.Builder
	goa         *withsupport.GroupOrAttrs
	mu          sync.Mutex
	level       slog.Leveler
	timeFormat  string
	replaceAttr func(groups []string, a slog.Attr) slog.Attr
	addSource   bool
}

// Options are options for a [Handler].
type Options struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// Humane uses [slog.LevelInfo] as its default level.
	Level slog.Leveler
	// ReplaceAttr rewrites Attrs before they are logged.
	// By default, ReplaceAttr is nil, and no changes are made to Attrs.
	// Note: Humane's handler does not apply ReplaceAttr to the level or
	// message Attrs because they receive specific treatment by the
	// handler.
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
	// TimeFormat defaults to "2006-01-02T03:04.05 MST".
	TimeFormat string
	// AddSource defaults to false.
	AddSource bool
}

// NewHandler constructs a Handler with default options.
func NewHandler(w io.Writer) slog.Handler {
	return Options{}.NewHandler(w)
}

// NewHandler constructs a Handler with the given options.
func (opts Options) NewHandler(w io.Writer) slog.Handler {
	h := &handler{
		w:           w,
		level:       opts.Level,
		timeFormat:  opts.TimeFormat,
		replaceAttr: opts.ReplaceAttr,
		addSource:   opts.AddSource,
	}
	if opts.Level == nil {
		h.level = defaultLevel
	}
	if h.timeFormat == "" {
		h.timeFormat = defaultTimeFormat
	}
	return h
}

// Enabled determines whether a handler logs at a given level.
func (h *handler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

// WithGroup returns a new handler with the given group appended to whatever
// groups the receiver already has.
func (h *handler) WithGroup(name string) slog.Handler {
	h2 := h.clone()
	h2.goa = h2.goa.WithGroup(name)
	return h2
}

// WithAttrs returns a new handler that has the attributes of the receiver plus
// the attributes passed to WithAttrs.
func (h *handler) WithAttrs(as []slog.Attr) slog.Handler {
	h2 := h.clone()
	h2.goa = h2.goa.WithAttrs(as)
	return h2
}

func (h *handler) clone() *handler {
	return &handler{
		w:           h.w,
		level:       h.level,
		timeFormat:  h.timeFormat,
		replaceAttr: h.replaceAttr,
		addSource:   h.addSource,
	}
}

// Handle handles a given record.
func (h *handler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	fmt.Fprintf(&h.s, "%s | %s |", r.Level.String(), r.Message)
	h.mu.Unlock()
	groups := h.goa.Apply(h.formatAttr)
	r.Attrs(func(a slog.Attr) {
		h.formatAttr(groups, a)
	})
	if h.addSource {
		// Skip slog.log, Handle, h.handle.
		sourceAttr := h.newSourceAttr(3)
		h.formatAttr(nil, sourceAttr)
	}
	timeAttr := slog.String(slog.TimeKey, r.Time.Format(h.timeFormat))
	h.formatAttr(nil, timeAttr)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := fmt.Fprintln(h.w, h.s.String())
	h.s.Reset()
	return err
}

func (h *handler) formatAttr(groups []string, a slog.Attr) {
	if a.Value.Kind() == slog.KindGroup {
		gs := a.Value.Group()
		if len(gs) == 0 {
			return
		}
		if a.Key != "" {
			groups = append(groups, a.Key)
		}
		for _, g := range gs {
			h.formatAttr(groups, g)
		}
		return
	}
	if h.replaceAttr != nil {
		a = h.replaceAttr(groups, a)
	}
	key := a.Key
	if key == "" {
		return
	}
	if len(groups) > 0 {
		key = strings.Join(groups, ".") + "." + key
	}
	h.writeAttr(slog.Any(key, a.Value))
}

func (h *handler) writeAttr(a slog.Attr) {
	k := a.Key
	if needsQuoting(k) {
		k = fmt.Sprintf("%q", k)
	}
	v := a.Value.String()
	if needsQuoting(v) {
		v = fmt.Sprintf("%q", v)
	}
	h.mu.Lock()
	fmt.Fprintf(&h.s, " %s=%s", k, v)
	h.mu.Unlock()
}

func (h *handler) newSourceAttr(calldepth int) slog.Attr {
	// Add 1 to calldepth for this function.
	calldepth++
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}
	return slog.String(slog.SourceKey, fmt.Sprintf("%s:%d", file, line))
}

func needsQuoting(s string) bool {
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			if !safeSet[b] {
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
// safeSet holds the value true if the ASCII character with the given array
// position can be represented as a logfmt key or value without being quoted.
//
// All values are true except for ' ', '"', and '='.
var safeSet = [utf8.RuneSelf]bool{
	' ':      false,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      false,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     true,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}
