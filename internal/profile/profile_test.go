package profile_test

import (
	"testing"
	"time"

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

func TestIsOlderThan(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name     string
		Duration time.Duration
		Object   blob.Object
		Expected bool
	}{
		{
			Name:     "should return true if the object is old enough",
			Expected: true,
			Duration: time.Minute,
			Object: blob.Object{
				LastModified: time.Now().Add(-time.Hour),
			},
		},
		{
			Name:     "should return false if the object is not old enough",
			Expected: false,
			Duration: time.Hour,
			Object: blob.Object{
				LastModified: time.Now(),
			},
		},
		{
			Name:     "should return false for a zero duration",
			Expected: false,
			Duration: 0,
			Object: blob.Object{
				LastModified: time.Now(),
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			filter := profile.IsOlderThan(tc.Duration)
			actual := filter(tc.Object)
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func TestIsLargerThan(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name     string
		Size     int64
		Object   blob.Object
		Expected bool
	}{
		{
			Name:     "should return true if the object is large enough",
			Expected: true,
			Size:     10,
			Object: blob.Object{
				Size: 20,
			},
		},
		{
			Name:     "should return false if the object is not large enough",
			Expected: false,
			Size:     10,
			Object: blob.Object{
				Size: 5,
			},
		},
		{
			Name:     "should return false for a zero size",
			Expected: false,
			Size:     0,
			Object: blob.Object{
				Size: 10,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			filter := profile.IsLargerThan(tc.Size)
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
