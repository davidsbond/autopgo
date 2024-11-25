package target_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/davidsbond/autopgo/internal/target"
	"github.com/davidsbond/autopgo/internal/testutil"
)

func TestConsulSource_List(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name         string
		App          string
		Expected     []target.Target
		ExpectsError bool
		Handler      http.Handler
	}{
		{
			Name: "success",
			App:  "test",
			Expected: []target.Target{
				{
					Address: "https://127.0.0.1:8080",
					Path:    "/test/app",
				},
			},
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.EqualValues(t, http.MethodGet, r.Method)
				encoder := json.NewEncoder(w)

				if r.URL.Path == "/v1/catalog/services" {
					require.NoError(t, encoder.Encode(map[string][]string{
						"test": {
							"autopgo.scrape=true",
							"autopgo.scrape.app=test",
							"autopgo.scrape.scheme=https",
							"autopgo.scrape.path=/test/app",
						},
					}))
				}

				if r.URL.Path == "/v1/catalog/service/test" {
					require.NoError(t, encoder.Encode([]*api.CatalogService{
						{
							ServiceAddress: "127.0.0.1",
							ServiceTags: []string{
								"autopgo.scrape=true",
								"autopgo.scrape.app=test",
								"autopgo.scrape.scheme=https",
								"autopgo.scrape.path=/test/app",
							},
							ServicePort: 8080,
						},
					}))
				}
			}),
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()
			svr := httptest.NewServer(tc.Handler)
			t.Cleanup(svr.Close)

			client, err := api.NewClient(&api.Config{Address: svr.URL})
			require.NoError(t, err)

			source := target.NewConsulSource(client, tc.App)

			actual, err := source.List(ctx)
			if tc.ExpectsError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func TestConsulSource_List_Integration(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	client := testutil.ConsulContainer(t)
	expected := testutil.ConsulTarget(t, client)

	source := target.NewConsulSource(client, "test")
	results, err := source.List(ctx)
	require.NoError(t, err)

	if assert.Len(t, results, 1) {
		actual := results[0]
		assert.Equal(t, expected, actual)
	}
}
