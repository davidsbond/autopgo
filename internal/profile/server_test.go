package profile_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/davidsbond/autopgo/internal/blob"
	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/internal/profile/mocks"
)

func TestHTTPController_Upload(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name           string
		App            string
		Profile        []byte
		Setup          func(blobs *mocks.MockBlobRepository, events *mocks.MockEventWriter)
		ExpectedStatus int
	}{
		{
			Name:           "invalid app name",
			App:            "// not a valid string",
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "invalid profile",
			App:            "test-app",
			ExpectedStatus: http.StatusBadRequest,
			Profile:        []byte("invalid profile"),
		},
		{
			Name:           "error opening writer",
			App:            "test-app",
			ExpectedStatus: http.StatusInternalServerError,
			Profile:        validProfile,
			Setup: func(blobs *mocks.MockBlobRepository, events *mocks.MockEventWriter) {
				blobs.EXPECT().
					NewWriter(mock.Anything, appKeyMatcher("test-app")).
					Return(nil, io.EOF)
			},
		},
		{
			Name:           "error publishing event",
			App:            "test-app",
			ExpectedStatus: http.StatusInternalServerError,
			Profile:        validProfile,
			Setup: func(blobs *mocks.MockBlobRepository, events *mocks.MockEventWriter) {
				blobs.EXPECT().
					NewWriter(mock.Anything, appKeyMatcher("test-app")).
					Return(&WriteCloser{}, nil)

				events.EXPECT().
					Write(mock.Anything, uploadedEventMatcher("test-app")).
					Return(io.EOF)
			},
		},
		{
			Name:           "error closing writer",
			App:            "test-app",
			ExpectedStatus: http.StatusInternalServerError,
			Profile:        validProfile,
			Setup: func(blobs *mocks.MockBlobRepository, events *mocks.MockEventWriter) {
				blobs.EXPECT().
					NewWriter(mock.Anything, appKeyMatcher("test-app")).
					Return(&WriteCloser{closeError: io.EOF}, nil)
			},
		},
		{
			Name:           "error on profile write",
			App:            "test-app",
			ExpectedStatus: http.StatusInternalServerError,
			Profile:        validProfile,
			Setup: func(blobs *mocks.MockBlobRepository, events *mocks.MockEventWriter) {
				blobs.EXPECT().
					NewWriter(mock.Anything, appKeyMatcher("test-app")).
					Return(&WriteCloser{writeError: io.EOF}, nil)
			},
		},
		{
			Name:           "success",
			App:            "test-app",
			ExpectedStatus: http.StatusOK,
			Profile:        validProfile,
			Setup: func(blobs *mocks.MockBlobRepository, events *mocks.MockEventWriter) {
				blobs.EXPECT().
					NewWriter(mock.Anything, appKeyMatcher("test-app")).
					Return(&WriteCloser{}, nil)

				events.EXPECT().
					Write(mock.Anything, uploadedEventMatcher("test-app")).
					Return(nil)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			blobs := mocks.NewMockBlobRepository(t)
			events := mocks.NewMockEventWriter(t)

			if tc.Setup != nil {
				tc.Setup(blobs, events)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(tc.Profile))
			r.SetPathValue("app", tc.App)

			profile.NewHTTPController(blobs, events).Upload(w, r)

			assert.EqualValues(t, tc.ExpectedStatus, w.Code)
		})
	}
}

func TestHTTPController_Download(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name            string
		App             string
		ExpectedStatus  int
		ExpectedProfile []byte
		Setup           func(blobs *mocks.MockBlobRepository)
	}{
		{
			Name:           "invalid app name",
			App:            "// not a valid string",
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:            "success",
			App:             "test-app",
			ExpectedStatus:  http.StatusOK,
			ExpectedProfile: validProfile,
			Setup: func(blobs *mocks.MockBlobRepository) {
				blobs.EXPECT().
					NewReader(mock.Anything, appKeyMatcher("test-app")).
					Return(&ReadCloser{data: bytes.NewBuffer(validProfile)}, nil)
			},
		},
		{
			Name:           "profile does not exist",
			App:            "test-app",
			ExpectedStatus: http.StatusNotFound,
			Setup: func(blobs *mocks.MockBlobRepository) {
				blobs.EXPECT().
					NewReader(mock.Anything, appKeyMatcher("test-app")).
					Return(nil, blob.ErrNotExist)
			},
		},
		{
			Name:           "error opening writer",
			App:            "test-app",
			ExpectedStatus: http.StatusInternalServerError,
			Setup: func(blobs *mocks.MockBlobRepository) {
				blobs.EXPECT().
					NewReader(mock.Anything, appKeyMatcher("test-app")).
					Return(nil, io.EOF)
			},
		},
		{
			Name:           "error on profile read",
			App:            "test-app",
			ExpectedStatus: http.StatusInternalServerError,
			Setup: func(blobs *mocks.MockBlobRepository) {
				blobs.EXPECT().
					NewReader(mock.Anything, appKeyMatcher("test-app")).
					Return(&ReadCloser{readError: io.ErrClosedPipe}, nil)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			blobs := mocks.NewMockBlobRepository(t)
			if tc.Setup != nil {
				tc.Setup(blobs)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.SetPathValue("app", tc.App)

			profile.NewHTTPController(blobs, nil).Download(w, r)

			assert.Equal(t, tc.ExpectedStatus, w.Code)
			if tc.ExpectedProfile != nil {
				assert.Equal(t, tc.ExpectedProfile, w.Body.Bytes())
			}
		})
	}
}
