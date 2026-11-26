package worker

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

// AMQPPubSubWrapper wraps Watermill AMQP publisher and subscriber.
type AMQPPubSubWrapper struct {
	*amqp.Publisher
	*amqp.Subscriber
}

func (w *AMQPPubSubWrapper) Publish(topic string, messages ...*message.Message) error {
	return w.Publisher.Publish(topic, messages...)
}

func (w *AMQPPubSubWrapper) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return w.Subscriber.Subscribe(ctx, topic)
}

func (w *AMQPPubSubWrapper) SubscribeInitialize(topic string) error {
	return w.Subscriber.SubscribeInitialize(topic)
}

func (w *AMQPPubSubWrapper) Close() error {
	_ = w.Publisher.Close()
	return w.Subscriber.Close()
}

// NewAMQPPubSub creates a durable RabbitMQ publisher and subscriber wrapper.
func NewAMQPPubSub(amqpURL string, watermillLogger watermill.LoggerAdapter) (PubSub, error) {
	amqpConfig := amqp.NewDurableQueueConfig(amqpURL)
	amqpPub, err := amqp.NewPublisher(amqpConfig, watermillLogger)
	if err != nil {
		return nil, err
	}
	amqpSub, err := amqp.NewSubscriber(amqpConfig, watermillLogger)
	if err != nil {
		_ = amqpPub.Close()
		return nil, err
	}
	return &AMQPPubSubWrapper{Publisher: amqpPub, Subscriber: amqpSub}, nil
}

// IsFeatureEnabled evaluates whether a domain feature flag is activated.
func IsFeatureEnabled(envVar string) bool {
	val := os.Getenv(envVar)
	if val == "false" {
		return false
	}
	if os.Getenv("ENABLE_SMALL_BUSINESS_MODE") == "true" && val == "" {
		return false
	}
	return true
}

// CreateRouter constructs the Watermill message router and registers domain handlers
// with default retry, envelope validation, and per-topic dead-letter queue middleware.
func CreateRouter(
	ps PubSub,
	watermillLogger watermill.LoggerAdapter,
	logger *slog.Logger,
	customerBLL customerbllv1.CustomerBLLServiceClient,
	internalBLL internalbllv1.InternalBLLServiceClient,
	employeeBFFURL, pebbleProxyURL string,
	httpClient *http.Client,
) (*message.Router, error) {
	return CreateRouterWithRetry(
		ps,
		watermillLogger,
		logger,
		customerBLL,
		internalBLL,
		employeeBFFURL,
		pebbleProxyURL,
		httpClient,
		DefaultRetryConfig(watermillLogger),
	)
}

