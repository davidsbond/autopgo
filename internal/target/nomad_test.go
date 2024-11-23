package target_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/davidsbond/autopgo/internal/target"
)

func TestNomadSource_List(t *testing.T) {
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

				if r.URL.Path == "/v1/services" {
					require.NoError(t, encoder.Encode([]*api.ServiceRegistrationListStub{
						{
							Namespace: "default",
							Services: []*api.ServiceRegistrationStub{
								{
									ServiceName: "test",
									Tags: []string{
										"autopgo.scrape=true",
										"autopgo.scrape.app=test",
										"autopgo.scrape.scheme=https",
										"autopgo.scrape.path=/test/app",
									},
								},
							},
						},
					}))

					return
				}

				if r.URL.Path == "/v1/service/test" {
					require.NoError(t, encoder.Encode([]*api.ServiceRegistration{
						{
							ServiceName: "test",
							Namespace:   "test",
							Tags: []string{
								"autopgo.scrape=true",
								"autopgo.scrape.app=test",
								"autopgo.scrape.scheme=https",
								"autopgo.scrape.path=/test/app",
							},
							Address: "127.0.0.1",
							Port:    8080,
						},
					}))

					return
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

			source := target.NewNomadSource(client, tc.App)

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
