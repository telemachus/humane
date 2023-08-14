package humane_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"testing/slogtest"
	"time"

	"github.com/telemachus/humane"
)

// This code is (very lightly) adapted from examples in slog and slogtest.
// Thanks to Jonathan Amsterdam for both.
func TestSlogtest(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := humane.NewHandler(&buf, &humane.Options{TimeFormat: time.RFC3339})
	results := func() []map[string]any {
		ms := []map[string]any{}
		for _, line := range bytes.Split(buf.Bytes(), []byte{'\n'}) {
			if len(line) == 0 {
				continue
			}
			m, err := parseHumane(line)
			if err != nil {
				t.Fatal(err)
			}
			ms = append(ms, m)
		}
		return ms
	}
	if err := slogtest.TestHandler(h, results); err != nil {
		t.Error(err)
	}
}

func parseHumane(bs []byte) (map[string]any, error) {
	top := map[string]any{}
	s := string(bytes.TrimSpace(bs))
	// First, we need to divide each line into three parts (level, message,
	// kv pairs). Humane delimits these three parts by " | ".
	// Then we need to create proper key-value pairs for level and message.
	pieces := strings.Split(s, " | ")
	top[slog.LevelKey] = strings.TrimSpace(pieces[0])
	top[slog.MessageKey] = strings.TrimSpace(pieces[1])
	// The rest of the line contains kv pairs that we can (roughly) divide
	// by spaces. This is crude since it will split a quoted key or value
	// that contains a space. For this test, however, this will work---as
	// long as I make sure to set a time format without whitespace.
	s = pieces[2]
	for len(s) > 0 {
		kv, rest, _ := strings.Cut(s, " ")
		k, value, found := strings.Cut(kv, "=")
		if !found {
			return nil, fmt.Errorf("no '=' in %q", kv)
		}
		keys := strings.Split(k, ".")
		// Populate a tree of maps for a dotted path such as "a.b.c=x".
		m := top
		for _, key := range keys[:len(keys)-1] {
			x, ok := m[key]
			var m2 map[string]any
			if !ok {
				m2 = map[string]any{}
				m[key] = m2
			} else {
				m2, ok = x.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("value for %q in composite key %q is not map[string]any", key, k)
				}
			}
			m = m2
		}
		m[keys[len(keys)-1]] = value
		s = rest
	}
	return top, nil
}
