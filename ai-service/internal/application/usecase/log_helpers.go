package usecase

import "log/slog"

// Tiny wrappers around slog.Attr to keep call-sites compact.
func slogString(k, v string) slog.Attr { return slog.String(k, v) }
func slogInt(k string, v int) slog.Attr { return slog.Int(k, v) }
func slogError(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "")
	}
	return slog.String("error", err.Error())
}
