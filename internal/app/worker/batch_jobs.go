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

// RunACHRecurringBatch claims due recurring deposits from Customer BLL and dispatches settlement transfers.
func RunACHRecurringBatch(ctx context.Context, customerBLL customerbllv1.CustomerBLLServiceClient, ps message.Publisher, logger *slog.Logger) error {
	if customerBLL == nil {
		return errors.New("Customer BLL client not configured for ACH recurring batch")
	}

	res, err := customerBLL.ClaimDueRecurringDeposits(ctx, &customerbllv1.ClaimDueRecurringDepositsRequest{})
	if err != nil {
		return fmt.Errorf("failed to claim due recurring deposits via Customer BLL: %w", err)
	}

	now := time.Now().UTC()
	dispatchedCount := 0

	for _, dep := range res.DueDeposits {
		transferID := fmt.Sprintf("ach-rec-%s-%s", dep.DepositID, now.Format("2006010215"))
		payload := ACHTransferRequestedPayload{
			TransferID:   transferID,
			CustomerID:   dep.CustomerID,
			PortfolioID:  dep.PortfolioID,
			BankAccount:  dep.BankAccount,
			AmountUSD:    dep.AmountUSD,
			TransferType: "deposit",
		}
		event := EventEnvelope[ACHTransferRequestedPayload]{
			EventID:       watermill.NewUUID(),
			EventType:     "events.banking.ach_transfer_requested.v1",
			OccurredAt:    now,
			CorrelationID: "cron-recurring-ach-" + dep.DepositID,
			Payload:       payload,
		}
		data, err := json.Marshal(event)
		if err != nil {
			logger.Error("failed to marshal recurring ACH event", "error", err, "deposit_id", dep.DepositID)
			continue
		}
		msg := message.NewMessage(event.EventID, data)
		if ps != nil {
			if err := ps.Publish("events.banking.ach_transfer_requested.v1", msg); err != nil {
				logger.Error("failed to publish recurring transfer event", "error", err, "deposit_id", dep.DepositID)
				continue
			}
		}
		dispatchedCount++
		logger.Info("recurring ACH deposit transfer dispatched via Customer BLL",
			"deposit_id", dep.DepositID,
			"amount_usd", dep.AmountUSD,
		)
	}

	logger.Info("completed ACH recurring deposit batch cycle via Customer BLL", "dispatched_transfers", dispatchedCount)
	return nil
}

// RunEOMStatementBatch coordinates End-of-Month PDF statement generation across all customer portfolios via Customer BLL.
func RunEOMStatementBatch(ctx context.Context, customerBLL customerbllv1.CustomerBLLServiceClient, ps message.Publisher, logger *slog.Logger, now time.Time) error {
	if customerBLL == nil {
		return errors.New("Customer BLL client not configured for EOM statement batch")
	}

	targetDate := now.AddDate(0, -1, 0)
	period := targetDate.Format("2006-01")
	targetYear := targetDate.Year()
	logger.Info("executing automated End-of-Month statement & tax batch generator via Customer BLL", "period", period, "year", targetYear)

	res, err := customerBLL.GenerateBatchStatements(ctx, &customerbllv1.GenerateBatchStatementsRequest{
		Year:   int32(targetYear),
		Period: period,
	})
	if err != nil {
		return fmt.Errorf("customer BLL statement batch generation failed: %w", err)
	}

	event := EventEnvelope[StatementGeneratedPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "events.reporting.statements_generated.v1",
		OccurredAt:    now.UTC(),
		CorrelationID: "batch-eom-stmt-" + period,
		Payload: StatementGeneratedPayload{
			Period:          period,
			TotalPortfolios: int(res.TotalPortfolios),
			GeneratedAt:     now.UTC(),
			Status:          "completed",
		},
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal statement generated event: %w", err)
	}
	msg := message.NewMessage(event.EventID, data)
	if ps != nil {
		if err := ps.Publish("events.reporting.statements_generated.v1", msg); err != nil {
			return fmt.Errorf("failed to publish statements_generated event: %w", err)
		}
	}
	logger.Info("automated End-of-Month statements generated successfully via Customer BLL",
		"period", period,
		"total_portfolios", res.TotalPortfolios,
	)
	return nil
}
