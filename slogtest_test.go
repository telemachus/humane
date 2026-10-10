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
// Credit goes to Jonathan Amsterdam for both.
func TestSlogtest(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	newHandler := func(*testing.T) slog.Handler {
		buf.Reset()
		return humane.NewHandler(&buf, &humane.Options{TimeFormat: time.RFC3339})
	}
	result := func(t *testing.T) map[string]any {
		t.Helper()
		m, err := parseHumane(buf.Bytes())
		if err != nil {
			t.Fatal(err)
		}

		return m
	}
	slogtest.Run(t, newHandler, result)
}

func parseHumane(bs []byte) (map[string]any, error) {
	top := map[string]any{}
	s := string(bytes.TrimSpace(bs))
	// Divide each line into level, message, and key-value pairs. Humane
	// separates the level from the message with " | ", and the message
	// from the pairs with " |", which is all that remains when there
	// are no pairs.
	level, afterLevel, _ := strings.Cut(s, " | ")
	msg, attrs, _ := strings.Cut(afterLevel, " |")
	top[slog.LevelKey] = strings.TrimSpace(level)
	top[slog.MessageKey] = strings.TrimSpace(msg)
	// The rest of the line contains kv pairs that we can (roughly) divide
	// by spaces. This is crude since it will split a quoted key or value
	// that contains a space. For this test, however, this will work---as
	// long as we set a time format without whitespace.
	s = strings.TrimSpace(attrs)
	for s != "" {
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
