package testutil

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	mc "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/minio"

	"github.com/davidsbond/autopgo/internal/blob"
)

func MinioContainer(t *testing.T) *blob.Bucket {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	container, err := minio.Run(ctx, "minio/minio:RELEASE.2024-01-16T16-07-38Z")
	require.NoError(t, err)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		require.NoError(t, container.Terminate(ctx))
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "9000")
	require.NoError(t, err)

	fullHost := net.JoinHostPort(host, port.Port())
	endpoint := url.URL{
		Scheme: "http",
		Host:   fullHost,
	}

	minioClient, err := mc.New(fullHost, &mc.Options{
		Creds: credentials.NewStaticV4("minioadmin", "minioadmin", ""),
	})
	require.NoError(t, err)
	require.NoError(t, minioClient.MakeBucket(ctx, "default", mc.MakeBucketOptions{}))

	params := url.Values{
		"use_path_style": []string{"true"},
		"disable_https":  []string{"true"},
		"endpoint":       []string{endpoint.String()},
	}

	blobURL := url.URL{
		Scheme:   "s3",
		Host:     "default",
		RawQuery: params.Encode(),
	}

	t.Setenv("AWS_REGION", "dev")
	t.Setenv("AWS_ACCESS_KEY_ID", "minioadmin")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "minioadmin")

	bucket, err := blob.NewBucket(ctx, blobURL.String())
	require.NoError(t, err)
	return bucket
}
