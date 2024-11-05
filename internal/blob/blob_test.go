package blob_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/davidsbond/autopgo/internal/blob"
)

func TestAll(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name     string
		Input    blob.Object
		Filters  []blob.Filter
		Expected bool
	}{
		{
			Name: "return true if all filters return true",
			Input: blob.Object{
				Key:          "test",
				Size:         1000,
				LastModified: time.Now(),
			},
			Expected: true,
			Filters: []blob.Filter{
				func(obj blob.Object) bool {
					return obj.Key == "test"
				},
				func(obj blob.Object) bool {
					return obj.Size == 1000
				},
				func(obj blob.Object) bool {
					return obj.LastModified.Before(time.Now())
				},
			},
		},
		{
			Name: "return false if any filters return false",
			Input: blob.Object{
				Key:          "test",
				Size:         1000,
				LastModified: time.Now(),
			},
			Expected: false,
			Filters: []blob.Filter{
				func(obj blob.Object) bool {
					return obj.Key == "test"
				},
				func(obj blob.Object) bool {
					return obj.Size == 1000
				},
				func(obj blob.Object) bool {
					return obj.LastModified.After(time.Now())
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			filter := blob.All(tc.Filters...)

			actual := filter(tc.Input)
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func TestAny(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name     string
		Input    blob.Object
		Filters  []blob.Filter
		Expected bool
	}{
		{
			Name: "return true if any filters return true",
			Input: blob.Object{
				Key:          "test",
				Size:         1000,
				LastModified: time.Now(),
			},
			Expected: true,
			Filters: []blob.Filter{
				func(obj blob.Object) bool {
					return obj.Key == "test"
				},
				func(obj blob.Object) bool {
					return obj.Size == 1000
				},
				func(obj blob.Object) bool {
					return obj.LastModified.After(time.Now())
				},
			},
		},
		{
			Name: "return false if no filters return true",
			Input: blob.Object{
				Key:          "test",
				Size:         1000,
				LastModified: time.Now(),
			},
			Expected: false,
			Filters: []blob.Filter{
				func(obj blob.Object) bool {
					return obj.Key == "test1"
				},
				func(obj blob.Object) bool {
					return obj.Size == 10001
				},
				func(obj blob.Object) bool {
					return obj.LastModified.After(time.Now())
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			filter := blob.Any(tc.Filters...)

			actual := filter(tc.Input)
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}
