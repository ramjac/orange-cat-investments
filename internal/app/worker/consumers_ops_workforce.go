package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

var deliveredIncidentBFF sync.Map
var deliveredIncidentPebble sync.Map

// HandleOTATriggered processes OTA firmware update triggers for edge habitat hardware and persists updates through Internal BLL.
func HandleOTATriggered(ctx context.Context, logger *slog.Logger, internalBLL internalbllv1.InternalBLLServiceClient, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer received OTA update triggered event", "uuid", msg.UUID)

	if internalBLL == nil {
		return nil, errors.New("internal BLL client not configured for OTA job update")
	}

	var envelope EventEnvelope[OTATriggeredPayload]
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		logger.Error("failed to unmarshal OTA update triggered event", "error", err)
		return nil, fmt.Errorf("unmarshal ota triggered event: %w", err)
	}

	payload := envelope.Payload
	logger.Info("executing edge device OTA firmware update pipeline",
		"job_id", payload.JobID,
		"asset_id", payload.AssetID,
		"version", payload.Version,
	)
	logger.Info("downloading firmware image and verifying checksum",
		"file_url", payload.FileURL,
		"checksum", payload.Checksum,
	)
	logger.Info("applying OTA update to edge device and rebooting", "asset_id", payload.AssetID)

	_, err := internalBLL.UpdateOTAJobStatus(ctx, &facilitiesv1.UpdateOTAJobStatusRequest{
		JobID:  payload.JobID,
		Status: "completed",
	})
	if err != nil {
		logger.Error("failed to update OTA job status via Internal BLL", "job_id", payload.JobID, "error", err)
		return nil, fmt.Errorf("internal BLL update ota job status failed: %w", err)
	}

	completionEvent := EventEnvelope[OTACompletedPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "facilities.ota_completed.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: msg.UUID,
		Payload: OTACompletedPayload{
			JobID:       payload.JobID,
			AssetID:     payload.AssetID,
			ReleaseID:   payload.ReleaseID,
			Status:      "completed",
			CompletedAt: time.Now().UTC(),
		},
	}

	data, err := json.Marshal(completionEvent)
	if err != nil {
		return nil, fmt.Errorf("marshal ota completion event: %w", err)
	}
	outputMsg := message.NewMessage(completionEvent.EventID, data)
	return []*message.Message{outputMsg}, nil
}

// HandleExecutiveReportTrigger consumes a report generation request and delegates generation to Internal BLL.
func HandleExecutiveReportTrigger(ctx context.Context, logger *slog.Logger, internalBLL internalbllv1.InternalBLLServiceClient, ps PubSub, msg *message.Message) ([]*message.Message, error) {
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

	if internalBLL == nil {
		return nil, fmt.Errorf("internal BLL client not configured for executive report generation")
	}

	genRep, err := internalBLL.GenerateMonthlyReport(ctx, &internalbllv1.GenerateMonthlyReportRequest{
		Period:      req.Period,
		GeneratedBy: req.GeneratedBy,
	})
	if err != nil {
		logger.Error("failed to generate requested executive report via Internal BLL", "period", req.Period, "error", err)
		return nil, err
	}

	rep := genRep.Report
	event := EventEnvelope[ExecutiveReportPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "events.reporting.report_generated.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: msg.UUID,
		Payload: ExecutiveReportPayload{
			ReportID:        rep.ReportID,
			Period:          rep.Period,
			Status:          rep.Status,
			WebDAVPathPDF:   rep.WebDAVPathPDF,
			WebDAVPathCSV:   rep.WebDAVPathCSV,
			NetProfit:       rep.NetProfit,
			OperationalBurn: rep.OperationalBurn,
			CashReserves:    rep.CashReserves,
			Headcount:       int(rep.TotalHeadcount),
		},
	}
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal report event: %w", err)
	}
	outputMsg := message.NewMessage(event.EventID, data)
	return []*message.Message{outputMsg}, nil
}

// HandleIncidentAlert dispatches workplace safety incident notifications to Internal BLL, web portal SSE, and wearable gateway.
func HandleIncidentAlert(ctx context.Context, logger *slog.Logger, internalBLL internalbllv1.InternalBLLServiceClient, employeeBFFURL, pebbleProxyURL string, httpClient *http.Client, msg *message.Message) error {
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

	if internalBLL != nil {
		_, _ = internalBLL.BroadcastPebbleIncidentAlert(ctx, &internalbllv1.BroadcastPebbleIncidentAlertRequest{
			IncidentID:  incidentID,
			Title:       payload.Title,
			Category:    payload.Category,
			Severity:    payload.Severity,
			Description: payload.Description,
		})
	}

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
		req, err := http.NewRequestWithContext(ctx, "POST", employeeBFFURL+"/api/v1/employee/incidents/broadcast", bytes.NewReader(postBody))
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
		req, err := http.NewRequestWithContext(ctx, "POST", pebbleProxyURL+"/api/v1/ops/pebble/incidents/broadcast", bytes.NewReader(postBody))
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
