package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

func main() {
	logger := watermill.NewStdLogger(false, false)
	pubsub := gochannel.NewGoChannel(gochannel.Config{}, logger)
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}

    router.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
        return func(msg *message.Message) ([]*message.Message, error) {
            fmt.Println("Custom middleware processing message with UUID", msg.UUID)
            // Can we get topic from msg?
            return h(msg)
        }
    })

	router.AddHandler(
		"test_handler",
		"original_topic",
		pubsub,
		"target_topic",
		pubsub,
		func(msg *message.Message) ([]*message.Message, error) {
			return nil, fmt.Errorf("handler error")
		},
	)

	go router.Run(context.Background())
	<-router.Running()

	msg := message.NewMessage(watermill.NewUUID(), []byte("payload"))
	pubsub.Publish("original_topic", msg)
	time.Sleep(time.Second)
}
