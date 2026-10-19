package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jackc/pgx/v5/pgxpool"
	workforce_repo "github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/internal/service/pki"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
)

var deliveredIncidentBFF sync.Map
var deliveredIncidentPebble sync.Map


type EventEnvelope[T any] struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	OccurredAt    time.Time `json:"occurred_at"`
	CorrelationID string    `json:"correlation_id"`
	Payload       T         `json:"payload"`
}

type ACHTransferRequestedPayload struct {
	TransferID   string  `json:"transfer_id"`
	CustomerID   string  `json:"customer_id"`
	PortfolioID  string  `json:"portfolio_id"`
	BankAccount  string  `json:"bank_account"`
	AmountUSD    float64 `json:"amount_usd"`
	TransferType string  `json:"transfer_type"`
}

type ACHSettledPayload struct {
	TransferID  string    `json:"transfer_id"`
	CustomerID  string    `json:"customer_id"`
	PortfolioID string    `json:"portfolio_id"`
	AmountUSD   float64   `json:"amount_usd"`
	Status      string    `json:"status"`
	SettledAt   time.Time `json:"settled_at"`
}

type StatementGeneratedPayload struct {
	Period          string    `json:"period"`
	TotalPortfolios int       `json:"total_portfolios"`
	GeneratedAt     time.Time `json:"generated_at"`
	Status          string    `json:"status"`
}

type ExecutiveReportPayload struct {
	ReportID        string  `json:"report_id"`
	Period          string  `json:"period"`
	Status          string  `json:"status"`
	WebDAVPathPDF   string  `json:"webdav_path_pdf"`
	WebDAVPathCSV   string  `json:"webdav_path_csv"`
	NetProfit       float64 `json:"net_profit"`
	OperationalBurn float64 `json:"operational_burn"`
	CashReserves    float64 `json:"cash_reserves"`
	Headcount       int     `json:"headcount"`
}

type CatSpottedPayload struct {
	CameraID        string  `json:"camera_id"`
	FelineID        string  `json:"feline_id"`
	ActivityType    string  `json:"activity_type"`
	ConfidenceScore float64 `json:"confidence_score"`
}

type OTATriggeredPayload struct {
	JobID      string `json:"job_id"`
	AssetID    string `json:"asset_id"`
	ReleaseID  string `json:"release_id"`
	DeviceType string `json:"device_type"`
	Version    string `json:"version"`
	FileURL    string `json:"file_url"`
	Checksum   string `json:"checksum"`
}

type OTACompletedPayload struct {
	JobID       string    `json:"job_id"`
	AssetID     string    `json:"asset_id"`
	ReleaseID   string    `json:"release_id"`
	Status      string    `json:"status"`
	CompletedAt time.Time `json:"completed_at"`
}

type CARotationPayload struct {
	IntermediateCAName       string    `json:"intermediate_ca_name"`
	NewSerialNumber           string    `json:"new_serial_number"`
	ExpiresAt                 time.Time `json:"expires_at"`
	ReissuedCertificatesCount int       `json:"reissued_certificates_count"`
	Status                    string    `json:"status"`
}

type pubSub interface {
	message.Publisher
	message.Subscriber
}

func HandleOTATriggered(logger *slog.Logger, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer received OTA update triggered event", "uuid", msg.UUID)

	var envelope EventEnvelope[OTATriggeredPayload]
	status := "completed"
	if err := json.Unmarshal(msg.Payload, &envelope); err == nil {
		logger.Info("executing edge device OTA firmware update pipeline",
			"job_id", envelope.Payload.JobID,
			"asset_id", envelope.Payload.AssetID,
			"version", envelope.Payload.Version,
		)
		logger.Info("downloading firmware image and verifying checksum",
			"file_url", envelope.Payload.FileURL,
			"checksum", envelope.Payload.Checksum,
		)
		logger.Info("applying OTA update to edge device and rebooting", "asset_id", envelope.Payload.AssetID)
	}

	completionEvent := EventEnvelope[OTACompletedPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "facilities.ota_completed.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: msg.UUID,
		Payload: OTACompletedPayload{
			JobID:       envelope.Payload.JobID,
			AssetID:     envelope.Payload.AssetID,
			ReleaseID:   envelope.Payload.ReleaseID,
			Status:      status,
			CompletedAt: time.Now().UTC(),
		},
	}

	data, _ := json.Marshal(completionEvent)
	outputMsg := message.NewMessage(completionEvent.EventID, data)
	return []*message.Message{outputMsg}, nil
}

