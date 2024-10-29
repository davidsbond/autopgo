package profile_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/davidsbond/autopgo/internal/blob"
	"github.com/davidsbond/autopgo/internal/event"
	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/internal/profile/mocks"
)

func TestWorker_HandleEvent(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name         string
		Event        event.Envelope
		ExpectsError bool
		Setup        func(blobs *mocks.MockBlobRepository, events *mocks.MockEventWriter)
	}{
		{
			Name: "handle profile.uploaded with base profile",
			Event: event.Envelope{
				ID:        uuid.NewString(),
				Timestamp: time.Now(),
				Type:      profile.EventTypeUploaded,
				Payload: mustMarshal(t, profile.UploadedEvent{
					App:        "test-app",
					ProfileKey: "test-app/staging/12345",
				}),
			},
			Setup: func(blobs *mocks.MockBlobRepository, events *mocks.MockEventWriter) {
				blobs.EXPECT().
					NewReader(mock.Anything, "test-app/staging/12345").
					Return(&ReadCloser{data: bytes.NewBuffer(validProfile)}, nil)

				blobs.EXPECT().
					NewReader(mock.Anything, "test-app/default.pgo").
					Return(&ReadCloser{data: bytes.NewBuffer(validProfile)}, nil)

				blobs.EXPECT().
					NewWriter(mock.Anything, "test-app/default.pgo").
					Return(&WriteCloser{}, nil)

				events.EXPECT().
					Write(mock.Anything, profile.MergedEvent{
						App:        "test-app",
						ProfileKey: "test-app/staging/12345",
						MergedKey:  "test-app/default.pgo",
					}).
					Return(nil)
			},
		},
		{
			Name: "handle profile.uploaded with no base profile",
			Event: event.Envelope{
				ID:        uuid.NewString(),
				Timestamp: time.Now(),
				Type:      profile.EventTypeUploaded,
				Payload: mustMarshal(t, profile.UploadedEvent{
					App:        "test-app",
					ProfileKey: "test-app/staging/12345",
				}),
			},
			Setup: func(blobs *mocks.MockBlobRepository, events *mocks.MockEventWriter) {
				blobs.EXPECT().
					NewReader(mock.Anything, "test-app/staging/12345").
					Return(&ReadCloser{data: bytes.NewBuffer(validProfile)}, nil)

				blobs.EXPECT().
					NewReader(mock.Anything, "test-app/default.pgo").
					Return(nil, blob.ErrNotExist)

				blobs.EXPECT().
					NewWriter(mock.Anything, "test-app/default.pgo").
					Return(&WriteCloser{}, nil)

				events.EXPECT().
					Write(mock.Anything, profile.MergedEvent{
						App:        "test-app",
						ProfileKey: "test-app/staging/12345",
						MergedKey:  "test-app/default.pgo",
					}).
					Return(nil)
			},
		},
		{
			Name: "handle profile.merged",
			Event: event.Envelope{
				ID:        uuid.NewString(),
				Timestamp: time.Now(),
				Type:      profile.EventTypeMerged,
				Payload: mustMarshal(t, profile.MergedEvent{
					App:        "test-app",
					ProfileKey: "test-app/staging/12345",
					MergedKey:  "test-app/default.pgo",
				}),
			},
			Setup: func(blobs *mocks.MockBlobRepository, events *mocks.MockEventWriter) {
				blobs.EXPECT().
					Delete(mock.Anything, "test-app/staging/12345").
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

			err := profile.NewWorker(blobs, events).HandleEvent(context.Background(), tc.Event)
			if tc.ExpectsError {
				require.Error(t, err)
			}

			require.NoError(t, err)
		})
	}
}
