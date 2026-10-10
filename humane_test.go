package humane_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/netip"
	"path/filepath"
	"runtime"
	"slices"
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
	h := humane.NewHandler(&buf, nil)
	if h.Enabled(t.Context(), slog.LevelDebug) {
		t.Error("Enabled(LevelDebug) = true; want false")
	}
	if !h.Enabled(t.Context(), slog.LevelInfo) {
		t.Error("Enabled(LevelInfo) = false; want true")
	}
	r := slog.NewRecord(time.Date(2026, time.October, 5, 15, 4, 5, 0, time.UTC), slog.LevelInfo, "foo", 0)
	if err := h.Handle(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := ` INFO | foo | time="2026-10-05 15:04:05 UTC"` + "\n"
	if got != want {
		t.Errorf("Handle with nil opts = %q; want %q", got, want)
	}
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
	h := humane.NewHandler(&buf, &humane.Options{TimeFormat: "2006-01-02"})
	r := slog.NewRecord(time.Date(2026, time.October, 5, 23, 59, 59, 0, time.UTC), slog.LevelInfo, "foo", 0)
	if err := h.Handle(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := " INFO | foo | time=2026-10-05\n"
	if got != want {
		t.Errorf(`Handle with TimeFormat "2006-01-02" = %q; want %q`, got, want)
	}
}

func TestHumaneTimeQuotedByOutput(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := humane.NewHandler(&buf, &humane.Options{TimeFormat: "Jan_2"})
	r := slog.NewRecord(time.Date(2026, time.October, 5, 12, 0, 0, 0, time.UTC), slog.LevelInfo, "foo", 0)
	if err := h.Handle(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := ` INFO | foo | time="Oct 5"` + "\n"
	if got != want {
		t.Errorf(`Handle with TimeFormat "Jan_2" = %q; want %q`, got, want)
	}
}

func TestHumaneEmptyKeyAndValue(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		want string
		args []any
	}{
		"empty key":   {args: []any{"", "v"}, want: " INFO | m | \"\"=v\n"},
		"empty value": {args: []any{"e", ""}, want: " INFO | m | e=\"\"\n"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			logger := slog.New(humane.NewHandler(&buf, &humane.Options{ReplaceAttr: removeTime}))
			logger.Info("m", tc.args...)
			if got := buf.String(); got != tc.want {
				t.Errorf("logger.Info(%q, %q...) = %q; want %q", "m", tc.args, got, tc.want)
			}
		})
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
		{
			name: "newline in value",
			desc: `log.Info("foo", "bar\nbar")`,
			args: []any{"foo", "bar\nbar"},
			want: ` INFO | message | foo="bar\nbar"` + "\n",
		},
		{
			name: "backslash in quoted value",
			desc: `log.Info("foo", "a\b c")`,
			args: []any{"foo", `a\b c`},
			want: ` INFO | message | foo="a\\b c"` + "\n",
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

	r1 := slog.NewRecord(time.Now(), slog.LevelInfo, "msg1", 0)
	r1.AddAttrs(slog.Group("req",
		slog.Group("db", slog.String("query", "SELECT")),
		slog.String("id", "123")))
	r2 := slog.NewRecord(time.Now(), slog.LevelWarn, "msg2", 0)
	r2.AddAttrs(slog.Group("auth",
		slog.String("user", "alice"),
		slog.Group("perms", slog.Bool("admin", false))))
	records := []slog.Record{r1, r2}

	const numGoroutines = 100
	const recordsPerGoroutine = 10
	var wg sync.WaitGroup

	// Synchronize start of goroutines for maximum race potential.
	start := make(chan struct{})

	for range numGoroutines {
		wg.Go(func() {
			<-start

			for j := range recordsPerGoroutine {
				record := records[j%len(records)]
				err := handler.Handle(t.Context(), record)
				if err != nil {
					t.Errorf("Handle failed: %v", err)
				}
			}
		})
	}

	close(start)
	wg.Wait()

	const (
		line1 = " INFO | msg1 | req.db.query=SELECT req.id=123\n"
		line2 = " WARN | msg2 | auth.user=alice auth.perms.admin=false\n"
	)
	want := map[string]int{
		line1: numGoroutines * recordsPerGoroutine / 2,
		line2: numGoroutines * recordsPerGoroutine / 2,
	}
	got := map[string]int{}
	for line := range strings.Lines(buf.String()) {
		got[line]++
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("concurrent Handle output mismatch (-want +got):\n%s", diff)
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
		receivedGroups = append(receivedGroups, slices.Clone(groups))

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
			Groups: slices.Clone(groups),
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
			sourceGroups = slices.Clone(groups)
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

func TestHumaneNilTextMarshaler(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(humane.NewHandler(&buf, &humane.Options{ReplaceAttr: removeTime}))
	var ip *netip.Addr
	logger.Info("m", "ip", ip)
	got := buf.String()
	want := " INFO | m | ip=<nil>\n"
	if got != want {
		t.Errorf(`logger.Info("m", "ip", (*netip.Addr)(nil)) = %q; want %q`, got, want)
	}
}

func TestReplaceAttrReturnsGroup(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	replace := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == "y" && len(groups) == 0 {
			return slog.Group("y", slog.Int("a", 1))
		}

		return removeTime(groups, a)
	}
	logger := slog.New(humane.NewHandler(&buf, &humane.Options{ReplaceAttr: replace}))
	logger.Info("m", "y", 0)
	got := buf.String()
	want := " INFO | m | y.a=1\n"
	if got != want {
		t.Errorf("ReplaceAttr returning a group = %q; want %q", got, want)
	}
}

func TestHumaneNonStandardLevel(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(humane.NewHandler(&buf, &humane.Options{ReplaceAttr: removeTime}))
	logger.Log(t.Context(), slog.LevelInfo+2, "m")
	got := buf.String()
	want := " INFO+2 | m |\n"
	if got != want {
		t.Errorf("logger.Log(LevelInfo+2, %q) = %q; want %q", "m", got, want)
	}
}