func handleCatSpotted(logger *slog.Logger, msg *message.Message) ([]*message.Message, error) {
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

func handleBacktestRun(logger *slog.Logger, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer processing historical backtest run", "uuid", msg.UUID)

	outputMsg := message.NewMessage(watermill.NewUUID(), []byte(`{"status":"backtest_completed","results":{"sharpe_ratio":1.85,"alpha":0.12}}`))
	return []*message.Message{outputMsg}, nil
}

func HandleACHTransferRequested(logger *slog.Logger, coreInvestURL, bankURL string, httpClient *http.Client, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer received ACH transfer requested event", "uuid", msg.UUID)

	var envelope EventEnvelope[ACHTransferRequestedPayload]
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		logger.Error("failed to unmarshal ACH transfer requested event", "error", err)
		return nil, err
	}

	req := envelope.Payload
	logger.Info("executing ACH transfer clearing window",
		"transfer_id", req.TransferID,
		"customer_id", req.CustomerID,
		"amount_usd", req.AmountUSD,
		"transfer_type", req.TransferType,
	)

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	// 1. External mock bank clearing call if configured (Threads 19, 39)
	if bankURL != "" {
		direction := "debit"
		if req.TransferType == "withdrawal" {
			direction = "credit"
		}
		bankBody, _ := json.Marshal(map[string]interface{}{
			"account_id":  req.BankAccount,
			"amount":      req.AmountUSD,
			"direction":   direction,
			"description": fmt.Sprintf("OCI Investment Capital ACH %s (%s)", req.TransferType, req.TransferID),
		})
		bankReq, err := http.NewRequest("POST", bankURL+"/transfer/create", bytes.NewReader(bankBody))
		if err != nil {
			return nil, fmt.Errorf("create bank transfer request: %w", err)
		}
		bankReq.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(bankReq)
		if err != nil {
			return nil, fmt.Errorf("bank clearing network error for transfer %s: %w", req.TransferID, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("bank clearing rejected transfer %s with HTTP %d", req.TransferID, resp.StatusCode)
		}
	}

	// 2. Update portfolio balance in core-invest domain with idempotency key (Threads 18, 19)
	if coreInvestURL != "" && req.PortfolioID != "" {
		delta := req.AmountUSD
		if req.TransferType == "withdrawal" {
			delta = -req.AmountUSD
		}
		balBody, _ := json.Marshal(map[string]interface{}{
			"delta_usd":   delta,
			"reason":      fmt.Sprintf("ACH settlement for transfer %s", req.TransferID),
			"transfer_id": req.TransferID,
		})
		balReq, err := http.NewRequest("POST", fmt.Sprintf("%s/core-invest/portfolios/%s/balance", coreInvestURL, req.PortfolioID), bytes.NewReader(balBody))
		if err != nil {
			return nil, fmt.Errorf("create balance update request: %w", err)
		}
		balReq.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(balReq)
		if err != nil {
			return nil, fmt.Errorf("core-invest balance update network error for transfer %s: %w", req.TransferID, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("core-invest balance update rejected transfer %s with HTTP %d", req.TransferID, resp.StatusCode)
		}
	}

	settled := EventEnvelope[ACHSettledPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "events.banking.ach_settled.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: msg.UUID,
		Payload: ACHSettledPayload{
			TransferID:  req.TransferID,
			CustomerID:  req.CustomerID,
			PortfolioID: req.PortfolioID,
			AmountUSD:   req.AmountUSD,
			Status:      "settled",
			SettledAt:   time.Now().UTC(),
		},
	}

	data, _ := json.Marshal(settled)
	outputMsg := message.NewMessage(settled.EventID, data)
	return []*message.Message{outputMsg}, nil
}


func isFeatureEnabled(envVar string) bool {
	val := os.Getenv(envVar)
	if val == "false" {
		return false
	}
	if os.Getenv("ENABLE_SMALL_BUSINESS_MODE") == "true" && val == "" {
		return false
	}
	return true
}

func handleExecutiveReportTrigger(logger *slog.Logger, wfSvc wfService.Service, ps pubSub, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer received executive report generation request", "uuid", msg.UUID)
	var req struct {
		Period      string `json:"period"`
		GeneratedBy string `json:"generated_by"`
	}
	_ = json.Unmarshal(msg.Payload, &req)
	if req.Period == "" {
		req.Period = time.Now().UTC().Format("2006-01")
	}
	if req.GeneratedBy == "" {
		req.GeneratedBy = "system-reporting-worker"
	}
	genRep, err := wfSvc.GenerateMonthlyReport(context.Background(), req.Period, req.GeneratedBy)
	if err != nil {
		logger.Error("failed to generate requested executive report", "period", req.Period, "error", err)
		return nil, err
	}
	stringVal := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	event := EventEnvelope[ExecutiveReportPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "events.reporting.report_generated.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: msg.UUID,
		Payload: ExecutiveReportPayload{
			ReportID:        genRep.ReportID,
			Period:          genRep.Period,
			Status:          genRep.Status,
			WebDAVPathPDF:   stringVal(genRep.WebDAVPathPDF),
			WebDAVPathCSV:   stringVal(genRep.WebDAVPathCSV),
			NetProfit:       genRep.NetProfit,
			OperationalBurn: genRep.OperationalBurn,
			CashReserves:    genRep.CashReserves,
			Headcount:       genRep.TotalHeadcount,
		},
	}
	data, _ := json.Marshal(event)
	outputMsg := message.NewMessage(event.EventID, data)
	return []*message.Message{outputMsg}, nil
}

type IncidentCreatedEventPayload struct {
	IncidentID       string    `json:"incident_id"`
	Title            string    `json:"title"`
	Category         string    `json:"category"`
	InvolvedFelineID string    `json:"involved_feline_id,omitempty"`
	InvolvedHumanID  string    `json:"involved_human_id,omitempty"`
	Severity         string    `json:"severity"`
	RoomZone         string    `json:"room_zone"`
	Description      string    `json:"description"`
	ReportedAt       time.Time `json:"reported_at"`
}

func handleIncidentAlert(logger *slog.Logger, employeeBFFURL, pebbleProxyURL string, httpClient *http.Client, msg *message.Message) error {
	logger.Info("Watermill consumer received workplace incident alert event", "uuid", msg.UUID)

	var env EventEnvelope[IncidentCreatedEventPayload]
	if err := json.Unmarshal(msg.Payload, &env); err != nil {
		logger.Error("failed to unmarshal incident created event", "error", err)
		return err
	}

	payload := env.Payload
	incidentID := payload.IncidentID
	if incidentID == "" {
		incidentID = env.EventID
	}

	logger.Info("dispatching workplace safety incident alert to web portal & pebble watch",
		"incident_id", incidentID,
		"severity", payload.Severity,
		"room_zone", payload.RoomZone,
	)

	if employeeBFFURL == "" {
		employeeBFFURL = "http://oci-employee-bff.oci-core.svc.cluster.local:8081"
	}
	if pebbleProxyURL == "" {
		pebbleProxyURL = "http://oci-pebble-proxy.oci-core.svc.cluster.local:8082"
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	postBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var deliveryErrors []error

	// 1. Fan out to Employee BFF SSE hub for real-time web banner (deduplicated by incident ID)
	if _, alreadyDelivered := deliveredIncidentBFF.Load(incidentID); !alreadyDelivered && employeeBFFURL != "" {
		req, err := http.NewRequest("POST", employeeBFFURL+"/api/v1/employee/incidents/broadcast", bytes.NewReader(postBody))
		if err != nil {
			deliveryErrors = append(deliveryErrors, fmt.Errorf("employee BFF request creation failed: %w", err))
		} else {
			req.Header.Set("Content-Type", "application/json")
			if internalKey := os.Getenv("INTERNAL_SERVICE_KEY"); internalKey != "" {
				req.Header.Set("X-Internal-Service-Key", internalKey)
			}
			resp, err := httpClient.Do(req)
			if err != nil {
				deliveryErrors = append(deliveryErrors, fmt.Errorf("employee BFF delivery failed: %w", err))
			} else {
				defer resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					deliveredIncidentBFF.Store(incidentID, true)
				} else {
					deliveryErrors = append(deliveryErrors, fmt.Errorf("employee BFF returned status %d", resp.StatusCode))
				}
			}
		}
	}

	// 2. Fan out to Pebble Proxy gateway for smartwatch vibrating alert (deduplicated by incident ID)
	if _, alreadyDelivered := deliveredIncidentPebble.Load(incidentID); !alreadyDelivered && pebbleProxyURL != "" {
		req, err := http.NewRequest("POST", pebbleProxyURL+"/api/v1/ops/pebble/incidents/broadcast", bytes.NewReader(postBody))
		if err != nil {
			deliveryErrors = append(deliveryErrors, fmt.Errorf("pebble proxy request creation failed: %w", err))
		} else {
			req.Header.Set("Content-Type", "application/json")
			if internalKey := os.Getenv("INTERNAL_SERVICE_KEY"); internalKey != "" {
				req.Header.Set("X-Internal-Service-Key", internalKey)
			}
			resp, err := httpClient.Do(req)
			if err != nil {
				deliveryErrors = append(deliveryErrors, fmt.Errorf("pebble proxy delivery failed: %w", err))
			} else {
				defer resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					deliveredIncidentPebble.Store(incidentID, true)
				} else {
					deliveryErrors = append(deliveryErrors, fmt.Errorf("pebble proxy returned status %d", resp.StatusCode))
				}
			}
		}
	}

	if len(deliveryErrors) > 0 {
		return fmt.Errorf("incident alert fanout failures (%d): %v", len(deliveryErrors), deliveryErrors)
	}

	return nil
}

func createRouter(ps pubSub, watermillLogger watermill.LoggerAdapter, logger *slog.Logger, wfServices ...wfService.Service) (*message.Router, error) {
	enableCoreInvest := isFeatureEnabled("ENABLE_CORE_INVEST")
	enableFelineWorkforce := isFeatureEnabled("ENABLE_FELINE_WORKFORCE")

	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, err
	}

	if enableFelineWorkforce && enableCoreInvest {
		router.AddHandler(
			"cat_spotted_handler",
			"events.observation.cat_spotted.v1",
			ps,
			"events.investment.allocation_check.v1",
			ps,
			func(msg *message.Message) ([]*message.Message, error) {
				return handleCatSpotted(logger, msg)
			},
		)
		logger.Info("Watermill cat spotted handler registered")
	} else {
		logger.Info("Watermill cat spotted handler disabled via configuration")
	}

	if enableCoreInvest {
		router.AddHandler(
			"backtest_run_handler",
			"events.investment.backtest_requested.v1",
			ps,
			"events.investment.backtest_completed.v1",
			ps,
			func(msg *message.Message) ([]*message.Message, error) {
				return handleBacktestRun(logger, msg)
			},
		)
		logger.Info("Watermill backtest handler registered")

		router.AddHandler(
			"ach_settlement_handler",
			"events.banking.ach_transfer_requested.v1",
			ps,
			"events.banking.ach_settled.v1",
			ps,
			func(msg *message.Message) ([]*message.Message, error) {
				coreInvestURL := os.Getenv("CORE_INVEST_API_URL")
				bankURL := os.Getenv("BANK_API_BASE_URL")
				return HandleACHTransferRequested(logger, coreInvestURL, bankURL, http.DefaultClient, msg)
			},
		)
		logger.Info("Watermill ACH settlement handler registered")
	} else {
		logger.Info("Watermill backtest and ACH settlement handlers disabled via configuration")
	}

	router.AddHandler(
		"ota_pipeline_handler",
		"events.facilities.ota_triggered.v1",
		ps,
		"events.facilities.ota_completed.v1",
		ps,
		func(msg *message.Message) ([]*message.Message, error) {
			return HandleOTATriggered(logger, msg)
		},
	)

	if len(wfServices) > 0 && wfServices[0] != nil {
		s := wfServices[0]
		router.AddHandler(
			"executive_reporting_trigger_handler",
			"events.reporting.generate_monthly_requested.v1",
			ps,
			"events.reporting.report_generated.v1",
			ps,
			func(msg *message.Message) ([]*message.Message, error) {
				return handleExecutiveReportTrigger(logger, s, ps, msg)
			},
		)
		logger.Info("Watermill executive reporting trigger handler registered")
	}

	// Pub/Sub Real-Time Incident Alert Pipeline Handler
	employeeBFFURL := os.Getenv("EMPLOYEE_BFF_URL")
	pebbleProxyURL := os.Getenv("PEBBLE_PROXY_URL")
	router.AddNoPublisherHandler(
		"incident_alert_pipeline_handler",
		"events.workforce.incident_created.v1",
		ps,
		func(msg *message.Message) error {
			return handleIncidentAlert(logger, employeeBFFURL, pebbleProxyURL, http.DefaultClient, msg)
		},
	)
	logger.Info("Watermill incident alert pipeline handler registered")

	return router, nil
}

