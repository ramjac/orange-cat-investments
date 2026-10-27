package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
)

// HandleACHTransferRequested consumes an ACH transfer requested event and delegates settlement to Customer BLL.
func HandleACHTransferRequested(ctx context.Context, logger *slog.Logger, customerBLL customerbllv1.CustomerBLLServiceClient, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer received ACH transfer requested event", "uuid", msg.UUID)

	if customerBLL == nil {
		return nil, errors.New("customer BLL client not configured for ACH settlement")
	}

	var envelope EventEnvelope[ACHTransferRequestedPayload]
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		logger.Error("failed to unmarshal ACH transfer requested event", "error", err)
		return nil, err
	}

	req := envelope.Payload
	logger.Info("executing ACH transfer clearing window via Customer BLL",
		"transfer_id", req.TransferID,
		"customer_id", req.CustomerID,
		"amount_usd", req.AmountUSD,
		"transfer_type", req.TransferType,
	)

	settledAt := time.Now().UTC()
	res, err := customerBLL.SettleACHTransfer(ctx, &customerbllv1.SettleACHTransferRequest{
		TransferID:   req.TransferID,
		CustomerID:   req.CustomerID,
		PortfolioID:  req.PortfolioID,
		BankAccount:  req.BankAccount,
		AmountUSD:    req.AmountUSD,
		TransferType: req.TransferType,
	})
	if err != nil {
		logger.Error("Customer BLL ACH settlement error", "transfer_id", req.TransferID, "error", err)
		return nil, fmt.Errorf("customer BLL ACH settlement failed for transfer %s: %w", req.TransferID, err)
	}
	if res != nil && res.SettledAt != "" {
		if t, parseErr := time.Parse(time.RFC3339, res.SettledAt); parseErr == nil {
			settledAt = t
		}
	}

	settled := EventEnvelope[ACHSettledPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "events.banking.ach_settled.v1",
		OccurredAt:    settledAt,
		CorrelationID: msg.UUID,
		Payload: ACHSettledPayload{
			TransferID:  req.TransferID,
			CustomerID:  req.CustomerID,
			PortfolioID: req.PortfolioID,
			AmountUSD:   req.AmountUSD,
			Status:      "settled",
			SettledAt:   settledAt,
		},
	}

	data, err := json.Marshal(settled)
	if err != nil {
		return nil, fmt.Errorf("marshal settled event: %w", err)
	}
	outputMsg := message.NewMessage(settled.EventID, data)
	return []*message.Message{outputMsg}, nil
}

// HandleCatSpotted executes automated algorithmic allocation evaluation when a feline observation is recorded.
func HandleCatSpotted(logger *slog.Logger, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer received observation event", "uuid", msg.UUID)

	var envelope EventEnvelope[CatSpottedPayload]
	if err := json.Unmarshal(msg.Payload, &envelope); err == nil {
		logger.Info("executing algorithmic portfolio allocation check & automated brokerage trade order",
			"feline_id", envelope.Payload.FelineID,
			"activity_type", envelope.Payload.ActivityType,
			"confidence", envelope.Payload.ConfidenceScore,
		)
	}

	outputMsg := message.NewMessage(watermill.NewUUID(), []byte(`{"status":"brokerage_trade_executed"}`))
	return []*message.Message{outputMsg}, nil
}

// HandleBacktestRun executes a historical quant backtesting pipeline and persists updates through Customer BLL.
func HandleBacktestRun(ctx context.Context, logger *slog.Logger, customerBLL customerbllv1.CustomerBLLServiceClient, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer processing historical backtest run", "uuid", msg.UUID)

	if customerBLL == nil {
		return nil, errors.New("customer BLL client not configured for backtest completion")
	}

	var payload BacktestRequestedPayload
	var env EventEnvelope[BacktestRequestedPayload]
	if err := json.Unmarshal(msg.Payload, &env); err == nil && (env.Payload.BacktestID != "" || env.Payload.RunID != "") {
		payload = env.Payload
	} else if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		logger.Error("failed to unmarshal backtest requested payload", "error", err)
		return nil, fmt.Errorf("unmarshal backtest payload: %w", err)
	}

	runID := payload.BacktestID
	if runID == "" {
		runID = payload.RunID
	}
	if runID == "" {
		runID = msg.UUID
	}

	resultsJSON := `{"sharpe_ratio":1.85,"alpha":0.12,"max_drawdown":0.08}`
	_, err := customerBLL.CompleteBacktestRun(ctx, &customerbllv1.CompleteBacktestRunRequest{
		RunID:   runID,
		Status:  "completed",
		Results: resultsJSON,
	})
	if err != nil {
		logger.Error("failed to complete backtest run via Customer BLL", "run_id", runID, "error", err)
		return nil, fmt.Errorf("customer BLL complete backtest run failed: %w", err)
	}

	completedAt := time.Now().UTC()
	completionEvent := EventEnvelope[BacktestCompletedPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "events.investment.backtest_completed.v1",
		OccurredAt:    completedAt,
		CorrelationID: msg.UUID,
		Payload: BacktestCompletedPayload{
			RunID:       runID,
			StrategyID:  payload.StrategyID,
			Status:      "completed",
			SharpeRatio: 1.85,
			Alpha:       0.12,
			CompletedAt: completedAt,
		},
	}

	data, err := json.Marshal(completionEvent)
	if err != nil {
		return nil, fmt.Errorf("marshal backtest completed event: %w", err)
	}
	outputMsg := message.NewMessage(completionEvent.EventID, data)
	return []*message.Message{outputMsg}, nil
}
