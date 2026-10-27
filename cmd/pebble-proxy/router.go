package main

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/versioning"
)

// NewRouter constructs the Chi HTTP router for Pebble Gateway Proxy.
func NewRouter(bllClient internalbllv1.InternalBLLServiceClient, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	h := NewGatewayHandler(bllClient, logger)

	r.Get("/healthz", h.Healthz)

	mountRoutes := func(prefix string) {
		p := func(path string) string {
			if prefix == "" {
				return path
			}
			return prefix + path
		}

		// Pebble Alerts & Physical ACK
		r.Get(p("/ops/pebble/alerts"), h.GetAlerts)
		r.Post(p("/ops/pebble/ack"), h.SubmitAck)

		// Incident Alert Ingestion
		r.Post(p("/ops/pebble/incidents/broadcast"), requireServiceAuth(h.HandleIncidentBroadcast))
		r.Post(p("/ops/pebble/alerts/incident"), requireServiceAuth(h.HandleIncidentBroadcast))

		// IT Helpdesk Tickets
		r.Get(p("/ops/tickets"), h.ListTickets)
		r.Post(p("/ops/tickets"), h.CreateTicket)
		r.Get(p("/ops/tickets/{id}"), h.GetTicket)
		r.Put(p("/ops/tickets/{id}"), h.UpdateTicket)
		r.Get(p("/ops/tickets/{id}/comments"), h.ListTicketComments)
		r.Post(p("/ops/tickets/{id}/comments"), h.AddTicketComment)

		// Push Notifications
		r.Post(p("/ops/notifications/push"), h.SendPushNotification)
		r.Post(p("/ops/notifications"), h.SendPushNotification)

		// Customer Support Tickets
		r.Get(p("/customer/support/tickets"), h.ListCustomerTickets)
		r.Post(p("/customer/support/tickets"), h.CreateCustomerTicket)
	}

	mountRoutes("")
	mountRoutes("/api/v1")

	return versioning.Middleware(versioning.DefaultConfig())(r)
}
