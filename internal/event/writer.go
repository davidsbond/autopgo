package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/davidsbond/autopgo/internal/logger"

	"cloud.google.com/go/pubsub/apiv1/pubsubpb"
	servicebus "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/IBM/sarama"
	snstypesv2 "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"gocloud.dev/pubsub"
)

type (
	// The Writer type is used to publish messages onto an event bus.
	Writer struct {
		events *pubsub.Topic
	}
)

// NewWriter returns a new instance of the Writer type that will publish events to the bus described in its URL string.
// See the gocloud.dev documentation for more information on provider specific urls.
func NewWriter(ctx context.Context, url string) (*Writer, error) {
	topic, err := pubsub.OpenTopic(ctx, url)
	if err != nil {
		return nil, err
	}

	return &Writer{events: topic}, nil
}

// Close the connection to the event bus.
func (w *Writer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return w.events.Shutdown(ctx)
}

// Write an event onto the bus. Messages must implement the Payload interface and are wrapped in an Envelope before
// publishing. If the event bus supports message keys/partitioning the Payload.Key method will be used to populate it.
func (w *Writer) Write(ctx context.Context, e Payload) error {
	envelope, err := wrap(e)
	if err != nil {
		return fmt.Errorf("failed to wrap event: %w", err)
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	log := logger.FromContext(ctx).With(
		slog.String("event.id", envelope.ID),
		slog.String("event.type", envelope.Type),
		slog.Time("event.timestamp", envelope.Timestamp),
	)

	log.DebugContext(ctx, "publishing event")

	return w.events.Send(ctx, &pubsub.Message{
		Body:       body,
		BeforeSend: keyFunc(e.Key()),
	})
}

func keyFunc(key string) func(asFunc func(interface{}) bool) error {
	if key == "" {
		return nil
	}

	return func(as func(interface{}) bool) error {
		pubsubMessage := &pubsubpb.PubsubMessage{}
		snsMessage := &snstypesv2.PublishBatchRequestEntry{}
		kafkaMessage := &sarama.ProducerMessage{}
		azureMessage := &servicebus.Message{}

		switch {
		case as(&pubsubMessage):
			pubsubMessage.OrderingKey = key
		case as(&snsMessage):
			snsMessage.MessageGroupId = &key
		case as(&kafkaMessage):
			kafkaMessage.Key = sarama.StringEncoder(key)
		case as(&azureMessage):
			azureMessage.PartitionKey = &key
		}

		return nil
	}
}