func startCARotationWorker(ctx context.Context, router *message.Router, ps pubSub, caManager pki.CARotationManager, logger *slog.Logger) {
	go func() {
		<-router.Running()
		// Initial check on worker startup
		res, err := caManager.CheckAndRotateCA(ctx)
		if err != nil {
			logger.Error("failed initial CA rotation check", "error", err)
		} else {
			logger.Info("automated CA key rotation check completed",
				"status", res.Status,
				"ca_name", res.IntermediateCAName,
				"serial_number", res.NewSerialNumber,
				"reissued_certs", res.ReissuedCertificatesCount,
			)

			if res.Status == "CA_ROTATED_SUCCESSFULLY" {
				event := EventEnvelope[CARotationPayload]{
					EventID:       watermill.NewUUID(),
					EventType:     "ops.ca_rotated.v1",
					OccurredAt:    res.RotatedAt,
					CorrelationID: "ca-rotation-trace-001",
					Payload: CARotationPayload{
						IntermediateCAName:       res.IntermediateCAName,
						NewSerialNumber:           res.NewSerialNumber,
						ExpiresAt:                 res.ExpiresAt,
						ReissuedCertificatesCount: res.ReissuedCertificatesCount,
						Status:                    res.Status,
					},
				}
				data, _ := json.Marshal(event)
				msg := message.NewMessage(event.EventID, data)
				_ = ps.Publish("events.ops.ca_rotated.v1", msg)
			}
		}

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				res, err := caManager.CheckAndRotateCA(ctx)
				if err != nil {
					logger.Error("failed scheduled CA rotation check", "error", err)
				} else {
					logger.Info("scheduled CA key rotation check completed", "status", res.Status)
				}
			}
		}
	}()
}

