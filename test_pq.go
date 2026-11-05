package main

import (
	"fmt"
	"log"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

func main() {
	logger := watermill.NewStdLogger(false, false)
	pubsub := gochannel.NewGoChannel(gochannel.Config{}, logger)
	pqMiddleware, err := middleware.PoisonQueue(pubsub, "topic.dlq")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%T\n", pqMiddleware)
}
