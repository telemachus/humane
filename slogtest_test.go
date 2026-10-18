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
	line := string(bytes.TrimSpace(bs))
	// Divide each line into level, message, and key-value pairs. Humane
	// separates the level from the message with " | ", and the message
	// from the pairs with " |", which is all that remains when there
	// are no pairs.
	level, afterLevel, _ := strings.Cut(line, " | ")
	msg, attrs, _ := strings.Cut(afterLevel, " |")
	top[slog.LevelKey] = strings.TrimSpace(level)
	top[slog.MessageKey] = strings.TrimSpace(msg)
	// The rest of the line contains kv pairs that we can (roughly) divide
	// by spaces. This is crude since it will split a quoted key or value
	// that contains a space. For this test, however, this will work---as
	// long as we set a time format without whitespace.
	rest := strings.TrimSpace(attrs)
	for rest != "" {
		var kv string
		kv, rest, _ = strings.Cut(rest, " ")
		path, value, found := strings.Cut(kv, "=")
		if !found {
			return nil, fmt.Errorf("no '=' in %q", kv)
		}
		if err := setPath(top, path, value); err != nil {
			return nil, err
		}
	}

	return top, nil
}

// setPath populates a tree of maps under top for a dotted path such as
// "a.b.c", and it stores value at the final key.
func setPath(top map[string]any, path, value string) error {
	keys := strings.Split(path, ".")
	node := top
	for _, key := range keys[:len(keys)-1] {
		x, ok := node[key]
		var child map[string]any
		if !ok {
			child = map[string]any{}
			node[key] = child
		} else {
			child, ok = x.(map[string]any)
			if !ok {
				return fmt.Errorf("value for %q in composite key %q is not map[string]any", key, path)
			}
		}
		node = child
	}
	node[keys[len(keys)-1]] = value

	return nil
}
