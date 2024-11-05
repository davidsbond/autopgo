package profile_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/davidsbond/autopgo/internal/blob"
	"github.com/davidsbond/autopgo/internal/profile"
)

func TestIsMergedProfile(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name     string
		Object   blob.Object
		Expected bool
	}{
		{
			Name:     "should return true for merged profiles",
			Expected: true,
			Object: blob.Object{
				Key: "test/default.pgo",
			},
		},
		{
			Name:     "should return false for non-merged profiles",
			Expected: false,
			Object: blob.Object{
				Key: "test/staging/010101010",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			filter := profile.IsMergedProfile()
			actual := filter(tc.Object)
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func TestIsApplication(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name     string
		App      string
		Object   blob.Object
		Expected bool
	}{
		{
			Name:     "should return true for the correct application name",
			Expected: true,
			App:      "test",
			Object: blob.Object{
				Key: "test/default.pgo",
			},
		},
		{
			Name:     "should return false for for an incorrect application name",
			App:      "test-1",
			Expected: false,
			Object: blob.Object{
				Key: "test/staging/010101010",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			filter := profile.IsApplication(tc.App)
			actual := filter(tc.Object)
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func TestIsValidAppName(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name     string
		App      string
		Expected bool
	}{
		{
			Name:     "should return true if the app name is valid",
			App:      "test-app-123",
			Expected: true,
		},
		{
			Name:     "should return false if the app name is not valid",
			App:      "not/allowed/here",
			Expected: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			actual := profile.IsValidAppName(tc.App)
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}
