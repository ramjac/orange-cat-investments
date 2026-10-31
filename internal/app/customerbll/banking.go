package customerbll

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/pkg/telemetry"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) InitiateACHDeposit(ctx context.Context, in *customerbllv1.InitiateACHDepositRequest) (*customerbllv1.InitiateACHDepositResponse, error) {
	if in == nil || in.CustomerID == "" || in.AmountUSD <= 0 {
		return nil, status.Error(codes.InvalidArgument, "customer_id and amount_usd > 0 required")
	}

	portfolioID := in.PortfolioID
	if portfolioID == "" && s.brokerClient != nil {
		accs, err := s.brokerClient.ListAccounts(ctx)
		if err == nil {
			for _, a := range accs {
				if a.ID == in.CustomerID || a.AccountNumber == in.CustomerID || (in.CustomerID != "" && strings.Contains(strings.ToLower(a.CustomerName), strings.ToLower(in.CustomerID))) {
					portfolioID = a.ID
					break
				}
			}
			if portfolioID == "" && len(accs) > 0 {
				portfolioID = accs[0].ID
			}
		}
	}
	if portfolioID == "" {
		portfolioID = fmt.Sprintf("port-%s", in.CustomerID)
	}

	accountID := in.AccountID
	if accountID == "" && s.bankClient != nil {
		accs, err := s.bankClient.GetAccounts(ctx, in.CustomerID)
		if err == nil && len(accs) > 0 {
			accountID = accs[0].AccountID
		}
	}

	freq := in.Frequency
	if freq == "" {
		freq = "one_time"
	}

	var createdDep *coreinvestv1.RecurringDeposit
	if s.ciClient != nil {
		res, err := s.ciClient.CreateRecurringDeposit(ctx, &coreinvestv1.CreateRecurringDepositRequest{
			CustomerID:  in.CustomerID,
			PortfolioID: portfolioID,
			AmountUSD:   in.AmountUSD,
			Frequency:   freq,
			BankAccount: in.BankAccount,
		})
		telemetry.RecordBankLinkingAttempt(ctx, false)
		if err != nil {
			s.log().Warn("failed to persist recurring deposit schedule in core-invest-api via gRPC", "error", err)
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		if res != nil {
			createdDep = res.Deposit
		}
	}

	if createdDep == nil {
		now := time.Now().UTC()
		createdDep = &coreinvestv1.RecurringDeposit{
			DepositID:    fmt.Sprintf("dep-%d", now.UnixNano()),
			CustomerID:   in.CustomerID,
			PortfolioID:  portfolioID,
			AmountUSD:    in.AmountUSD,
			Frequency:    freq,
			BankAccount:  in.BankAccount,
			Status:       "active",
			NextChargeAt: now.AddDate(0, 1, 0).Format(time.RFC3339),
		}
	}

	transferID := "ach-" + watermill.NewUUID()[:8]
	nowStr := time.Now().UTC().Format(time.RFC3339)

	achPayload := map[string]interface{}{
		"transfer_id":   transferID,
		"customer_id":   in.CustomerID,
		"portfolio_id":  portfolioID,
		"bank_account":  in.BankAccount,
		"amount_usd":    in.AmountUSD,
		"transfer_type": "deposit",
		"frequency":     freq,
	}
	eventID := watermill.NewUUID()
	event := envelope.NewEventEnvelope(eventID, envelope.TopicACHTransferRequested, transferID, achPayload)
	eventBytes, _ := json.Marshal(event)
	if s.publisher != nil {
		if err := s.publisher.Publish(envelope.TopicACHTransferRequested, message.NewMessage(event.EventID, eventBytes)); err != nil {
			s.log().Error("failed to publish ACH transfer requested event", "error", err)
			return nil, status.Error(codes.Internal, "failed to publish ACH transfer event to message broker")
		}
	}

	return &customerbllv1.InitiateACHDepositResponse{
		TransferID:  transferID,
		CustomerID:  in.CustomerID,
		PortfolioID: portfolioID,
		AmountUSD:   in.AmountUSD,
		Frequency:   freq,
		Status:      "pending_settlement",
		RequestedAt: nowStr,
		Deposit:     createdDep,
	}, nil
}

func (s *Server) ListDeposits(ctx context.Context, in *customerbllv1.ListDepositsRequest) (*customerbllv1.ListDepositsResponse, error) {
	if s.ciClient == nil {
		return &customerbllv1.ListDepositsResponse{Deposits: []*coreinvestv1.RecurringDeposit{}}, nil
	}
	res, err := s.ciClient.ListRecurringDeposits(ctx, &coreinvestv1.ListRecurringDepositsRequest{
		CustomerID: in.CustomerID,
	})
	telemetry.RecordBankLinkingAttempt(ctx, false)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list recurring deposits: %v", err)
	}
	return &customerbllv1.ListDepositsResponse{Deposits: res.Deposits}, nil
}

