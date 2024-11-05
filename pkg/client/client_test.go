package client_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/davidsbond/autopgo/internal/api"
	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/pkg/client"
)

func TestClient_Upload(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name         string
		App          string
		Data         []byte
		Setup        func(t *testing.T) http.Handler
		ExpectsError bool
	}{
		{
			Name: "successful upload",
			App:  "test",
			Data: []byte("test"),
			Setup: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.EqualValues(t, http.MethodPost, r.Method)
					assert.EqualValues(t, "/api/profile/test", r.URL.Path)

					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					assert.EqualValues(t, []byte("test"), body)
					api.Respond(r.Context(), w, http.StatusCreated, profile.UploadResponse{
						Key: "test",
					})
				})
			},
		},
		{
			Name:         "returns errors",
			App:          "test",
			Data:         []byte("test"),
			ExpectsError: true,
			Setup: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					api.ErrorResponse(r.Context(), w, "uh oh", http.StatusInternalServerError)
				})
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			handler := tc.Setup(t)
			server := httptest.NewServer(handler)
			defer server.Close()

			cl := client.New(server.URL)
			err := cl.Upload(context.Background(), tc.App, bytes.NewReader(tc.Data))
			if tc.ExpectsError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_Download(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name         string
		App          string
		Expected     []byte
		Setup        func(t *testing.T) http.Handler
		ExpectsError bool
	}{
		{
			Name:     "successful download",
			App:      "test",
			Expected: []byte("test"),
			Setup: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.EqualValues(t, http.MethodGet, r.Method)
					assert.EqualValues(t, "/api/profile/test", r.URL.Path)

					_, err := io.Copy(w, bytes.NewReader([]byte("test")))
					require.NoError(t, err)
				})
			},
		},
		{
			Name:         "profile not found",
			App:          "test",
			ExpectsError: true,
			Setup: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					api.ErrorResponse(r.Context(), w, "uh oh", http.StatusNotFound)
				})
			},
		},
		{
			Name:         "returns errors",
			App:          "test",
			ExpectsError: true,
			Setup: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					api.ErrorResponse(r.Context(), w, "uh oh", http.StatusInternalServerError)
				})
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			handler := tc.Setup(t)
			server := httptest.NewServer(handler)
			defer server.Close()

			cl := client.New(server.URL)
			data := bytes.NewBuffer(nil)
			err := cl.Download(context.Background(), tc.App, data)
			if tc.ExpectsError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.EqualValues(t, tc.Expected, data.Bytes())
		})
	}
}

func TestClient_List(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name         string
		Setup        func(t *testing.T) http.Handler
		ExpectsError bool
		Expected     []profile.Profile
	}{
		{
			Name: "successful list",
			Expected: []profile.Profile{
				{
					Key:          "test",
					Size:         1000,
					LastModified: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			Setup: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.EqualValues(t, http.MethodGet, r.Method)
					assert.EqualValues(t, "/api/profile", r.URL.Path)

					api.Respond(r.Context(), w, http.StatusOK, profile.ListResponse{
						Profiles: []profile.Profile{
							{
								Key:          "test",
								Size:         1000,
								LastModified: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
							},
						},
					})
				})
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			handler := tc.Setup(t)
			server := httptest.NewServer(handler)
			defer server.Close()

			cl := client.New(server.URL)
			actual, err := cl.List(context.Background())
			if tc.ExpectsError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func TestClient_Delete(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name         string
		App          string
		Setup        func(t *testing.T) http.Handler
		ExpectsError bool
	}{
		{
			Name: "successful delete",
			App:  "test",
			Setup: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.EqualValues(t, http.MethodDelete, r.Method)
					assert.EqualValues(t, "/api/profile/test", r.URL.Path)
				})
			},
		},
		{
			Name:         "profile not found",
			App:          "test",
			ExpectsError: true,
			Setup: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					api.ErrorResponse(r.Context(), w, "uh oh", http.StatusNotFound)
				})
			},
		},
		{
			Name:         "returns errors",
			App:          "test",
			ExpectsError: true,
			Setup: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					api.ErrorResponse(r.Context(), w, "uh oh", http.StatusInternalServerError)
				})
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			handler := tc.Setup(t)
			server := httptest.NewServer(handler)
			defer server.Close()

			cl := client.New(server.URL)
			err := cl.Delete(context.Background(), tc.App)
			if tc.ExpectsError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
