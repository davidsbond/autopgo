// Package api provides types and functions used for uniform handling of HTTP responses and errors.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/davidsbond/autopgo/internal/logger"
)

type (
	// The Error type represents an error as returned by the API.
	Error struct {
		// The error message.
		Message string `json:"message"`
		// The HTTP status code.
		Code int `json:"code"`
	}
)

func (e Error) Error() string {
	return fmt.Sprintf("%s (%d)", e.Message, e.Code)
}

// ErrorResponse writes a JSON-encoded Error to the http.ResponseWriter.
func ErrorResponse(ctx context.Context, w http.ResponseWriter, message string, code int) {
	e := Error{
		Message: message,
		Code:    code,
	}

	Respond(ctx, w, code, e)
}

// Respond writes a JSON-encoded HTTP response to the http.ResponseWriter.
func Respond(ctx context.Context, w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.FromContext(ctx).
			With(slog.String("error", err.Error())).
			ErrorContext(ctx, "failed to write response")
	}
}

// Decode the contents of the io.Reader into a new instance of type T. Expects a JSON-encoded object.
func Decode[T any](r io.Reader) (T, error) {
	var t T
	if err := json.NewDecoder(r).Decode(&t); err != nil {
		return t, err
	}
	return t, nil
}
