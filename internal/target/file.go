package target

import (
	"context"
	"encoding/json"
	"os"

	"github.com/davidsbond/autopgo/internal/closers"
)

type (
	// The FileSource type describes a source of targets loaded from a JSON file.
	FileSource struct {
		targets []Target
	}
)

// NewFileSource returns a new instance of the FileSource type that loads Target data from the JSON file specified
// at the given location. The file is expected to contain a JSON-encoded array of the Target type.
func NewFileSource(ctx context.Context, location string) (*FileSource, error) {
	f, err := os.Open(location)
	if err != nil {
		return nil, err
	}

	defer closers.Close(ctx, f)
	var targets []Target
	if err = json.NewDecoder(f).Decode(&targets); err != nil {
		return nil, err
	}

	return &FileSource{targets: targets}, nil
}

// List all targets within the file.
func (fs *FileSource) List(_ context.Context) ([]Target, error) {
	return fs.targets, nil
}
