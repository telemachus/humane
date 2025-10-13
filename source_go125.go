//go:build go1.25

package humane

import "log/slog"

func source(r slog.Record) *slog.Source {
	return r.Source()
}
