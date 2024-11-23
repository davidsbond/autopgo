package target_test

import (
	"context"
	"encoding/json"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/davidsbond/autopgo/internal/target"
)

func TestFileSource_List(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name         string
		Location     string
		ExpectsError bool
		Expected     []target.Target
	}{
		{
			Name:     "success",
			Location: "testdata/targets.json",
			Expected: []target.Target{
				{
					Address: "http://localhost:8080",
					Path:    "/debug/pprof/profile",
				},
				{
					Address: "http://localhost:8081",
					Path:    "/debug/pprof/profile",
				},
				{
					Address: "http://localhost:8082",
					Path:    "/debug/pprof/profile",
				},
			},
		},
		{
			Name:         "no file",
			Location:     "testdata/nope.json",
			ExpectsError: true,
		},
		{
			Name:         "invalid file",
			Location:     "testdata/targets.invalid.json",
			ExpectsError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			source, err := target.NewFileSource(ctx, tc.Location)
			if tc.ExpectsError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			actual, err := source.List(ctx)
			require.NoError(t, err)
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func TestFileSource_SIGHUP(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	f, err := os.CreateTemp(os.TempDir(), "autopgo")
	require.NoError(t, err)

	// Create a JSON file with a single target initially.
	targets := []target.Target{
		{
			Address: "https://test.com:8080",
			Path:    "/test",
		},
	}

	require.NoError(t, json.NewEncoder(f).Encode(targets))
	require.NoError(t, f.Close())

	source, err := target.NewFileSource(ctx, f.Name())
	require.NoError(t, err)

	// Make sure we only have the single target.
	actual, err := source.List(ctx)
	require.NoError(t, err)
	require.EqualValues(t, targets, actual)

	// Rewrite the file with a second target
	targets = append(targets, target.Target{
		Address: "https://test.com:8081",
		Path:    "/test2",
	})

	f, err = os.Create(f.Name())
	require.NoError(t, err)
	require.NoError(t, json.NewEncoder(f).Encode(targets))
	require.NoError(t, f.Close())

	// Send a SIGHUP to force the target source to reload.
	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGHUP))

	// Make sure the list now has both targets. This is done async so we should check periodically for the updated
	// list.
	assert.Eventually(t, func() bool {
		actual, err = source.List(ctx)
		require.NoError(t, err)
		return len(targets) == len(actual)
	}, time.Minute, time.Second)

	require.EqualValues(t, targets, actual)
}
