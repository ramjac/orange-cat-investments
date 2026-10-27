package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/internal/service/pki"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

// StartCARotationWorker initiates the automated intermediate CA certificate rotation worker.
func StartCARotationWorker(ctx context.Context, router *message.Router, ps PubSub, caManager pki.CARotationManager, logger *slog.Logger) {
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

// StartCatSpottedPublisher periodically publishes feline telemetry observation events.
func StartCatSpottedPublisher(ctx context.Context, router *message.Router, ps PubSub, internalBLL internalbllv1.InternalBLLServiceClient) {
	go func() {
		<-router.Running()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		felineID := os.Getenv("OBSERVATION_FELINE_ID")
		if felineID == "" && internalBLL != nil {
			if res, err := internalBLL.GetFelineTarget(ctx, &internalbllv1.GetFelineTargetRequest{}); err == nil && res.FelineID != "" {
				felineID = res.FelineID
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
				_ = ps.Publish("events.observation.cat_spotted.v1", msg)
			}
		}
	}()
}

// StartExecutiveReportingWorker periodically verifies and generates monthly executive reports via Internal BLL.
func StartExecutiveReportingWorker(ctx context.Context, router *message.Router, ps PubSub, internalBLL internalbllv1.InternalBLLServiceClient, logger *slog.Logger) {
	go func() {
		<-router.Running()
		if internalBLL == nil {
			return
		}

		period := time.Now().UTC().Format("2006-01")
		logger.Info("checking executive management report status via Internal BLL", "period", period)
		chk, err := internalBLL.CheckExecutiveReport(ctx, &internalbllv1.CheckExecutiveReportRequest{Period: period})
		if err != nil || chk == nil || !chk.Exists {
			logger.Info("generating monthly executive management report via Internal BLL", "period", period)
			rep, err := internalBLL.GenerateMonthlyReport(ctx, &internalbllv1.GenerateMonthlyReportRequest{
				Period:      period,
				GeneratedBy: "system-reporting-worker",
			})
			if err != nil {
				logger.Error("failed to generate monthly executive report", "period", period, "error", err)
			} else if rep != nil && rep.Report != nil {
				r := rep.Report
				logger.Info("monthly executive report generated and archived to Nextcloud WebDAV",
					"report_id", r.ReportID,
					"period", r.Period,
					"webdav_pdf", r.WebDAVPathPDF,
					"webdav_csv", r.WebDAVPathCSV,
					"net_profit", r.NetProfit,
				)
				event := EventEnvelope[ExecutiveReportPayload]{
					EventID:       watermill.NewUUID(),
					EventType:     "events.reporting.report_generated.v1",
					OccurredAt:    time.Now().UTC(),
					CorrelationID: "reporting-init-" + r.ReportID,
					Payload: ExecutiveReportPayload{
						ReportID:        r.ReportID,
						Period:          r.Period,
						Status:          r.Status,
						WebDAVPathPDF:   r.WebDAVPathPDF,
						WebDAVPathCSV:   r.WebDAVPathCSV,
						NetProfit:       r.NetProfit,
						OperationalBurn: r.OperationalBurn,
						CashReserves:    r.CashReserves,
						Headcount:       int(r.TotalHeadcount),
					},
				}
				data, _ := json.Marshal(event)
				msg := message.NewMessage(event.EventID, data)
				_ = ps.Publish("events.reporting.report_generated.v1", msg)
			}
		} else {
			logger.Info("executive management report already exists for current period", "period", period, "report_id", chk.ReportID)
		}

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				currentPeriod := now.UTC().Format("2006-01")
				existing, err := internalBLL.CheckExecutiveReport(ctx, &internalbllv1.CheckExecutiveReportRequest{Period: currentPeriod})
				if err != nil || existing == nil || !existing.Exists {
					logger.Info("scheduled monthly executive report generation triggered via Internal BLL", "period", currentPeriod)
					genRep, err := internalBLL.GenerateMonthlyReport(ctx, &internalbllv1.GenerateMonthlyReportRequest{
						Period:      currentPeriod,
						GeneratedBy: "system-reporting-worker",
					})
					if err != nil {
						logger.Error("failed scheduled executive report generation", "period", currentPeriod, "error", err)
					} else if genRep != nil && genRep.Report != nil {
						r := genRep.Report
						logger.Info("scheduled executive report generated successfully", "report_id", r.ReportID, "period", r.Period)
						event := EventEnvelope[ExecutiveReportPayload]{
							EventID:       watermill.NewUUID(),
							EventType:     "events.reporting.report_generated.v1",
							OccurredAt:    now.UTC(),
							CorrelationID: "reporting-scheduled-" + r.ReportID,
							Payload: ExecutiveReportPayload{
								ReportID:        r.ReportID,
								Period:          r.Period,
								Status:          r.Status,
								WebDAVPathPDF:   r.WebDAVPathPDF,
								WebDAVPathCSV:   r.WebDAVPathCSV,
								NetProfit:       r.NetProfit,
								OperationalBurn: r.OperationalBurn,
								CashReserves:    r.CashReserves,
								Headcount:       int(r.TotalHeadcount),
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

// StartRecurringACHProcessor starts the recurring ACH deposit runner on 1-hour intervals.
func StartRecurringACHProcessor(ctx context.Context, router *message.Router, ps PubSub, customerBLL customerbllv1.CustomerBLLServiceClient, logger *slog.Logger) {
	go func() {
		<-router.Running()
		_ = RunACHRecurringBatch(ctx, customerBLL, ps, logger)

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = RunACHRecurringBatch(ctx, customerBLL, ps, logger)
			}
		}
	}()
}

// StartEOMStatementBatchGenerator starts the monthly statement scheduler.
func StartEOMStatementBatchGenerator(ctx context.Context, router *message.Router, ps PubSub, customerBLL customerbllv1.CustomerBLLServiceClient, logger *slog.Logger) {
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
					_ = RunEOMStatementBatch(ctx, customerBLL, ps, logger, now)
				}
			}
		}
	}()
}
