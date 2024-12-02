package target

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/davidsbond/autopgo/internal/closers"
	"github.com/davidsbond/autopgo/internal/logger"
)

type (
	// The FileSource type describes a source of targets loaded from a JSON file.
	FileSource struct {
		location string
		mux      sync.RWMutex
		targets  []Target
	}
)

// NewFileSource returns a new instance of the FileSource type that loads Target data from the JSON file specified
// at the given location. The file is expected to contain a JSON-encoded array of the Target type. The FileSource type
// will also listen for SIGHUP signals and reread the JSON file and update its list of targets.
func NewFileSource(ctx context.Context, location string) (*FileSource, error) {
	targets, err := readTargetsFromFile(ctx, location)
	if err != nil {
		return nil, err
	}

	source := &FileSource{
		targets:  targets,
		location: location,
	}

	go source.handleUpdates(ctx)
	return source, nil
}

// List all targets within the file.
func (fs *FileSource) List(_ context.Context) ([]Target, error) {
	fs.mux.RLock()
	defer fs.mux.RUnlock()

	return fs.targets, nil
}

func (fs *FileSource) handleUpdates(ctx context.Context) {
	update := make(chan os.Signal, 1)
	defer close(update)

	signal.Notify(update, syscall.SIGHUP)
	defer signal.Stop(update)

	for {
		select {
		case <-ctx.Done():
			return
		case <-update:
			log := logger.FromContext(ctx).With(slog.String("file", fs.location))

			targets, err := readTargetsFromFile(ctx, fs.location)
			if err != nil {
				log.With(slog.String("error", err.Error())).Error("failed to read updated targets")

				continue
			}

			fs.mux.Lock()
			fs.targets = targets
			fs.mux.Unlock()

			log.Debug("targets updated")
		}
	}
}

func readTargetsFromFile(ctx context.Context, location string) ([]Target, error) {
	f, err := os.Open(location)
	if err != nil {
		return nil, err
	}

	defer closers.Close(ctx, f)
	var targets []Target
	if err = json.NewDecoder(f).Decode(&targets); err != nil {
		return nil, err
	}

	return targets, nil
}

// Check performs os.Stat on the desired config file. This method is used to implement the operation.Checker interface
// for use in health checks.
func (fs *FileSource) Check(_ context.Context) error {
	_, err := os.Stat(fs.location)
	return err
}

// Name returns "file://<config location>". This method is used to implement the operation.Checker interface for use in
// health checks.
func (fs *FileSource) Name() string {
	return "file://" + fs.location
}
