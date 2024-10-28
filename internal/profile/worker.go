package profile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"

	"github.com/davidsbond/autopgo/internal/logger"

	"github.com/google/pprof/profile"

	"github.com/davidsbond/autopgo/internal/blob"
	"github.com/davidsbond/autopgo/internal/closers"
	"github.com/davidsbond/autopgo/internal/event"
)

type (
	// The Worker type is used to handle profile events and merge uploaded profiles together into a single
	// base profile.
	Worker struct {
		reader EventReader
		writer EventWriter
		blobs  BlobRepository
	}
)

// NewWorker returns a new instance of the Worker type that will read and write profile data via the BlobRepository
// implementation, read events via the EventReader implementation and publish events via the EventWriter implementation.
func NewWorker(blobs BlobRepository, reader EventReader, writer EventWriter) *Worker {
	return &Worker{
		blobs:  blobs,
		reader: reader,
		writer: writer,
	}
}

// Work inbound profile events. When an EventTypeUploaded event is handled, the newly uploaded profile is merged into
// the base profile. When EventTypeMerged is handled, the uploaded profile is deleted from blob storage.
func (w *Worker) Work(ctx context.Context) error {
	types := []string{
		EventTypeMerged,
		EventTypeUploaded,
	}

	return w.reader.Read(ctx, types, func(ctx context.Context, evt event.Envelope) error {
		switch evt.Type {
		case EventTypeUploaded:
			return w.handleEventTypeUploaded(ctx, evt)
		case EventTypeMerged:
			return w.handleEventTypeMerged(ctx, evt)
		default:
			return nil
		}
	})
}

func (w *Worker) handleEventTypeUploaded(ctx context.Context, evt event.Envelope) error {
	payload, err := event.Unmarshal[UploadedEvent](evt)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log := logger.FromContext(ctx).With(
		slog.String("profile.key", payload.ProfileKey),
		slog.String("profile.app", payload.App),
	)

	newProfileReader, err := w.blobs.NewReader(ctx, payload.ProfileKey)
	switch {
	case errors.Is(err, blob.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("failed to read profile at %s: %w", payload.ProfileKey, err)
	}
	defer closers.Close(ctx, newProfileReader)

	basePath := path.Join(payload.App, "default.pgo")
	baseProfileReader, err := w.blobs.NewReader(ctx, basePath)
	switch {
	case errors.Is(err, blob.ErrNotExist):
		log.DebugContext(ctx, "app has no base profile, upload will be used")
		break
	case err != nil:
		return fmt.Errorf("failed to read profile at %s: %w", basePath, err)
	case baseProfileReader != nil:
		defer closers.Close(ctx, baseProfileReader)
	}

	newProfile, err := profile.Parse(newProfileReader)
	if err != nil {
		return fmt.Errorf("failed to parse profile at %s: %w", payload.ProfileKey, err)
	}

	profiles := []*profile.Profile{newProfile}
	if baseProfileReader != nil {
		baseProfile, err := profile.Parse(baseProfileReader)
		if err != nil {
			return fmt.Errorf("failed to parse profile at %s: %w", basePath, err)
		}

		profiles = append(profiles, baseProfile)
	}

	log.DebugContext(ctx, "merging upload with base profile")
	merged, err := profile.Merge(profiles)
	if err != nil {
		return fmt.Errorf("failed to merge profiles %s and %s: %w", payload.ProfileKey, basePath, err)
	}

	profileWriter, err := w.blobs.NewWriter(ctx, basePath)
	if err != nil {
		return fmt.Errorf("failed to open writer: %w", err)
	}

	if err = merged.Write(profileWriter); err != nil {
		return fmt.Errorf("failed to write merged profile: %w", err)
	}

	if err = profileWriter.Close(); err != nil {
		return fmt.Errorf("failed to write merged profile: %w", err)
	}

	return w.writer.Write(ctx, MergedEvent{
		App:        payload.App,
		ProfileKey: payload.ProfileKey,
		MergedKey:  basePath,
	})
}

func (w *Worker) handleEventTypeMerged(ctx context.Context, evt event.Envelope) error {
	payload, err := event.Unmarshal[MergedEvent](evt)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	err = w.blobs.Delete(ctx, payload.ProfileKey)
	switch {
	case errors.Is(err, blob.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("failed to delete profile at %s: %w", payload.ProfileKey, err)
	default:
		return nil
	}
}
