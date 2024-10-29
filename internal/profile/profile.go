package profile

import (
	"context"
	"io"
	"unicode/utf8"

	"github.com/davidsbond/autopgo/internal/event"
)

type (
	// The BlobRepository interface describes types that can interact with blob storage provides such as S3, GCS etc.
	BlobRepository interface {
		// NewWriter should return an io.WriterCloser implementation that will store data written to it under the
		// provided key.
		NewWriter(ctx context.Context, key string) (io.WriteCloser, error)
		// NewReader should return an io.ReadCloser implementation that will read data from blob storage at the
		// provided key. It should return blob.ErrNotExist if no data exists at the given key.
		NewReader(ctx context.Context, key string) (io.ReadCloser, error)
		// Delete should remove data stored under the given key from the blob store. It should return blob.ErrNotExist
		// if no object exists at the given key.
		Delete(ctx context.Context, key string) error
	}

	// The EventWriter interface describes types that can publish events onto an event bus such as Kafka, NATS, SQS
	// etc.
	EventWriter interface {
		// Write should push the given event payload onto the event bus.
		Write(ctx context.Context, evt event.Payload) error
	}

	// The UploadedEvent type is an event.Payload implementation describing a single profile that has been uploaded.
	UploadedEvent struct {
		// The application the profile relates to.
		App string `json:"app"`
		// The location of the profile within blob storage.
		ProfileKey string `json:"profileKey"`
	}

	// The MergedEvent type is an event.Payload implementation describing a profile that has been successfully merged
	// into the base profile.
	MergedEvent struct {
		// The application the profile relates to.
		App string `json:"app"`
		// The location of the uploaded profile within blob storage.
		ProfileKey string `json:"profileKey"`
		// The location of the base profile that has been merged.
		MergedKey string `json:"mergedKey"`
	}
)

// Constants for event types.
const (
	EventTypeUploaded = "profile.uploaded"
	EventTypeMerged   = "profile.merged"
)

// Type returns EventTypeUploaded.
func (e UploadedEvent) Type() string {
	return EventTypeUploaded
}

// Key returns the application name.
func (e UploadedEvent) Key() string {
	return e.App
}

// Type returns EventTypeMerged.
func (e MergedEvent) Type() string {
	return EventTypeMerged
}

// Key returns the application name.
func (e MergedEvent) Key() string {
	return e.App
}

// IsValidAppName returns false if the application name contains any characters that are not a-z, 0-9 or hyphens.
func IsValidAppName(app string) bool {
	for _, r := range app {
		if !utf8.ValidRune(r) {
			return false
		}

		if r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}

		return false
	}

	return true
}
