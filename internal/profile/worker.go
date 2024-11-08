package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"regexp"

	"github.com/google/pprof/profile"

	"github.com/davidsbond/autopgo/internal/blob"
	"github.com/davidsbond/autopgo/internal/closers"
	"github.com/davidsbond/autopgo/internal/event"
	"github.com/davidsbond/autopgo/internal/logger"
)

type (
	// The Worker type is used to handle profile events and merge uploaded profiles together into a single
	// base profile.
	Worker struct {
		writer  EventWriter
		blobs   BlobRepository
		pruning []PruneConfig
	}

	// The PruneConfig type represents a collection of pruning rules for a specific application.
	PruneConfig struct {
		// The application whose profiles should be pruned.
		App string `json:"app"`
		// The pruning rules to apply.
		Rules []PruneRule `json:"rules"`
	}

	// The PruneRule type represents a single pruning action to perform on a profile.
	PruneRule struct {
		// Drop determines the node under which child nodes will be pruned.
		Drop *regexp.Regexp `json:"drop"`
		// Keep determines nodes under Drop that should be kept.
		Keep *regexp.Regexp `json:"keep"`
	}
)

// NewWorker returns a new instance of the Worker type that will read and write profile data via the BlobRepository
// implementation, read events via the EventReader implementation and publish events via the EventWriter implementation.
func NewWorker(blobs BlobRepository, writer EventWriter, prune []PruneConfig) *Worker {
	return &Worker{
		blobs:   blobs,
		writer:  writer,
		pruning: prune,
	}
}

// HandleEvent is an event.Handler implementation that is used to handle inbound profile events and perform profile
// merge and deletion.
func (w *Worker) HandleEvent(ctx context.Context, evt event.Envelope) error {
	switch evt.Type {
	case EventTypeUploaded:
		return w.handleEventTypeUploaded(ctx, evt)
	case EventTypeMerged:
		return w.handleEventTypeMerged(ctx, evt)
	case EventTypeDeleted:
		return w.handleEventTypeDeleted(ctx, evt)
	default:
		return nil
	}
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

		log.DebugContext(ctx, "merging upload with base profile")
		profiles = append(profiles, baseProfile)
	}

	merged, err := profile.Merge(profiles)
	if err != nil {
		return fmt.Errorf("failed to merge profiles %s and %s: %w", payload.ProfileKey, basePath, err)
	}

	for _, prune := range w.pruning {
		if prune.App != payload.App {
			continue
		}

		for _, rule := range prune.Rules {
			attrs := make([]any, 0)
			if rule.Drop != nil {
				attrs = append(attrs, slog.String("prune.drop", rule.Drop.String()))
			}
			if rule.Keep != nil {
				attrs = append(attrs, slog.String("prune.keep", rule.Keep.String()))
			}

			log.With(attrs...).DebugContext(ctx, "pruning profile")

			merged.Prune(rule.Drop, rule.Keep)
		}

		break
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

func (w *Worker) handleEventTypeDeleted(ctx context.Context, evt event.Envelope) error {
	payload, err := event.Unmarshal[DeletedEvent](evt)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	for object, err := range w.blobs.List(ctx, IsApplication(payload.App)) {
		if err != nil {
			return err
		}

		err = w.blobs.Delete(ctx, object.Key)
		switch {
		case errors.Is(err, blob.ErrNotExist):
			continue
		case err != nil:
			return err
		}
	}

	return nil
}

// LoadPruneConfig attempts to parse the file at the specified location and decode it into an array of profile pruning
// rules that are applied when the worker merges profiles. The file is expected to be in JSON encoding.
func LoadPruneConfig(ctx context.Context, location string) ([]PruneConfig, error) {
	if location == "" {
		return nil, nil
	}

	f, err := os.Open(location)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		defer closers.Close(ctx, f)
	}

	var pruning []PruneConfig
	if err = json.NewDecoder(f).Decode(&pruning); err != nil {
		return nil, err
	}

	return pruning, nil
}