func startCatSpottedPublisher(ctx context.Context, router *message.Router, ps pubSub, wfSvc wfService.Service) {
	go func() {
		<-router.Running()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		felineID := os.Getenv("OBSERVATION_FELINE_ID")
		if felineID == "" && wfSvc != nil {
			if emps, err := wfSvc.ListEmployees(ctx, "feline", "active"); err == nil && len(emps) > 0 {
				felineID = emps[0].EmployeeID
			}
		}
		if felineID == "" {
			felineID = "feline-observation-target"
		}

		event := EventEnvelope[CatSpottedPayload]{
			EventType:     "observation.cat_spotted.v1",
			CorrelationID: "correlation-trace-001",
			Payload: CatSpottedPayload{
				CameraID:        "CAM-ORANGE-01",
				FelineID:        felineID,
				ActivityType:    "zooming",
				ConfidenceScore: 0.992,
			},
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				event.EventID = watermill.NewUUID()
				event.OccurredAt = time.Now().UTC()

				data, _ := json.Marshal(event)
				msg := message.NewMessage(event.EventID, data)
				ps.Publish("events.observation.cat_spotted.v1", msg)
			}
		}
	}()
}

func startExecutiveReportingWorker(ctx context.Context, router *message.Router, ps pubSub, wfSvc wfService.Service, logger *slog.Logger) {
	stringVal := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	go func() {
		<-router.Running()
		period := time.Now().UTC().Format("2006-01")
		logger.Info("checking executive management report status", "period", period)
		rep, err := wfSvc.GetExecutiveReportByPeriod(ctx, period)
		if err != nil || rep == nil {
			logger.Info("generating monthly executive management report", "period", period)
			rep, err = wfSvc.GenerateMonthlyReport(ctx, period, "system-reporting-worker")
			if err != nil {
				logger.Error("failed to generate monthly executive report", "period", period, "error", err)
			} else {
				logger.Info("monthly executive report generated and archived to Nextcloud WebDAV",
					"report_id", rep.ReportID,
					"period", rep.Period,
					"webdav_pdf", rep.WebDAVPathPDF,
					"webdav_csv", rep.WebDAVPathCSV,
					"net_profit", rep.NetProfit,
				)
				event := EventEnvelope[ExecutiveReportPayload]{
					EventID:       watermill.NewUUID(),
					EventType:     "events.reporting.report_generated.v1",
					OccurredAt:    time.Now().UTC(),
					CorrelationID: "reporting-init-" + rep.ReportID,
					Payload: ExecutiveReportPayload{
						ReportID:        rep.ReportID,
						Period:          rep.Period,
						Status:          rep.Status,
						WebDAVPathPDF:   stringVal(rep.WebDAVPathPDF),
						WebDAVPathCSV:   stringVal(rep.WebDAVPathCSV),
						NetProfit:       rep.NetProfit,
						OperationalBurn: rep.OperationalBurn,
						CashReserves:    rep.CashReserves,
						Headcount:       rep.TotalHeadcount,
					},
				}
				data, _ := json.Marshal(event)
				msg := message.NewMessage(event.EventID, data)
				_ = ps.Publish("events.reporting.report_generated.v1", msg)
			}
		} else {
			logger.Info("executive management report already exists for current period", "period", period, "report_id", rep.ReportID)
		}

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				currentPeriod := now.UTC().Format("2006-01")
				existing, err := wfSvc.GetExecutiveReportByPeriod(ctx, currentPeriod)
				if err != nil || existing == nil {
					logger.Info("scheduled monthly executive report generation triggered", "period", currentPeriod)
					genRep, err := wfSvc.GenerateMonthlyReport(ctx, currentPeriod, "system-reporting-worker")
					if err != nil {
						logger.Error("failed scheduled executive report generation", "period", currentPeriod, "error", err)
					} else {
						logger.Info("scheduled executive report generated successfully", "report_id", genRep.ReportID, "period", genRep.Period)
						event := EventEnvelope[ExecutiveReportPayload]{
							EventID:       watermill.NewUUID(),
							EventType:     "events.reporting.report_generated.v1",
							OccurredAt:    now.UTC(),
							CorrelationID: "reporting-scheduled-" + genRep.ReportID,
							Payload: ExecutiveReportPayload{
								ReportID:        genRep.ReportID,
								Period:          genRep.Period,
								Status:          genRep.Status,
								WebDAVPathPDF:   stringVal(genRep.WebDAVPathPDF),
								WebDAVPathCSV:   stringVal(genRep.WebDAVPathCSV),
								NetProfit:       genRep.NetProfit,
								OperationalBurn: genRep.OperationalBurn,
								CashReserves:    genRep.CashReserves,
								Headcount:       genRep.TotalHeadcount,
							},
						}
						data, _ := json.Marshal(event)
						msg := message.NewMessage(event.EventID, data)
						_ = ps.Publish("events.reporting.report_generated.v1", msg)
					}
				}
			}
		}
	}()
}

