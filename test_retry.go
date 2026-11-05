package main

import (
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

func main() {
    retryMiddleware := middleware.Retry{
		MaxRetries:      3,
		InitialInterval: time.Millisecond * 10,
		MaxInterval:     time.Millisecond * 100,
		Multiplier:      2,
		Logger:          watermill.NewStdLogger(false, false),
	}
    fmt.Printf("%T\n", retryMiddleware)
}
