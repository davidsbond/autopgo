// Package blob provides types for interacting with blob storage providers such as S3, GCS etc.
package blob

import (
	"context"
	"errors"
	"io"
	"iter"
	"log/slog"
	"time"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
	"gocloud.dev/gcerrors"

	"github.com/davidsbond/autopgo/internal/logger"
)

type (
	// The Bucket type is used to read, write & delete objects from a blob storage provider.
	Bucket struct {
		blob *blob.Bucket
	}

	// The Filter function allows callers of Bucket.List to programmatically filter results.
	Filter func(obj Object) bool

	// The Object type contains metadata on an object within the blob store.
	Object struct {
		// The object's key.
		Key string
		// The object's size in bytes.
		Size int64
		// When the object was last modified.
		LastModified time.Time
	}
)

var (
	// ErrNotExist is the error given when performing an action against an object that does not exist.
	ErrNotExist = errors.New("does not exist")
)

// NewBucket returns a new instance of the Bucket type that performs actions against a blob storage provider. The
// provider is determined by the URL string. See the gocloud.dev documentation for information on provider specific
// URLs.
func NewBucket(ctx context.Context, bucketURL string) (*Bucket, error) {
	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		return nil, err
	}

	_, err = bucket.IsAccessible(ctx)
	return &Bucket{blob: bucket}, err
}

// Close the connection to the blob store.
func (b *Bucket) Close() error {
	return b.blob.Close()
}

// NewReader returns an io.ReadCloser implementation that reads data from blob storage at a given path. Returns
// ErrNotExist if there is no object at the specified path. The reader must be closed by the caller.
func (b *Bucket) NewReader(ctx context.Context, path string) (io.ReadCloser, error) {
	logger.FromContext(ctx).
		With(slog.String("path", path)).
		DebugContext(ctx, "reading object")

	reader, err := b.blob.NewReader(ctx, path, &blob.ReaderOptions{})
	switch {
	case gcerrors.Code(err) == gcerrors.NotFound:
		return nil, ErrNotExist
	case err != nil:
		return nil, err
	default:
		return reader, nil
	}
}

// NewWriter returns an io.WriteCloser implementation that writes data to the blob store at a given path. The
// writer must by closed by the caller.
func (b *Bucket) NewWriter(ctx context.Context, path string) (io.WriteCloser, error) {
	logger.FromContext(ctx).
		With(slog.String("path", path)).
		DebugContext(ctx, "writing object")

	return b.blob.NewWriter(ctx, path, &blob.WriterOptions{})
}

// Delete an object at a specified path. Returns ErrNotExist if the object does not exist.
func (b *Bucket) Delete(ctx context.Context, path string) error {
	logger.FromContext(ctx).
		With(slog.String("path", path)).
		DebugContext(ctx, "deleting object")

	err := b.blob.Delete(ctx, path)
	switch {
	case gcerrors.Code(err) == gcerrors.NotFound:
		return ErrNotExist
	case err != nil:
		return err
	default:
		return nil
	}
}

// List objects within the bucket that match the given Filter. Provide a nil Filter to return all objects. This
// method returns an iterator so is used with a range statement. The second range parameter is an error that must be
// checked on each iteration.
func (b *Bucket) List(ctx context.Context, filter Filter) iter.Seq2[Object, error] {
	iterator := b.blob.List(&blob.ListOptions{})

	return func(yield func(Object, error) bool) {
		for {
			item, err := iterator.Next(ctx)
			switch {
			case errors.Is(err, io.EOF):
				return
			case err != nil:
				if !yield(Object{}, err) {
					return
				}
			}

			if item.IsDir {
				continue
			}

			obj := Object{
				Key:          item.Key,
				Size:         item.Size,
				LastModified: item.ModTime,
			}

			if filter != nil && !filter(obj) {
				continue
			}

			if !yield(obj, nil) {
				return
			}
		}
	}
}

// Exists returns true if an object exists at the given path.
func (b *Bucket) Exists(ctx context.Context, path string) (bool, error) {
	return b.blob.Exists(ctx, path)
}
