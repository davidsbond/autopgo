package target_test

import (
	"context"
	"testing"

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
