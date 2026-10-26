package internalbll

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/pkg/envelope"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var validIncidentCategories = map[string]bool{
	"scratching":          true,
	"unauthorized_perch":  true,
	"hairball_emergency":  true,
	"habitat_disruption":  true,
	"medical_hold_breach": true,
}

func (s *Server) CreateIncident(ctx context.Context, in *workforcev1.CreateIncidentRequest) (*workforcev1.CreateIncidentResponse, error) {
	if in.Title == "" || in.Description == "" {
		return nil, status.Error(codes.InvalidArgument, "title and description are required for incident logging")
	}
	category := in.Category
	if category == "" {
		category = "habitat_disruption"
	} else if !validIncidentCategories[category] {
		return nil, status.Error(codes.InvalidArgument, "invalid incident category: must be scratching, unauthorized_perch, hairball_emergency, habitat_disruption, or medical_hold_breach")
	}
	severity := in.Severity
	if severity == "" {
		severity = "medium"
	}
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}

	resp, err := s.wfClient.CreateIncident(ctx, &workforcev1.CreateIncidentRequest{
		Title:            in.Title,
		Category:         category,
		InvolvedFelineID: in.InvolvedFelineID,
		InvolvedHumanID:  in.InvolvedHumanID,
		Severity:         severity,
		Description:      in.Description,
	})
	if err != nil {
		return nil, err
	}

	protoInc := resp.Incident
	if protoInc != nil {
		if s.meiliClient != nil {
			_ = s.meiliClient.IndexDocuments(ctx, "employee_knowledge", []map[string]interface{}{
				{
					"id":       protoInc.IncidentID,
					"category": "incident",
					"title":    protoInc.Title,
					"snippet":  fmt.Sprintf("Severity: %s | %s", protoInc.Severity, protoInc.Description),
				},
			})
		}
		if s.publisher != nil {
			env := envelope.NewEventEnvelope(
				fmt.Sprintf("evt-inc-%d", time.Now().UnixNano()),
				"events.workforce.incident_created.v1",
				"",
				envelope.IncidentCreatedPayload{
					IncidentID:       protoInc.IncidentID,
					Title:            protoInc.Title,
					Category:         protoInc.Category,
					InvolvedFelineID: protoInc.InvolvedFelineID,
					InvolvedHumanID:  protoInc.InvolvedHumanID,
					Severity:         protoInc.Severity,
					Status:           protoInc.Status,
					Description:      protoInc.Description,
					ReportedAt:       time.Now().UTC(),
				},
			)
			_ = s.publisher.Publish("events.workforce.incident_created.v1", env)
		}
	}

	return resp, nil
}

func (s *Server) ListIncidents(ctx context.Context, in *workforcev1.ListIncidentsRequest) (*workforcev1.ListIncidentsResponse, error) {
	if s.wfClient == nil {
		return &workforcev1.ListIncidentsResponse{}, nil
	}
	return s.wfClient.ListIncidents(ctx, in)
}

func (s *Server) UpdateIncidentStatus(ctx context.Context, in *workforcev1.UpdateIncidentStatusRequest) (*workforcev1.UpdateIncidentStatusResponse, error) {
	if in.IncidentID == "" || in.Status == "" {
		return nil, status.Error(codes.InvalidArgument, "incident_id and status are required")
	}
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.UpdateIncidentStatus(ctx, in)
}

// BroadcastPebbleIncidentAlert creates an urgent IT ticket and Pebble alert in ops-api for an incident.
func (s *Server) BroadcastPebbleIncidentAlert(ctx context.Context, in *internalbllv1.BroadcastPebbleIncidentAlertRequest) (*internalbllv1.BroadcastPebbleIncidentAlertResponse, error) {
	if s.opsClient == nil {
		return &internalbllv1.BroadcastPebbleIncidentAlertResponse{Status: "skipped"}, nil
	}
	sev := strings.ToLower(in.Severity)
	if sev == "" {
		sev = "high"
	}
	resp, err := s.opsClient.CreateTicket(ctx, &opsv1.CreateTicketRequest{
		Title:          fmt.Sprintf("[Feline Incident] %s", in.Title),
		Body:           fmt.Sprintf("Category: %s | Description: %s", in.Category, in.Description),
		Severity:       sev,
		Category:       "feline_safety",
		Source:         "workforce_incident",
		AuthorUsername: "safety-monitor",
		IsUrgent:       true,
	})
	if err != nil {
		return nil, err
	}
	ticketID := ""
	if resp.Ticket != nil {
		ticketID = resp.Ticket.TicketID
	}
	if s.meiliClient != nil {
		_ = s.meiliClient.IndexDocuments(ctx, "employee_knowledge", []map[string]interface{}{
			{
				"id":       in.IncidentID,
				"category": "incident",
				"title":    in.Title,
				"snippet":  fmt.Sprintf("Severity: %s | %s", sev, in.Description),
			},
		})
	}
	return &internalbllv1.BroadcastPebbleIncidentAlertResponse{
		TicketID: ticketID,
		Status:   "broadcasted",
	}, nil
}
