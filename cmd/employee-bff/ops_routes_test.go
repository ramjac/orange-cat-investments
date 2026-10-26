package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	opshandler "github.com/orange-cat-investments/oci/internal/handler/ops"
	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
)

func TestRegisterProtectedOpsRoutes(t *testing.T) {
	t.Setenv("ENV", "test")

	store := NewSessionStore()

	// Sessions with different roles
	engSess, _ := store.Create("emp-eng-01", "eng@oci.local", "Eng User", "engineering", "Engineering")
	hrSess, _ := store.Create("emp-hr-01", "hr@oci.local", "HR User", "hr", "Human Resources")
	vetSess, _ := store.Create("emp-vet-01", "vet@oci.local", "Vet User", "veterinary", "Veterinary Medicine")

	repo := opssvc.NewMockRepository()
	svc := opssvc.NewService(repo)
	opsH := opshandler.NewHandler(svc)

	mux := http.NewServeMux()
	registerProtectedOpsRoutes(mux, opsH, store)

	csrfToken := "valid-csrf-token-ops"

	addAuthAndCSRF := func(req *http.Request, sess *EmployeeSession, withCSRF bool) {
		if sess != nil {
			req.AddCookie(&http.Cookie{Name: "employee_session", Value: sess.SessionID})
		}
		if withCSRF {
			req.AddCookie(&http.Cookie{Name: "employee_csrf_token", Value: csrfToken})
			req.Header.Set("X-CSRF-Token", csrfToken)
		}
	}

	t.Run("Pebble Alerts endpoints authentication and role check", func(t *testing.T) {
		// 1. Unauthenticated GET /api/v1/ops/pebble/alerts -> 401
		req := httptest.NewRequest("GET", "/api/v1/ops/pebble/alerts", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for unauthenticated GET pebble alerts, got %d", rec.Code)
		}

		// 2. HR role (not in opsRole: ops, admin, engineering, facilities, staff) -> 403
		req = httptest.NewRequest("GET", "/api/v1/ops/pebble/alerts", nil)
		addAuthAndCSRF(req, hrSess, false)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for HR role on pebble alerts, got %d", rec.Code)
		}

		// 3. Engineering role (in opsRole) -> 200
		req = httptest.NewRequest("GET", "/api/v1/ops/pebble/alerts", nil)
		addAuthAndCSRF(req, engSess, false)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for Eng role on pebble alerts, got %d", rec.Code)
		}

		// 4. Unversioned alias /ops/pebble/alerts -> 200
		req = httptest.NewRequest("GET", "/ops/pebble/alerts", nil)
		addAuthAndCSRF(req, engSess, false)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for unversioned alias pebble alerts, got %d", rec.Code)
		}

		// 5. Pebble Ack POST missing CSRF -> 403
		body := `{"ticket_id":"ticket-001","acknowledged_by":"emp-eng-01"}`
		req = httptest.NewRequest("POST", "/api/v1/ops/pebble/ack", strings.NewReader(body))
		addAuthAndCSRF(req, engSess, false)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for POST pebble ack without CSRF, got %d", rec.Code)
		}
	})

	t.Run("IT Helpdesk Tickets endpoints", func(t *testing.T) {
		// 1. anyEmployeeRole (e.g., HR) can list tickets
		req := httptest.NewRequest("GET", "/api/v1/ops/tickets", nil)
		addAuthAndCSRF(req, hrSess, false)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for HR listing tickets, got %d", rec.Code)
		}

		// 2. anyEmployeeRole (e.g., Vet) can create ticket with CSRF
		ticketBody := `{"title":"Laptop screen flickering","description":"Display flickers when opening lid","severity":"medium","category":"hardware","author_username":"emp-vet-01"}`
		req = httptest.NewRequest("POST", "/api/v1/ops/tickets", strings.NewReader(ticketBody))
		addAuthAndCSRF(req, vetSess, true)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 for creating ticket, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var created opssvc.ITTicket
		_ = json.Unmarshal(rec.Body.Bytes(), &created)
		ticketID := created.TicketID

		// 3. GET ticket detail by ID
		req = httptest.NewRequest("GET", "/api/v1/ops/tickets/"+ticketID, nil)
		addAuthAndCSRF(req, vetSess, false)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for GET ticket by ID, got %d", rec.Code)
		}

		// 4. PUT ticket update with CSRF
		updateBody := `{"state":"in_progress","assignee_username":"emp-eng-01"}`
		req = httptest.NewRequest("PUT", "/api/v1/ops/tickets/"+ticketID, strings.NewReader(updateBody))
		addAuthAndCSRF(req, engSess, true)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for updating ticket, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		// 5. Ticket comments GET and POST
		req = httptest.NewRequest("GET", "/api/v1/ops/tickets/"+ticketID+"/comments", nil)
		addAuthAndCSRF(req, hrSess, false)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for listing comments, got %d", rec.Code)
		}

		commentBody := `{"comment":"Working on replacing display cable","author_username":"emp-eng-01","is_internal":true}`
		req = httptest.NewRequest("POST", "/api/v1/ops/tickets/"+ticketID+"/comments", strings.NewReader(commentBody))
		addAuthAndCSRF(req, engSess, true)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201 for adding comment, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("Push Notifications endpoints admin/ops RBAC restriction", func(t *testing.T) {
		pushBody := `{"topic":"alerts","title":"System Maintenance","message":"Downtime scheduled for 10 PM"}`

		// 1. HR role (not in adminOpsRole: ops, admin, engineering) -> 403
		req := httptest.NewRequest("POST", "/api/v1/ops/notifications/push", strings.NewReader(pushBody))
		addAuthAndCSRF(req, hrSess, true)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for HR on push notifications, got %d", rec.Code)
		}

		// 2. Vet role -> 403
		req = httptest.NewRequest("POST", "/api/v1/ops/notifications", strings.NewReader(pushBody))
		addAuthAndCSRF(req, vetSess, true)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for Vet on push notifications, got %d", rec.Code)
		}

		// 3. Engineering role (in adminOpsRole) -> 200
		req = httptest.NewRequest("POST", "/api/v1/ops/notifications/push", strings.NewReader(pushBody))
		addAuthAndCSRF(req, engSess, true)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for Eng sending push notification, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("Customer Support Tickets intake endpoints", func(t *testing.T) {
		// 1. GET /api/v1/customer/support/tickets with X-Customer-ID
		req := httptest.NewRequest("GET", "/api/v1/customer/support/tickets", nil)
		req.Header.Set("X-Customer-ID", "cust-longterm-arthur")
		addAuthAndCSRF(req, hrSess, false)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for customer tickets with customer header, got %d", rec.Code)
		}

		// 2. POST /api/v1/customer/support/tickets with CSRF and customer ID
		custTicketBody := `{"customer_id":"cust-100","title":"Unable to login","body":"Invalid credentials error"}`
		req = httptest.NewRequest("POST", "/api/v1/customer/support/tickets", strings.NewReader(custTicketBody))
		addAuthAndCSRF(req, vetSess, true)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 for creating customer ticket, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var created opssvc.ITTicket
		_ = json.Unmarshal(rec.Body.Bytes(), &created)
		custTicketID := created.TicketID

		// 3. GET customer ticket detail by ID
		req = httptest.NewRequest("GET", "/api/v1/customer/support/tickets/"+custTicketID, nil)
		addAuthAndCSRF(req, vetSess, false)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for GET customer ticket, got %d", rec.Code)
		}

		// 4. GET & POST customer ticket comments
		req = httptest.NewRequest("GET", "/api/v1/customer/support/tickets/"+custTicketID+"/comments", nil)
		addAuthAndCSRF(req, hrSess, false)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for customer ticket comments, got %d", rec.Code)
		}

		custCommentBody := `{"comment":"We are investigating this issue.","author_username":"cust-100"}`
		req = httptest.NewRequest("POST", "/api/v1/customer/support/tickets/"+custTicketID+"/comments", strings.NewReader(custCommentBody))
		addAuthAndCSRF(req, vetSess, true)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201 for adding customer ticket comment, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})
}
