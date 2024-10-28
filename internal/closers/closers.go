// Package closers providers helpers for working with io.Closer implementations.
package closers

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/davidsbond/autopgo/internal/logger"
)

// Close the provider io.Closer implementation, logging on error.
func Close(ctx context.Context, c io.Closer) {
	if err := c.Close(); err != nil {
		logger.FromContext(ctx).With(
			slog.String("error", err.Error()),
			slog.String("type", fmt.Sprintf("%T", c)),
		).ErrorContext(ctx, "failed to close")
	}
}