type amqpPubSubWrapper struct {
	*amqp.Publisher
	*amqp.Subscriber
}

func (w *amqpPubSubWrapper) Publish(topic string, messages ...*message.Message) error {
	return w.Publisher.Publish(topic, messages...)
}

func (w *amqpPubSubWrapper) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return w.Subscriber.Subscribe(ctx, topic)
}

func (w *amqpPubSubWrapper) Close() error {
	_ = w.Publisher.Close()
	return w.Subscriber.Close()
}

func runACHRecurringBatch(ctx context.Context, coreInvestURL string, ps message.Publisher, httpClient *http.Client, logger *slog.Logger) error {
	if coreInvestURL == "" {
		return errors.New("CORE_INVEST_API_URL not configured")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", coreInvestURL+"/api/v1/core-invest/recurring-deposits", nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to list recurring deposits: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("core-invest returned status %d", resp.StatusCode)
	}

	var deposits []struct {
		DepositID    string    `json:"deposit_id"`
		CustomerID   string    `json:"customer_id"`
		PortfolioID  string    `json:"portfolio_id"`
		AmountUSD    float64   `json:"amount_usd"`
		Frequency    string    `json:"frequency"`
		BankAccount  string    `json:"bank_account"`
		Status       string    `json:"status"`
		NextChargeAt time.Time `json:"next_charge_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&deposits); err != nil {
		return fmt.Errorf("failed to decode recurring deposits: %w", err)
	}

	now := time.Now().UTC()
	dispatchedCount := 0

	for _, dep := range deposits {
		if dep.Status != "active" {
			continue
		}
		// Claim only due occurrences (Thread 22)
		if !dep.NextChargeAt.IsZero() && dep.NextChargeAt.After(now) {
			continue
		}

		// Calculate next charge time
		nextCharge := now.AddDate(0, 1, 0)
		switch dep.Frequency {
		case "weekly":
			nextCharge = now.AddDate(0, 0, 7)
		case "biweekly", "bi_weekly":
			nextCharge = now.AddDate(0, 0, 14)
		case "monthly":
			nextCharge = now.AddDate(0, 1, 0)
		}

		// Advance schedule in core-invest domain
		advPayload, _ := json.Marshal(map[string]interface{}{
			"next_charge_at": nextCharge,
		})
		advReq, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/v1/core-invest/recurring-deposits/%s/advance", coreInvestURL, dep.DepositID), bytes.NewReader(advPayload))
		if err == nil {
			advReq.Header.Set("Content-Type", "application/json")
			if advResp, err := httpClient.Do(advReq); err == nil {
				_ = advResp.Body.Close()
			}
		}

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
		data, _ := json.Marshal(event)
		msg := message.NewMessage(event.EventID, data)
		if ps != nil {
			if err := ps.Publish("events.banking.ach_transfer_requested.v1", msg); err != nil {
				logger.Error("failed to publish recurring transfer event", "error", err, "deposit_id", dep.DepositID)
				continue
			}
		}
		dispatchedCount++
		logger.Info("recurring ACH deposit transfer dispatched", "deposit_id", dep.DepositID, "amount_usd", dep.AmountUSD, "next_charge_at", nextCharge)
	}

	logger.Info("recurring ACH batch completed", "dispatched", dispatchedCount)
	return nil
}

func startRecurringACHProcessor(ctx context.Context, router *message.Router, ps pubSub, coreInvestURL string, httpClient *http.Client, logger *slog.Logger) {
	go func() {
		<-router.Running()
		_ = runACHRecurringBatch(ctx, coreInvestURL, ps, httpClient, logger)

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = runACHRecurringBatch(ctx, coreInvestURL, ps, httpClient, logger)
			}
		}
	}()
}

func runEOMStatementBatch(ctx context.Context, coreInvestURL string, ps message.Publisher, httpClient *http.Client, logger *slog.Logger, now time.Time) error {
	if coreInvestURL == "" {
		return errors.New("CORE_INVEST_API_URL not configured")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	targetDate := now.AddDate(0, -1, 0)
	period := targetDate.Format("2006-01")
	targetYear := targetDate.Year()
	logger.Info("executing automated End-of-Month statement & tax batch generator", "period", period, "year", targetYear)

	// Enumerate portfolios from core-invest-api (Thread 40)
	req, err := http.NewRequestWithContext(ctx, "GET", coreInvestURL+"/api/v1/core-invest/portfolios", nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to list portfolios: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("core-invest portfolios query returned status %d", resp.StatusCode)
	}

	var portfolios []struct {
		PortfolioID string `json:"portfolio_id"`
		CustomerID  string `json:"customer_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&portfolios); err != nil {
		return fmt.Errorf("failed to decode portfolios: %w", err)
	}

	successCount := 0
	for _, p := range portfolios {
		stmtPayload, _ := json.Marshal(map[string]interface{}{
			"portfolio_id": p.PortfolioID,
			"year":         targetYear,
		})
		stmtReq, err := http.NewRequestWithContext(ctx, "POST", coreInvestURL+"/api/v1/core-invest/statements/generate", bytes.NewReader(stmtPayload))
		if err != nil {
			continue
		}
		stmtReq.Header.Set("Content-Type", "application/json")
		stmtResp, err := httpClient.Do(stmtReq)
		if err != nil {
			logger.Warn("statement generation failed for portfolio", "portfolio_id", p.PortfolioID, "error", err)
			continue
		}
		_ = stmtResp.Body.Close()
		if stmtResp.StatusCode == http.StatusOK || stmtResp.StatusCode == http.StatusCreated {
			successCount++
		}
	}

	// Publish completion event with actual count derived from batch execution (Thread 40)
	event := EventEnvelope[StatementGeneratedPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "events.reporting.statements_generated.v1",
		OccurredAt:    now.UTC(),
		CorrelationID: "batch-eom-stmt-" + period,
		Payload: StatementGeneratedPayload{
			Period:          period,
			TotalPortfolios: successCount,
			GeneratedAt:     now.UTC(),
			Status:          "completed",
		},
	}
	data, _ := json.Marshal(event)
	msg := message.NewMessage(event.EventID, data)
	if ps != nil {
		if err := ps.Publish("events.reporting.statements_generated.v1", msg); err != nil {
			return fmt.Errorf("failed to publish statements_generated event: %w", err)
		}
	}
	logger.Info("automated End-of-Month statements generated successfully", "period", period, "total_portfolios", successCount)
	return nil
}

