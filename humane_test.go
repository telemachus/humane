package humane_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/telemachus/humane"
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

func TestHumaneNilOpts(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	slog.New(humane.NewHandler(&buf, nil))
}

func TestHumaneBasic(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	opts := &humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(humane.NewHandler(&buf, opts))
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
	opts := &humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(humane.NewHandler(&buf, opts))
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
	opts := &humane.Options{ReplaceAttr: removeTime, Level: slog.LevelError}
	logger := slog.New(humane.NewHandler(&buf, opts))
	logger.Info("Testing 1, 2, 3")
	got := buf.String()
	want := ""
	if got != want {
		t.Errorf(`logger.Info("Testing 1, 2, 3") = %q; want %q`, got, want)
	}
}

func TestHumaneAddSource(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	opts := &humane.Options{ReplaceAttr: removeTimeTrimSource, AddSource: true}
	logger := slog.New(humane.NewHandler(&buf, opts))
	_, _, line, _ := runtime.Caller(0) //nolint:dogsled // This is an ugly but normal use of runtime.Caller.
	logger.Info("foo")
	want := fmt.Sprintf(" INFO | foo | source=humane_test.go:%d\n", line+1)
	got := buf.String()
	if got != want {
		t.Errorf(`logger.Info("foo") = %q; want %q`, got, want)
	}
}

func TestHumaneCustomTimeFormat(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	timeFormat := "2006-01-02"
	opts := &humane.Options{TimeFormat: timeFormat}
	logger := slog.New(humane.NewHandler(&buf, opts))
	logger.Info("foo")
	got := buf.String()
	want := fmt.Sprintf(
		" INFO | foo | %s=%s\n",
		slog.TimeKey,
		time.Now().Format(timeFormat),
	)
	if got != want {
		t.Errorf(`logger.Info("foo") (TimeFormat %q) = %q; want %q`, timeFormat, got, want)
	}
}

func TestHumaneSlogGroup(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	opts := &humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(humane.NewHandler(&buf, opts))
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
		t.Errorf(`logger.Info("message") (Groups) = %q; want %q`, got, want)
	}
}

func TestHumaneWithGroup(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	opts := &humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(humane.NewHandler(&buf, opts).WithGroup("GROUP"))
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
			`logger.Info("message") (WithGroup and Groups) = %q; want %q`,
			got,
			want,
		)
	}
}

func TestHumaneWithAttrs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	opts := &humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(
		humane.NewHandler(&buf, opts).WithAttrs(
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
		t.Errorf(`logger.Info("message") (WithAttrs) = %q; want %q`, got, want)
	}
}

func TestHumaneWithGroupWithAttrs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	opts := &humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(humane.NewHandler(&buf, opts))
	logger = logger.WithGroup("g").With("a", 1).WithGroup("h").With("b", 2)
	logger.Info("message")
	got := buf.String()
	want := " INFO | message | g.a=1 g.h.b=2\n"
	if got != want {
		t.Errorf(`logger.Info("message") (WithGroup and WithAttrs) = %q; want %q`, got, want)
	}
}

func TestQuotingForAttrs(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		desc string
		want string
		args []any
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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			opts := &humane.Options{ReplaceAttr: removeTime}
			logger := slog.New(humane.NewHandler(&buf, opts))
			logger.Info("message", tc.args...)
			got := buf.String()
			if got != tc.want {
				t.Errorf("%s got %q; want %q", tc.desc, got, tc.want)
			}
		})
	}
}

func TestQuotingForGroupNames(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		groupName string
		want      string
	}{
		{
			name:      "group with space",
			groupName: "my group",
			want:      ` INFO | message | "my group.key"=value` + "\n",
		},
		{
			name:      "group with equals",
			groupName: "group=name",
			want:      ` INFO | message | "group=name.key"=value` + "\n",
		},
		{
			name:      "group with quote",
			groupName: `group"name`,
			want:      ` INFO | message | "group\"name.key"=value` + "\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			opts := &humane.Options{ReplaceAttr: removeTime}
			logger := slog.New(humane.NewHandler(&buf, opts))

			logger.WithGroup(tc.groupName).Info("message", "key", "value")

			got := buf.String()
			if got != tc.want {
				t.Errorf("group name %q: got %q; want %q", tc.groupName, got, tc.want)
			}
		})
	}
}

