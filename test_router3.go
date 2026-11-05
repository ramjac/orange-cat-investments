package main

import (
	"context"
	"fmt"
	"time"
    "errors"
    "strings"

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

    pqRouter, err := middleware.PoisonQueueWithFilter(pubsub, "fallback.dlq", func(err error) bool {
        // By returning true, we send the message to DLQ
        return true
    })
    // Wait, the PoisonQueue needs to know which topic to publish to.
    // If it's statically "fallback.dlq", that's not `<topic>.dlq`.
    // Wait, Watermill's PoisonQueue uses the fixed topic you give it.
    // Let's see if there's a way to dynamically publish to `<original_topic>.dlq`.
    // Let's implement our own PoisonQueue.
    fmt.Printf("%T\n", pqRouter)
}
