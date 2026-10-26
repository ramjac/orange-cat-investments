package envelope

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

// PubSub combines Watermill Publisher and Subscriber interfaces.
type PubSub interface {
	message.Publisher
	message.Subscriber
}

// NewPublisherWithFallback returns a durable AMQP publisher when amqpURL is configured and reachable,
// or falls back to an in-memory GoChannel publisher.
func NewPublisherWithFallback(amqpURL string, logger *slog.Logger) message.Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	wmLogger := watermill.NewStdLogger(false, false)
	if amqpURL != "" {
		amqpConfig := amqp.NewDurableQueueConfig(amqpURL)
		pub, err := amqp.NewPublisher(amqpConfig, wmLogger)
		if err == nil {
			logger.Info("connected to RabbitMQ event bus", "url", amqpURL)
			return pub
		}
		logger.Warn("failed to connect to RabbitMQ, falling back to GoChannel", "error", err)
	}
	return gochannel.NewGoChannel(gochannel.Config{}, wmLogger)
}

// NewPubSubWithFallback returns a durable AMQP PubSub when amqpURL is configured and reachable,
// or falls back to an in-memory GoChannel.
func NewPubSubWithFallback(amqpURL string, logger *slog.Logger) (message.Publisher, message.Subscriber) {
	if logger == nil {
		logger = slog.Default()
	}
	wmLogger := watermill.NewStdLogger(false, false)
	if amqpURL != "" {
		amqpConfig := amqp.NewDurableQueueConfig(amqpURL)
		pub, pubErr := amqp.NewPublisher(amqpConfig, wmLogger)
		sub, subErr := amqp.NewSubscriber(amqpConfig, wmLogger)
		if pubErr == nil && subErr == nil {
			logger.Info("connected worker to RabbitMQ event bus", "url", amqpURL)
			return pub, sub
		}
		logger.Warn("failed to connect to RabbitMQ, falling back to GoChannel", "pub_err", pubErr, "sub_err", subErr)
	}
	ch := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, wmLogger)
	return ch, ch
}