func TestHumaneConcurrentGroupHandling(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	opts := &humane.Options{ReplaceAttr: removeTime}
	handler := humane.NewHandler(&buf, opts)

	records := []slog.Record{
		func() slog.Record {
			r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg1", 0)
			r.AddAttrs(slog.Group("req",
				slog.Group("db", slog.String("query", "SELECT")),
				slog.String("id", "123")))
			return r
		}(),
		func() slog.Record {
			r := slog.NewRecord(time.Now(), slog.LevelWarn, "msg2", 0)
			r.AddAttrs(slog.Group("auth",
				slog.String("user", "alice"),
				slog.Group("perms", slog.Bool("admin", false))))
			return r
		}(),
	}

	const numGoroutines = 100
	const recordsPerGoroutine = 10
	var wg sync.WaitGroup

	// Synchronize start of goroutines for maximum race potential.
	start := make(chan struct{})

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			for j := range recordsPerGoroutine {
				record := records[j%len(records)]
				err := handler.Handle(context.Background(), record)
				if err != nil {
					t.Errorf("Handle failed: %v", err)
				}
			}
		}()
	}

	close(start)
	wg.Wait()

	output := buf.String()
	if !strings.Contains(output, "req.db.query=") {
		t.Error("Expected nested group output not found")
	}
	if !strings.Contains(output, "auth.perms.admin=") {
		t.Error("Expected nested group output not found")
	}
}

func TestUnnamedInlineGroup(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	opts := &humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(humane.NewHandler(&buf, opts))

	// Place attrs for unnamed groups on parent level.
	logger.Info("message",
		slog.Group("outer",
			slog.String("a", "1"),
			slog.Group("",
				slog.String("b", "2"),
				slog.String("c", "3"),
			),
			slog.String("d", "4"),
		),
	)

	got := buf.String()
	want := " INFO | message | outer.a=1 outer.b=2 outer.c=3 outer.d=4\n"
	if got != want {
		t.Errorf("logger.Info with unnamed group = %q; want %q", got, want)
	}
}

func TestNoAttrsGroup(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	opts := &humane.Options{ReplaceAttr: removeTime}
	logger := slog.New(humane.NewHandler(&buf, opts))

	// Ignore groups without attrs (even if they have a name).
	logger.Info("message",
		slog.String("before", "1"),
		slog.Group("empty"),
		slog.String("after", "2"),
	)

	got := buf.String()
	want := " INFO | message | before=1 after=2\n"
	if got != want {
		t.Errorf("logger.Info with empty inline group = %q; want %q", got, want)
	}
}

func TestReplaceAttrGroupsInWithAttrs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer

	var receivedGroups [][]string
	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		receivedGroups = append(receivedGroups, append([]string(nil), groups...))
		return a
	}

	opts := &humane.Options{ReplaceAttr: replaceAttr}
	logger := slog.New(humane.NewHandler(&buf, opts))
	logger.WithGroup("g1").WithGroup("g2").With("a", "1", "b", "2")

	// ReplaceAttr should have be called twice (for "a" and "b"),
	// and both calls should receive groups = ["g1", "g2"].
	want := [][]string{
		{"g1", "g2"},
		{"g1", "g2"},
	}

	if diff := cmp.Diff(want, receivedGroups); diff != "" {
		t.Errorf("ReplaceAttr groups mismatch (-want +got):\n%s", diff)
	}
}

func TestReplaceAttrGroupsSlice(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer

	type attrContext struct {
		Key    string
		Groups []string
	}
	var got []attrContext

	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		got = append(got, attrContext{
			// Avoid slice reuse.
			Groups: append([]string(nil), groups...),
			Key:    a.Key,
		})
		return a
	}

	opts := &humane.Options{ReplaceAttr: replaceAttr}
	logger := slog.New(humane.NewHandler(&buf, opts))

	logger.WithGroup("g1").WithGroup("g2").Info("message",
		slog.String("a", "1"),
		slog.Group("g3",
			slog.String("b", "2"),
		),
	)

	want := []attrContext{
		{Key: "a", Groups: []string{"g1", "g2"}},
		{Key: "b", Groups: []string{"g1", "g2", "g3"}},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ReplaceAttr contexts mismatch (-want +got):\n%s", diff)
	}
}

func TestAddSourceWithGroup(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	opts := &humane.Options{ReplaceAttr: removeTime, AddSource: true}
	logger := slog.New(humane.NewHandler(&buf, opts)).WithGroup("g1").WithGroup("g2")
	logger.Info("message", "key", "value")
	got := buf.String()
	if !strings.Contains(got, " source=") || strings.Contains(got, "g1.g2.source=") {
		t.Errorf("got %q; source attribute should be top-level", got)
	}
}

func TestReplaceAttrGroupsForSource(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	var sourceGroups []string
	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			sourceGroups = append([]string(nil), groups...)
		}
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}
	opts := &humane.Options{ReplaceAttr: replaceAttr, AddSource: true}
	logger := slog.New(humane.NewHandler(&buf, opts)).WithGroup("g1")
	logger.Info("message")
	if len(sourceGroups) != 0 {
		t.Errorf("got %v; ReplaceAttr for source should receive nil for groups", sourceGroups)
	}
}