func (s *Server) ListBankAccounts(ctx context.Context, in *customerbllv1.ListBankAccountsRequest) (*customerbllv1.ListBankAccountsResponse, error) {
	if in == nil || in.CustomerID == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id is required")
	}

	if s.bankClient != nil {
		accs, err := s.bankClient.GetAccounts(ctx, in.CustomerID)
		if err == nil && len(accs) > 0 {
			out := make([]*customerbllv1.LinkedBankAccount, 0, len(accs))
			for _, a := range accs {
				out = append(out, &customerbllv1.LinkedBankAccount{
					AccountID:        a.AccountID,
					InstitutionName:  a.InstitutionName,
					AccountNumber:    a.AccountNumber,
					Name:             a.Name,
					Type:             a.AccountType,
					Subtype:          a.AccountSubtype,
					AvailableBalance: a.AvailableBalance,
					CurrentBalance:   a.CurrentBalance,
				})
			}
			return &customerbllv1.ListBankAccountsResponse{Accounts: out}, nil
		}
	}

	return &customerbllv1.ListBankAccountsResponse{
		Accounts: []*customerbllv1.LinkedBankAccount{
			{
				AccountID:        fmt.Sprintf("acc_%s_primary", in.CustomerID),
				InstitutionName:  "Primary Linked Institution",
				AccountNumber:    "****1001",
				Name:             "Primary Checking Account",
				AvailableBalance: 10000.00,
				CurrentBalance:   10000.00,
			},
		},
	}, nil
}

func (s *Server) SettleACHTransfer(ctx context.Context, in *customerbllv1.SettleACHTransferRequest) (*customerbllv1.SettleACHTransferResponse, error) {
	if in == nil || in.TransferID == "" {
		return nil, status.Error(codes.InvalidArgument, "transfer_id is required")
	}

	// 1. External bank clearing call if configured
	if s.bankClient != nil {
		direction := "debit"
		if in.TransferType == "withdrawal" {
			direction = "credit"
		}
		_, err := s.bankClient.CreateTransfer(ctx, plaid.CreateTransferRequest{
			AccountID:   in.BankAccount,
			Amount:      in.AmountUSD,
			Direction:   direction,
			Description: fmt.Sprintf("OCI Investment Capital ACH %s (%s)", in.TransferType, in.TransferID),
		})
		telemetry.RecordBankLinkingAttempt(ctx, false)
		if err != nil {
			return nil, status.Errorf(codes.Unavailable, "bank clearing failed for transfer %s: %v", in.TransferID, err)
		}
	}

	// 2. Update portfolio balance in core-invest domain
	if s.ciClient != nil && in.PortfolioID != "" {
		delta := in.AmountUSD
		if in.TransferType == "withdrawal" {
			delta = -in.AmountUSD
		}
		_, err := s.ciClient.UpdatePortfolioBalance(ctx, &coreinvestv1.UpdatePortfolioBalanceRequest{
			PortfolioID: in.PortfolioID,
			DeltaUSD:    delta,
			Reason:      fmt.Sprintf("ACH settlement for transfer %s", in.TransferID),
		})
		telemetry.RecordBankLinkingAttempt(ctx, false)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "core-invest balance update failed for transfer %s: %v", in.TransferID, err)
		}
	}

	return &customerbllv1.SettleACHTransferResponse{
		TransferID:  in.TransferID,
		CustomerID:  in.CustomerID,
		PortfolioID: in.PortfolioID,
		AmountUSD:   in.AmountUSD,
		Status:      "settled",
		SettledAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *Server) ClaimDueRecurringDeposits(ctx context.Context, in *customerbllv1.ClaimDueRecurringDepositsRequest) (*customerbllv1.ClaimDueRecurringDepositsResponse, error) {
	if s.ciClient == nil {
		return nil, status.Error(codes.Unavailable, "core-invest service unavailable")
	}
	res, err := s.ciClient.ListRecurringDeposits(ctx, &coreinvestv1.ListRecurringDepositsRequest{})
	telemetry.RecordBankLinkingAttempt(ctx, false)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list recurring deposits: %v", err)
	}

	now := time.Now().UTC()
	var due []*customerbllv1.RecurringDepositSummary
	for _, dep := range res.Deposits {
		if dep.Status != "active" {
			continue
		}
		nextChargeTime, _ := time.Parse(time.RFC3339, dep.NextChargeAt)
		if !nextChargeTime.IsZero() && nextChargeTime.After(now) {
			continue
		}
		nextCharge := now.AddDate(0, 1, 0)
		switch dep.Frequency {
		case "weekly":
			nextCharge = now.AddDate(0, 0, 7)
		case "biweekly", "bi_weekly":
			nextCharge = now.AddDate(0, 0, 14)
		case "monthly":
			nextCharge = now.AddDate(0, 1, 0)
		}
		nextChargeStr := nextCharge.Format(time.RFC3339)
		_, _ = s.ciClient.AdvanceRecurringDeposit(ctx, &coreinvestv1.AdvanceRecurringDepositRequest{
			DepositID:    dep.DepositID,
			NextChargeAt: nextChargeStr,
		})
		due = append(due, &customerbllv1.RecurringDepositSummary{
			DepositID:    dep.DepositID,
			CustomerID:   dep.CustomerID,
			PortfolioID:  dep.PortfolioID,
			AmountUSD:    dep.AmountUSD,
			Frequency:    dep.Frequency,
			BankAccount:  dep.BankAccount,
			Status:       dep.Status,
			NextChargeAt: nextChargeStr,
		})
	}

	return &customerbllv1.ClaimDueRecurringDepositsResponse{DueDeposits: due}, nil
}
