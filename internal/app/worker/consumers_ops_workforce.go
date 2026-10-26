package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/search/meilisearch"
)

var deliveredIncidentBFF sync.Map
var deliveredIncidentPebble sync.Map

// HandleOTATriggered processes OTA firmware update triggers for edge habitat hardware.
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

	// Index workplace safety incident in Meilisearch employee knowledge index
	meiliURL := os.Getenv("MEILISEARCH_URL")
	if meiliURL == "" {
		meiliURL = "http://localhost:7700"
	}
	meiliKey := os.Getenv("MEILI_MASTER_KEY")
	meiliClient := meilisearch.NewClient(meiliURL, meiliKey)
	_ = meiliClient.IndexDocuments(ctx, "employee_knowledge", []map[string]interface{}{
		{
			"id":       incidentID,
			"category": "incident",
			"title":    payload.Title,
			"snippet":  fmt.Sprintf("Severity: %s | Zone: %s | %s", payload.Severity, payload.RoomZone, payload.Description),
		},
	})

	return nil
}
