package ops

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"github.com/orange-cat-investments/oci/pkg/pebble"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	bllClient internalbllv1.InternalBLLServiceClient
}

func NewHandler(bllClient internalbllv1.InternalBLLServiceClient) *Handler {
	return &Handler{bllClient: bllClient}
}

func NewHandlerWithBLL(bllClient internalbllv1.InternalBLLServiceClient) *Handler {
	return &Handler{bllClient: bllClient}
}

func (h *Handler) SetBLLClient(client internalbllv1.InternalBLLServiceClient) {
	h.bllClient = client
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Pebble Watch Endpoints
	mux.HandleFunc("GET /api/v1/ops/pebble/alerts", h.GetPebbleAlerts)
	mux.HandleFunc("POST /api/v1/ops/pebble/ack", h.SubmitPebbleAck)

	// General IT Helpdesk Tickets
	mux.HandleFunc("GET /api/v1/ops/tickets", h.ListTickets)
	mux.HandleFunc("POST /api/v1/ops/tickets", h.CreateTicket)
	mux.HandleFunc("GET /api/v1/ops/tickets/{id}", h.GetTicket)
	mux.HandleFunc("PUT /api/v1/ops/tickets/{id}", h.UpdateTicket)
	mux.HandleFunc("GET /api/v1/ops/tickets/{id}/comments", h.ListTicketComments)
	mux.HandleFunc("POST /api/v1/ops/tickets/{id}/comments", h.AddTicketComment)

	// Push Notifications
	mux.HandleFunc("POST /api/v1/ops/notifications/push", h.SendPushNotification)
	mux.HandleFunc("POST /api/v1/ops/notifications", h.SendPushNotification)

	// Customer Bug & Feedback Intake
	mux.HandleFunc("GET /api/v1/customer/support/tickets", h.ListCustomerTickets)
	mux.HandleFunc("POST /api/v1/customer/support/tickets", h.CreateCustomerTicket)
	mux.HandleFunc("GET /api/v1/customer/support/tickets/{id}", h.GetCustomerTicket)
	mux.HandleFunc("GET /api/v1/customer/support/tickets/{id}/comments", h.ListCustomerTicketComments)
	mux.HandleFunc("POST /api/v1/customer/support/tickets/{id}/comments", h.AddCustomerTicketComment)

	// Direct path aliases (without /api/v1)
	mux.HandleFunc("GET /ops/pebble/alerts", h.GetPebbleAlerts)
	mux.HandleFunc("POST /ops/pebble/ack", h.SubmitPebbleAck)
	mux.HandleFunc("GET /ops/tickets", h.ListTickets)
	mux.HandleFunc("POST /ops/tickets", h.CreateTicket)
	mux.HandleFunc("GET /ops/tickets/{id}", h.GetTicket)
	mux.HandleFunc("PUT /ops/tickets/{id}", h.UpdateTicket)
	mux.HandleFunc("GET /ops/tickets/{id}/comments", h.ListTicketComments)
	mux.HandleFunc("POST /ops/tickets/{id}/comments", h.AddTicketComment)
	mux.HandleFunc("POST /ops/notifications/push", h.SendPushNotification)
	mux.HandleFunc("POST /ops/notifications", h.SendPushNotification)
	mux.HandleFunc("GET /customer/support/tickets", h.ListCustomerTickets)
	mux.HandleFunc("POST /customer/support/tickets", h.CreateCustomerTicket)
	mux.HandleFunc("GET /customer/support/tickets/{id}", h.GetCustomerTicket)
	mux.HandleFunc("GET /customer/support/tickets/{id}/comments", h.ListCustomerTicketComments)
	mux.HandleFunc("POST /customer/support/tickets/{id}/comments", h.AddCustomerTicketComment)
}

func (h *Handler) GetPebbleAlerts(w http.ResponseWriter, r *http.Request) {
	if h.bllClient == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

	resp, err := h.bllClient.ListAlerts(r.Context(), &opsv1.ListAlertsRequest{UnacknowledgedOnly: true})
	if err != nil {
		writeOpsGRPCError(w, "ListAlerts", err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	alerts := resp.Alerts
	if alerts == nil {
		alerts = []*opsv1.PebbleAlert{}
	}
	_ = json.NewEncoder(w).Encode(alerts)
}

func (h *Handler) SubmitPebbleAck(w http.ResponseWriter, r *http.Request) {
	var req pebble.PebbleAckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if h.bllClient == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

	resp, err := h.bllClient.AcknowledgeAlert(r.Context(), &opsv1.AcknowledgeAlertRequest{
		AlertID:        req.TicketID,
		AcknowledgedBy: req.AcknowledgedBy,
		ButtonID:       int32(req.ButtonID),
		WatchID:        req.WatchID,
		DeviceID:       req.DeviceID,
	})
	if err != nil {
		writeOpsGRPCError(w, "AcknowledgeAlert", err, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func containsSensitiveOpsDetails(msg string) bool {
	lower := strings.ToLower(msg)
	for _, needle := range []string{
		"dial tcp",
		"connection refused",
		"transport:",
		"user=",
		"database=",
		"password=",
		"sqlstate",
		"failed to connect to",
		"rpc error:",
	} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func sanitizeOpsError(httpStatus int, msg string) string {
	trimmed := strings.TrimSpace(msg)
	if trimmed != "" && !containsSensitiveOpsDetails(trimmed) {
		return trimmed
	}
	switch httpStatus {
	case http.StatusServiceUnavailable:
		return "ops service temporarily unavailable"
	case http.StatusGatewayTimeout:
		return "ops service request timed out"
	case http.StatusNotFound:
		return "resource not found"
	case http.StatusBadRequest:
		return "invalid request parameters"
	default:
		return "internal ops service error"
	}
}

func writeOpsGRPCError(w http.ResponseWriter, op string, err error, fallbackStatus int) {
	st, ok := status.FromError(err)
	code := status.Code(err)
	rawMsg := err.Error()
	if ok && st.Message() != "" {
		rawMsg = st.Message()
	}

	httpStatus := fallbackStatus
	switch code {
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		httpStatus = http.StatusGatewayTimeout
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	case codes.Internal, codes.DataLoss:
		httpStatus = http.StatusInternalServerError
	}

	if httpStatus >= 500 || code == codes.Unavailable || code == codes.DeadlineExceeded || code == codes.Internal || code == codes.Unknown {
		slog.Error("internal-bll ops gRPC call failed",
			"downstream", "internal-bll",
			"operation", op,
			"grpc_code", code.String(),
			"http_status", httpStatus,
			"error", err,
		)
	}

	http.Error(w, sanitizeOpsError(httpStatus, rawMsg), httpStatus)
}
