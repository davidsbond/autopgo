// Package event provides types for publishing and consuming events from event buses.
package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	_ "gocloud.dev/pubsub/awssnssqs"
	_ "gocloud.dev/pubsub/azuresb"
	_ "gocloud.dev/pubsub/gcppubsub"
	_ "gocloud.dev/pubsub/kafkapubsub"
	_ "gocloud.dev/pubsub/natspubsub"
)

type (
	// The Envelope type describes the structure of events published to and read from an event bus.
	Envelope struct {
		// A unique identifier for the event.
		ID string `json:"id"`
		// The time the event was published.
		Timestamp time.Time `json:"timestamp"`
		// Denotes the structure of the Payload field and the type to use for decoding.
		Type string `json:"type"`
		// The raw JSON of the event payload.
		Payload json.RawMessage `json:"payload"`
	}

	// The Payload interface describes types that can be used as event payloads.
	Payload interface {
		// Type should return a string unique to the event type.
		Type() string
		// Key should return a string used to preserve ordering of events.
		Key() string
	}
)

func wrap(payload Payload) (Envelope, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		ID:        uuid.NewString(),
		Timestamp: time.Now().UTC(),
		Type:      payload.Type(),
		Payload:   b,
	}, nil
}

// Unmarshal the event payload into the type specified via the type parameter.
func Unmarshal[T Payload](envelope Envelope) (T, error) {
	var t T
	if err := json.Unmarshal(envelope.Payload, &t); err != nil {
		return t, err
	}

	return t, nil
}
