package main

import (
	"context"
	"fmt"
	"time"
    "errors"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

func main() {
	logger := watermill.NewStdLogger(false, false)
	pubsub := gochannel.NewGoChannel(gochannel.Config{}, logger)
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}

    pq, err := middleware.PoisonQueue(pubsub, "original_topic.dlq")
    if err != nil { panic(err) }
    router.AddMiddleware(pq)

    router.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
        return func(msg *message.Message) ([]*message.Message, error) {
            fmt.Println("Custom validation middleware")
            return nil, errors.New("validation failed")
        }
    })

    retry := middleware.Retry{
		MaxRetries:      3,
		InitialInterval: time.Millisecond * 10,
		MaxInterval:     time.Millisecond * 100,
		Multiplier:      2,
		Logger:          watermill.NewStdLogger(false, false),
	}
    router.AddMiddleware(retry.Middleware)


	router.AddHandler(
		"test_handler",
		"original_topic",
		pubsub,
		"target_topic",
		pubsub,
		func(msg *message.Message) ([]*message.Message, error) {
            fmt.Println("Handler executed")
			return nil, fmt.Errorf("handler error")
		},
	)

    go func() {
        messages, _ := pubsub.Subscribe(context.Background(), "original_topic.dlq")
        for msg := range messages {
            fmt.Println("DLQ got message:", msg.UUID)
            msg.Ack()
        }
    }()

	go router.Run(context.Background())
	<-router.Running()

	msg := message.NewMessage(watermill.NewUUID(), []byte("payload"))
	pubsub.Publish("original_topic", msg)
	time.Sleep(time.Second)
}