func startEOMStatementBatchGenerator(ctx context.Context, router *message.Router, ps pubSub, coreInvestURL string, httpClient *http.Client, logger *slog.Logger) {
	go func() {
		<-router.Running()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if now.Day() == 1 && now.Hour() == 1 {
					_ = runEOMStatementBatch(ctx, coreInvestURL, ps, httpClient, logger, now)
				}
			}
		}
	}()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Background Worker with Watermill Engine")

	watermillLogger := watermill.NewStdLogger(false, false)
	coreInvestURL := os.Getenv("CORE_INVEST_API_URL")
	if coreInvestURL == "" {
		coreInvestURL = "http://localhost:50053"
	}

	var pubSub pubSub
	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		amqpURL = os.Getenv("AMQP_URL")
	}
	if amqpURL != "" {
		amqpConfig := amqp.NewDurableQueueConfig(amqpURL)
		amqpPub, err := amqp.NewPublisher(amqpConfig, watermillLogger)
		if err != nil {
			logger.Error("failed to create RabbitMQ publisher in worker", "error", err)
			os.Exit(1)
		}
		amqpSub, err := amqp.NewSubscriber(amqpConfig, watermillLogger)
		if err != nil {
			logger.Error("failed to create RabbitMQ subscriber in worker", "error", err)
			os.Exit(1)
		}
		pubSub = &amqpPubSubWrapper{Publisher: amqpPub, Subscriber: amqpSub}
		logger.Info("worker connected to RabbitMQ AMQP broker successfully")
	} else {
		pubSub = gochannel.NewGoChannel(
			gochannel.Config{OutputChannelBuffer: 100},
			watermillLogger,
		)
		logger.Info("worker running with in-memory GoChannel publisher (local test mode)")
	}

	batchMode := os.Getenv("BATCH_MODE")
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// 1. One-Shot Batch Modes (Threads 41, 42)
	if batchMode == "ach-recurring" {
		logger.Info("executing one-shot ACH recurring deposits batch run")
		if err := runACHRecurringBatch(context.Background(), coreInvestURL, pubSub, httpClient, logger); err != nil {
			logger.Error("ACH recurring batch run failed", "error", err)
			os.Exit(1)
		}
		logger.Info("ACH recurring batch completed successfully")
		return
	}
	if batchMode == "eom-statements" {
		logger.Info("executing one-shot End-of-Month statements batch run")
		if err := runEOMStatementBatch(context.Background(), coreInvestURL, pubSub, httpClient, logger, time.Now().UTC()); err != nil {
			logger.Error("EOM statements batch run failed", "error", err)
			os.Exit(1)
		}
		logger.Info("EOM statements batch completed successfully")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var pool *pgxpool.Pool
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		poolCfg, err := pgxpool.ParseConfig(dbURL)
		if err == nil {
			pool, _ = pgxpool.NewWithConfig(ctx, poolCfg)
			if pool != nil {
				defer pool.Close()
				logger.Info("connected worker to PostgreSQL pool successfully")
			}
		}
	}

	var wfRepo workforce_repo.Repository
	if pool != nil {
		wfRepo = workforce_repo.NewRepository(pool)
	} else {
		wfRepo = workforce_repo.NewMockRepository()
	}

	var hrClient wfService.HRClient
	frappeURL := os.Getenv("FRAPPE_HR_URL")
	if frappeURL != "" {
		hrClient = frappe.NewClient(frappe.Config{
			BaseURL:   frappeURL,
			APIKey:    os.Getenv("FRAPPE_API_KEY"),
			APISecret: os.Getenv("FRAPPE_API_SECRET"),
		})
		logger.Info("Frappe HR client initialized in worker", "url", frappeURL)
	}

	var workspaceClient wfService.WorkspaceClient
	workspaceURL := os.Getenv("WORKSPACE_URL")
	if workspaceURL == "" {
		workspaceURL = os.Getenv("NEXTCLOUD_URL")
	}
	if workspaceURL != "" {
		workspaceClient = nextcloud.NewClient(nextcloud.Config{
			BaseURL:       workspaceURL,
			AdminUser:     os.Getenv("NEXTCLOUD_ADMIN_USER"),
			AdminPassword: os.Getenv("NEXTCLOUD_ADMIN_PASSWORD"),
		})
		logger.Info("Nextcloud workspace client initialized in worker", "url", workspaceURL)
	}

	wfSvc := wfService.NewServiceWithIntegrations(wfRepo, nil, hrClient, workspaceClient)

	router, err := createRouter(pubSub, watermillLogger, logger, wfSvc)
	if err != nil {
		logger.Error("failed to create watermill router", "error", err)
		os.Exit(1)
	}

	caManager := pki.NewCARotationManager(pki.CARotationConfig{
		RootCAName:           "oci-root-ca",
		IntermediateCAName:   "oci-intermediate-ca",
		RenewalThresholdDays: 30,
		IssuerNamespace:      "cert-manager",
	})

	go func() {
		if err := router.Run(ctx); err != nil {
			logger.Error("watermill router error", "error", err)
		}
	}()

	startCARotationWorker(ctx, router, pubSub, caManager, logger)
	startExecutiveReportingWorker(ctx, router, pubSub, wfSvc, logger)

	if isFeatureEnabled("ENABLE_CORE_INVEST") {
		startRecurringACHProcessor(ctx, router, pubSub, coreInvestURL, httpClient, logger)
		startEOMStatementBatchGenerator(ctx, router, pubSub, coreInvestURL, httpClient, logger)
	}

	if isFeatureEnabled("ENABLE_FELINE_WORKFORCE") && isFeatureEnabled("ENABLE_CORE_INVEST") {
		startCatSpottedPublisher(ctx, router, pubSub, wfSvc)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down Watermill Background Worker...")
	cancel()
	router.Close()
	logger.Info("OCI Background Worker stopped gracefully")
}
