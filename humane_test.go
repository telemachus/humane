package humane_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/telemachus/humane"
	"golang.org/x/exp/slog"
)

func removeTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}

func removeTimeTrimSource(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey:
		return slog.Attr{}
	case slog.SourceKey:
		return slog.String(slog.SourceKey, filepath.Base(a.Value.String()))
	default:
		return a
	}
}

func TestHumaneBasic(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ho := humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(ho.NewHandler(&buf))
	logger.Info("foo")
	got := buf.String()
	want := " INFO | foo |\n"
	if got != want {
		t.Errorf(`logger.Info("foo") = %q; want %q`, got, want)
	}
}

func TestKeepTimeKeyInGroup(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ho := humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(ho.NewHandler(&buf))
	logger.WithGroup("request").Info("foo", slog.String("time", "3:00pm"))
	got := buf.String()
	want := " INFO | foo | request.time=3:00pm\n"
	if got != want {
		t.Errorf(`logger.WithGroup("request").Info("foo", "time", "3:00pm") = %q; want %q`, got, want)
	}
}

func TestHumaneCustomLevel(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ho := humane.Options{ReplaceAttr: removeTime, Level: slog.LevelError}
	logger := slog.New(ho.NewHandler(&buf))
	logger.Info("wtf?")
	got := buf.String()
	want := ""
	if got != want {
		t.Errorf(`logger.Info("wtf?") = %q; want %q`, got, want)
	}
}

func TestHumaneAddSource(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ho := humane.Options{ReplaceAttr: removeTimeTrimSource, AddSource: true}
	logger := slog.New(ho.NewHandler(&buf))
	logger.Info("foo")
	got := buf.String()
	want := " INFO | foo | source=humane_test.go:76\n"
	if got != want {
		t.Errorf(`logger.Info("foo") = %q; want %q`, got, want)
	}
}

func TestHumaneCustomTimeFormat(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	timeFormat := "2006-01-02"
	ho := humane.Options{TimeFormat: timeFormat}
	logger := slog.New(ho.NewHandler(&buf))
	logger.Info("foo")
	got := buf.String()
	want := fmt.Sprintf(
		" INFO | foo | %s=%s\n",
		slog.TimeKey,
		time.Now().Format(timeFormat),
	)
	if got != want {
		t.Errorf(`logger.Info("foo") (+ TimeFormat) = %q; want %q`, got, want)
	}
}

func TestHumaneSlogGroup(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ho := humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(ho.NewHandler(&buf))
	logger.Info("message",
		slog.Group(
			"foo",
			slog.Int("c", 3),
			slog.Group(
				"bar",
				slog.Int("d", 4),
			),
		),
		slog.Int("c", 3),
	)
	got := buf.String()
	want := " INFO | message | foo.c=3 foo.bar.d=4 c=3\n"
	if got != want {
		t.Errorf(`logger.Info("message") (+ Groups) = %q; want %q`, got, want)
	}
}

func TestHumaneWithGroup(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ho := humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(ho.NewHandler(&buf).WithGroup("GROUP"))
	logger.Info("message",
		slog.Group(
			"foo",
			slog.Int("c", 3),
			slog.Group(
				"bar",
				slog.Int("d", 4),
			),
		),
		slog.Int("c", 3),
	)
	got := buf.String()
	want := " INFO | message | GROUP.foo.c=3 GROUP.foo.bar.d=4 GROUP.c=3\n"
	if got != want {
		t.Errorf(
			`logger.Info("message") (+ WithGroup and Groups) = %q; want %q`,
			got,
			want,
		)
	}
}

func TestHumaneWithAttrs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ho := humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(
		ho.NewHandler(&buf).WithAttrs(
			[]slog.Attr{slog.Int("c", 3), slog.String("foo", "bar")},
		),
	)
	logger.Info("message",
		slog.Group(
			"foo",
			slog.Int("c", 3),
			slog.Group("bar", slog.Int("d", 4)),
		),
		slog.Int("c", 3),
	)
	got := buf.String()
	want := " INFO | message | c=3 foo=bar foo.c=3 foo.bar.d=4 c=3\n"
	if got != want {
		t.Errorf(`logger.Info("message") (+WithAttrs) = %q; want %q`, got, want)
	}
}

func TestHumaneWithGroupWithAttrs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ho := humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(ho.NewHandler(&buf))
	logger = logger.WithGroup("g").With("a", 1).WithGroup("h").With("b", 2)
	logger.Info("message")
	got := buf.String()
	want := " INFO | message | g.a=1 g.h.b=2\n"
	if got != want {
		t.Errorf(`logger.Info("message") (+WithGroup + WithAttrs) = %q; want %q`, got, want)
	}
}

func TestHumaneNeedsQuoting(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		desc string
		args []any
		want string
	}{
		{
			name: "space in value",
			desc: `log.Info("foo", "bar bar")`,
			args: []any{"foo", "bar bar"},
			want: ` INFO | message | foo="bar bar"` + "\n",
		},
		{
			name: "equal in value",
			desc: `log.Info("foo", "bar=bar")`,
			args: []any{"foo", "bar=bar"},
			want: ` INFO | message | foo="bar=bar"` + "\n",
		},
		{
			name: "quote in value",
			desc: `log.Info("foo", "bar"bar")`,
			args: []any{"foo", `bar"bar`},
			want: ` INFO | message | foo="bar\"bar"` + "\n",
		},
		{
			name: "space in key",
			desc: `log.Info("foo foo", "bar")`,
			args: []any{"foo foo", "bar"},
			want: ` INFO | message | "foo foo"=bar` + "\n",
		},
		{
			name: "equal in key",
			desc: `log.Info("foo=foo", "bar")`,
			args: []any{"foo=foo", "bar"},
			want: ` INFO | message | "foo=foo"=bar` + "\n",
		},
		{
			name: "quote in key",
			desc: `log.Info("foo"foo", "bar")`,
			args: []any{`foo"foo`, "bar"},
			want: ` INFO | message | "foo\"foo"=bar` + "\n",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			ho := humane.Options{ReplaceAttr: removeTime}
			logger := slog.New(ho.NewHandler(&buf))
			logger.Info("message", tc.args...)
			got := buf.String()
			if got != tc.want {
				t.Errorf("%s got %q; want %q", tc.desc, got, tc.want)
			}
		})
	}
}
