package valkey

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/pkg/envelope"
)

// InvalidationListener coordinates event subscriptions to eagerly evict caches.
type InvalidationListener struct {
	cache      Cache
	logger     *slog.Logger
	subscriber message.Subscriber
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewInvalidationListener creates an invalidation listener.
func NewInvalidationListener(cache Cache, subscriber message.Subscriber, logger *slog.Logger) *InvalidationListener {
	if logger == nil {
		logger = slog.Default()
	}
	return &InvalidationListener{
		cache:      cache,
		logger:     logger,
		subscriber: subscriber,
	}
}

// Start begins consuming events and invalidating caches in the background.
func (l *InvalidationListener) Start(ctx context.Context) {
	if l.subscriber == nil {
		l.logger.Info("no subscriber configured for cache invalidation listener, skipping async event listener")
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	l.cancel = cancel

	topics := []string{
		"events.investment.trade_executed.v1",
		"events.investment.order_placed.v1",
		"events.workforce.care_schedule_updated.v1",
		"events.workforce.incident_created.v1",
	}

	for _, topic := range topics {
		messages, err := l.subscriber.Subscribe(ctx, topic)
		if err != nil {
			l.logger.Warn("failed to subscribe to cache invalidation topic", "topic", topic, "error", err)
			continue
		}

		l.wg.Add(1)
		go l.consumeTopic(ctx, topic, messages)
	}

	l.logger.Info("started event-driven cache invalidation listener across topics", "count", len(topics))
}

func (l *InvalidationListener) consumeTopic(ctx context.Context, topic string, messages <-chan *message.Message) {
	defer l.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			// Process message and guarantee Ack() to prevent queue clogging or dead-letter queue
			// flooding for non-retryable invalidation or deserialization errors.
			func() {
				defer func() {
					if r := recover(); r != nil {
						l.logger.Error("panic recovered in cache invalidation consumer", "topic", topic, "panic", r)
					}
					msg.Ack()
				}()
				l.handleMessage(ctx, topic, msg)
			}()
		}
	}
}

type genericPayload struct {
	CustomerID  string `json:"customer_id"`
	PortfolioID string `json:"portfolio_id"`
	FelineID    string `json:"feline_id"`
}

func (l *InvalidationListener) handleMessage(ctx context.Context, topic string, msg *message.Message) {
	var env envelope.EventEnvelope[json.RawMessage]
	if err := json.Unmarshal(msg.Payload, &env); err != nil {
		l.logger.Warn("invalidation listener received unparseable envelope", "topic", topic, "error", err)
		return
	}

	var payload genericPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		l.logger.Warn("invalidation listener received unparseable inner payload", "topic", topic, "error", err)
		return
	}

	switch topic {
	case "events.investment.trade_executed.v1", "events.investment.order_placed.v1":
		if payload.CustomerID != "" {
			keysToDel := []string{
				KeyPortfolio(payload.CustomerID),
				KeyOrders(payload.CustomerID),
			}
			if payload.PortfolioID != "" {
				keysToDel = append(keysToDel, KeyPortfolioWithID(payload.CustomerID, payload.PortfolioID))
			}
			if err := l.cache.Delete(ctx, keysToDel...); err != nil {
				l.logger.Warn("failed to evict customer portfolio cache on trade event", "customer_id", payload.CustomerID, "error", err)
			}
			_ = l.cache.DeletePrefix(ctx, fmt.Sprintf("cache:portfolio:%s:", payload.CustomerID))
			l.logger.Info("evicted customer portfolio cache on trade event", "customer_id", payload.CustomerID)
		} else {
			if err := l.cache.DeletePrefix(ctx, "cache:portfolio:"); err != nil {
				l.logger.Warn("failed to evict all portfolio caches on trade event", "error", err)
			} else {
				l.logger.Info("evicted all portfolio caches on trade event")
			}
		}

	case "events.workforce.care_schedule_updated.v1":
		if payload.FelineID != "" {
			if err := l.cache.Delete(ctx, KeyCareSchedule(payload.FelineID)); err != nil {
				l.logger.Warn("failed to evict care schedule cache on workforce event", "feline_id", payload.FelineID, "error", err)
			} else {
				l.logger.Info("evicted care schedule cache on workforce event", "feline_id", payload.FelineID)
			}
		} else {
			if err := l.cache.DeletePrefix(ctx, "cache:care_schedule:"); err != nil {
				l.logger.Warn("failed to evict all care schedule caches on workforce event", "error", err)
			} else {
				l.logger.Info("evicted all care schedule caches on workforce event")
			}
		}

	case "events.workforce.incident_created.v1":
		if err := l.cache.Delete(ctx, KeyActiveIncidents()); err != nil {
			l.logger.Warn("failed to evict active incident cache on incident created event", "error", err)
		} else {
			l.logger.Info("evicted active incident cache on incident created event")
		}
	}
}

// Stop terminates background consumption.
func (l *InvalidationListener) Stop() {
	if l.cancel != nil {
		l.cancel()
	}
	l.wg.Wait()
}
