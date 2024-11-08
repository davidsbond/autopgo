package profile_test

import (
	"bytes"
	"context"
	"regexp"
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
		Pruning      []profile.PruneConfig
	}{
		{
			Name: "handle profile.uploaded with base profile, no pruning",
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
			Name: "handle profile.uploaded with no base profile, no pruning",
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
		{
			Name: "handle profile.uploaded with base profile, with pruning",
			Event: event.Envelope{
				ID:        uuid.NewString(),
				Timestamp: time.Now(),
				Type:      profile.EventTypeUploaded,
				Payload: mustMarshal(t, profile.UploadedEvent{
					App:        "test-app",
					ProfileKey: "test-app/staging/12345",
				}),
			},
			Pruning: []profile.PruneConfig{
				{
					App: "test-app",
					Rules: []profile.PruneRule{
						{
							Drop: regexp.MustCompile(`github\.com/aws/aws-sdk-go.*`),
						},
					},
				},
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
			Name: "handle profile.deleted",
			Event: event.Envelope{
				ID:        uuid.NewString(),
				Timestamp: time.Now(),
				Type:      profile.EventTypeDeleted,
				Payload: mustMarshal(t, profile.DeletedEvent{
					App: "test-app",
				}),
			},
			Setup: func(blobs *mocks.MockBlobRepository, events *mocks.MockEventWriter) {
				blobs.EXPECT().
					List(mock.Anything, mock.Anything).
					Return(func(yield func(blob.Object, error) bool) {
						yield(blob.Object{
							Key:          "test-app/default.pgo",
							Size:         1000,
							LastModified: time.Now(),
						}, nil)
					})

				blobs.EXPECT().
					Delete(mock.Anything, "test-app/default.pgo").
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

			err := profile.NewWorker(blobs, events, tc.Pruning).HandleEvent(context.Background(), tc.Event)
			if tc.ExpectsError {
				require.Error(t, err)
			}

			require.NoError(t, err)
		})
	}
}
