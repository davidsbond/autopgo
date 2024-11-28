package blob_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/davidsbond/autopgo/internal/blob"
	"github.com/davidsbond/autopgo/internal/testutil"
)

func TestBucket_ReadWrite_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip()
		return
	}

	ctx := context.Background()
	bucket := testutil.MinioContainer(t)

	testData(t, bucket, "test-key", []byte("hello world"))

	reader, err := bucket.NewReader(ctx, "test-key")
	require.NoError(t, err)

	actual, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())

	assert.Equal(t, []byte("hello world"), actual)
}

func TestBucket_Delete_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip()
		return
	}

	ctx := context.Background()
	bucket := testutil.MinioContainer(t)

	t.Run("it should error if the key does not exist", func(t *testing.T) {
		err := bucket.Delete(ctx, "test-key")
		require.EqualValues(t, blob.ErrNotExist, err)
	})

	testData(t, bucket, "test-key", []byte("hello world"))

	t.Run("it should delete successfully", func(t *testing.T) {
		err := bucket.Delete(ctx, "test-key")
		require.NoError(t, err)

		_, err = bucket.NewReader(ctx, "test-key")
		require.EqualValues(t, blob.ErrNotExist, err)
	})
}

func TestBucket_List_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip()
		return
	}

	ctx := context.Background()
	bucket := testutil.MinioContainer(t)

	testData(t, bucket, "test-key", []byte("hello world"))

	t.Run("it should list without filters", func(t *testing.T) {
		items := make([]blob.Object, 0)
		for item, err := range bucket.List(ctx, nil) {
			require.NoError(t, err)
			items = append(items, item)
		}

		if assert.Len(t, items, 1) {
			assert.Equal(t, "test-key", items[0].Key)
		}
	})

	t.Run("it should list with filters", func(t *testing.T) {
		alwaysExclude := func(o blob.Object) bool { return false }

		items := make([]blob.Object, 0)
		for item, err := range bucket.List(ctx, alwaysExclude) {
			require.NoError(t, err)
			items = append(items, item)
		}

		assert.Len(t, items, 0)
	})
}

func TestBucket_Exists_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip()
		return
	}

	ctx := context.Background()
	bucket := testutil.MinioContainer(t)

	testData(t, bucket, "test-key", []byte("hello world"))

	t.Run("it should return true if the key exists", func(t *testing.T) {
		exists, err := bucket.Exists(ctx, "test-key")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("it should return false if the key does not exist", func(t *testing.T) {
		exists, err := bucket.Exists(ctx, "test-key-1")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func testData(t *testing.T, bucket *blob.Bucket, key string, data []byte) {
	ctx := context.Background()
	writer, err := bucket.NewWriter(ctx, key)
	require.NoError(t, err)
	_, err = io.Copy(writer, bytes.NewBuffer(data))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
}