// CreateRouterWithRetry constructs the Watermill message router with a caller-specified
// retry configuration for domain handlers.
func CreateRouterWithRetry(
	ps PubSub,
	watermillLogger watermill.LoggerAdapter,
	logger *slog.Logger,
	customerBLL customerbllv1.CustomerBLLServiceClient,
	internalBLL internalbllv1.InternalBLLServiceClient,
	employeeBFFURL, pebbleProxyURL string,
	httpClient *http.Client,
	retryCfg middleware.Retry,
) (*message.Router, error) {
	enableCoreInvest := IsFeatureEnabled("ENABLE_CORE_INVEST")
	enableFelineWorkforce := IsFeatureEnabled("ENABLE_FELINE_WORKFORCE")

	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, err
	}
	router.AddMiddleware(TraceContextMiddleware)

	if enableFelineWorkforce && enableCoreInvest {
		h := router.AddHandler(
			"cat_spotted_handler",
			"events.observation.cat_spotted.v1",
			ps,
			"events.investment.allocation_check.v1",
			ps,
			func(msg *message.Message) ([]*message.Message, error) {
				return HandleCatSpotted(logger, msg)
			},
		)
		if err := AttachHandlerResilience(h, "events.observation.cat_spotted.v1", ps, retryCfg, logger); err != nil {
			return nil, err
		}
		logger.Info("Watermill cat spotted handler registered")
	} else {
		logger.Info("Watermill cat spotted handler disabled via configuration")
	}

	if enableCoreInvest {
		btHandler := router.AddHandler(
			"backtest_run_handler",
			"events.investment.backtest_requested.v1",
			ps,
			"events.investment.backtest_completed.v1",
			ps,
			func(msg *message.Message) ([]*message.Message, error) {
				return HandleBacktestRun(msg.Context(), logger, customerBLL, msg)
			},
		)
		if err := AttachHandlerResilience(btHandler, "events.investment.backtest_requested.v1", ps, retryCfg, logger); err != nil {
			return nil, err
		}
		logger.Info("Watermill backtest handler registered")

		achHandler := router.AddHandler(
			"ach_settlement_handler",
			"events.banking.ach_transfer_requested.v1",
			ps,
			"events.banking.ach_settled.v1",
			ps,
			func(msg *message.Message) ([]*message.Message, error) {
				return HandleACHTransferRequested(msg.Context(), logger, customerBLL, msg)
			},
		)
		if err := AttachHandlerResilience(achHandler, "events.banking.ach_transfer_requested.v1", ps, retryCfg, logger); err != nil {
			return nil, err
		}
		logger.Info("Watermill ACH settlement handler registered")
	} else {
		logger.Info("Watermill backtest and ACH settlement handlers disabled via configuration")
	}

	otaHandler := router.AddHandler(
		"ota_pipeline_handler",
		"events.facilities.ota_triggered.v1",
		ps,
		"events.facilities.ota_completed.v1",
		ps,
		func(msg *message.Message) ([]*message.Message, error) {
			return HandleOTATriggered(msg.Context(), logger, internalBLL, msg)
		},
	)
	if err := AttachHandlerResilience(otaHandler, "events.facilities.ota_triggered.v1", ps, retryCfg, logger); err != nil {
		return nil, err
	}

	if internalBLL != nil {
		repHandler := router.AddHandler(
			"executive_reporting_trigger_handler",
			"events.reporting.generate_monthly_requested.v1",
			ps,
			"events.reporting.report_generated.v1",
			ps,
			func(msg *message.Message) ([]*message.Message, error) {
				return HandleExecutiveReportTrigger(msg.Context(), logger, internalBLL, ps, msg)
			},
		)
		if err := AttachHandlerResilience(repHandler, "events.reporting.generate_monthly_requested.v1", ps, retryCfg, logger); err != nil {
			return nil, err
		}
		logger.Info("Watermill executive reporting trigger handler registered")
	}

	// Pub/Sub Real-Time Incident Alert Pipeline Handler
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	incHandler := router.AddNoPublisherHandler(
		"incident_alert_pipeline_handler",
		"events.workforce.incident_created.v1",
		ps,
		func(msg *message.Message) error {
			return HandleIncidentAlert(msg.Context(), logger, internalBLL, employeeBFFURL, pebbleProxyURL, httpClient, msg)
		},
	)
	if err := AttachHandlerResilience(incHandler, "events.workforce.incident_created.v1", ps, retryCfg, logger); err != nil {
		return nil, err
	}
	logger.Info("Watermill incident alert pipeline handler registered")

	if smtpNotifier := NewSMTPNotifierFromEnv(logger); smtpNotifier != nil {
		onbEmailHandler := router.AddNoPublisherHandler(
			"onboarding_welcome_email_handler",
			envelope.TopicEmployeeOnboarded,
			ps,
			func(msg *message.Message) error {
				return HandleEmployeeOnboarded(msg.Context(), logger, smtpNotifier, msg)
			},
		)
		if err := AttachHandlerResilience(onbEmailHandler, envelope.TopicEmployeeOnboarded, ps, retryCfg, logger); err != nil {
			return nil, err
		}

		tradeEmailHandler := router.AddNoPublisherHandler(
			"trade_confirmation_email_handler",
			envelope.TopicTradeExecuted,
			ps,
			func(msg *message.Message) error {
				return HandleTradeExecutedEmail(msg.Context(), logger, smtpNotifier, msg)
			},
		)
		if err := AttachHandlerResilience(tradeEmailHandler, envelope.TopicTradeExecuted, ps, retryCfg, logger); err != nil {
			return nil, err
		}

		achEmailHandler := router.AddNoPublisherHandler(
			"ach_settled_email_handler",
			envelope.TopicACHSettled,
			ps,
			func(msg *message.Message) error {
				return HandleACHSettledEmail(msg.Context(), logger, smtpNotifier, msg)
			},
		)
		if err := AttachHandlerResilience(achEmailHandler, envelope.TopicACHSettled, ps, retryCfg, logger); err != nil {
			return nil, err
		}
		logger.Info("Watermill SMTP transactional email handlers registered")
	}

	return router, nil
}
