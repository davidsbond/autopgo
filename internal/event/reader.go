package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"gocloud.dev/pubsub"

	"github.com/davidsbond/autopgo/internal/logger"
)

type (
	// The Reader type is used to consume messages from an event bus.
	Reader struct {
		events    *pubsub.Subscription
		lastError error
	}

	// The Handler type is a function used to handle a single inbound event.
	Handler func(ctx context.Context, e Envelope) error
)

// NewReader returns a new instance of the Reader type that will consume events as described in its URL string. See
// the gocloud.dev documentation for more information on provider specific urls.
func NewReader(ctx context.Context, url string) (*Reader, error) {
	subscription, err := pubsub.OpenSubscription(ctx, url)
	if err != nil {
		return nil, err
	}

	return &Reader{
		events: subscription,
	}, nil
}

// Close the connection to the event bus.
func (r *Reader) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return r.events.Shutdown(ctx)
}

// Read messages from the event bus whose types appear in the types parameter. For each event, the Handler is invoked.
// If the handler returns an error, the message is nack'd where supported by the event bus. This method blocks until
// the Handler returns an error or the provided context is cancelled.
func (r *Reader) Read(ctx context.Context, types []string, h Handler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			message, err := r.events.Receive(ctx)
			if err != nil {
				r.lastError = err
				return err
			}

			r.lastError = nil

			var envelope Envelope
			if err = json.Unmarshal(message.Body, &envelope); err != nil {
				nack(message)
				return fmt.Errorf("could not unmarshal envelope: %w", err)
			}

			log := logger.FromContext(ctx).With(
				slog.String("event.id", envelope.ID),
				slog.String("event.type", envelope.Type),
				slog.Time("event.timestamp", envelope.Timestamp),
			)

			if !slices.Contains(types, envelope.Type) {
				log.DebugContext(ctx, "ignoring event")
				message.Ack()
				continue
			}

			log.DebugContext(ctx, "consumed event")
			if err = h(ctx, envelope); err != nil {
				nack(message)
				return fmt.Errorf("failed to handle event %s: %w", envelope.ID, err)
			}

			message.Ack()
		}
	}
}

func nack(message *pubsub.Message) {
	if message.Nackable() {
		message.Nack()
	}
}

// Name returns "event-reader" This method is used to implement the operation.Checker interface for use in health checks.
func (r *Reader) Name() string {
	return "event-reader"
}

// Check returns the last non-nil error when trying to read an event. This method is used to implement the
// operation.Checker interface for use in health checks.
func (r *Reader) Check(_ context.Context) error {
	return r.lastError
}
