// Package logger provides functions for working with structured logging via context.Context.
package logger

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/davidsbond/autopgo/internal/server"
)

type (
	ctxKey struct{}
)

// FromContext attempts to return a slog.Logger stored within the provided context.Context. Returns slog.Default if
// the context does not contain a logger.
func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(ctxKey{}).(*slog.Logger)
	if logger == nil || !ok {
		return slog.Default()
	}

	return logger
}

// ToContext adds the provided slog.Logger to the given context.Context and returns it.
func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// Middleware is a server.Middleware implementation that ensures each inbound HTTP request's context contains the
// slog.Logger.
func Middleware(logger *slog.Logger) server.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := ToContext(r.Context(), logger)

			logger.With(
				slog.String("http.method", r.Method),
				slog.String("http.path", r.URL.Path),
			).DebugContext(ctx, "handling request")

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LevelFromString converts a provided string into its corresponding slog.Level. Returns slog.LevelInfo for an invalid
// string.
func LevelFromString(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
