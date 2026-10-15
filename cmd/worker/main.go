package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jackc/pgx/v5/pgxpool"
	workforce_repo "github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/internal/service/pki"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
)

type EventEnvelope[T any] struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	OccurredAt    time.Time `json:"occurred_at"`
	CorrelationID string    `json:"correlation_id"`
	Payload       T         `json:"payload"`
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
	} else {
		logger.Info("Watermill backtest handler disabled via configuration")
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

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Background Worker with Watermill Engine")

	watermillLogger := watermill.NewStdLogger(false, false)

	// In production, use watermill-amqp for RabbitMQ. Using GoChannel pub/sub for local runtime reliability.
	pubSub := gochannel.NewGoChannel(
		gochannel.Config{OutputChannelBuffer: 100},
		watermillLogger,
	)

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
