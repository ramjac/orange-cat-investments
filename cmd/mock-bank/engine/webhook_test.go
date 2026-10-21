package engine

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-bank/ledger"
)

func TestNewWebhookDispatcher(t *testing.T) {
	url := "http://localhost:8080/webhook"
	d := NewWebhookDispatcher(url)

	if d == nil {
		t.Fatal("expected NewWebhookDispatcher to return non-nil")
	}
	if d.webhookURL != url {
		t.Errorf("expected webhookURL to be %s, got %s", url, d.webhookURL)
	}
	if d.httpClient == nil {
		t.Fatal("expected httpClient to be non-nil")
	}
	if d.httpClient.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", d.httpClient.Timeout)
	}
}

func TestWebhookDispatcher_Dispatch_EmptyURL(t *testing.T) {
	d := NewWebhookDispatcher("")
	transfer := &ledger.ACHTransfer{
		TransferID: "tr_12345",
		AccountID:  "acc_67890",
		Amount:     100.0,
		Status:     ledger.TransferStatusPending,
	}

	// Dispatch should do nothing when webhookURL is empty
	d.Dispatch(context.Background(), transfer)
}

func TestWebhookDispatcher_Dispatch_Success(t *testing.T) {
	received := make(chan struct {
		contentType string
		userAgent   string
		event       WebhookEvent
	}, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var evt WebhookEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		received <- struct {
			contentType string
			userAgent   string
			event       WebhookEvent
		}{
			contentType: r.Header.Get("Content-Type"),
			userAgent:   r.Header.Get("User-Agent"),
			event:       evt,
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	dispatcher := NewWebhookDispatcher(ts.URL)
	transfer := &ledger.ACHTransfer{
		TransferID: "tr_test_1",
		AccountID:  "acc_test_1",
		Amount:     250.75,
		Status:     ledger.TransferStatusSettled,
		ReturnCode: "R01",
	}

	dispatcher.Dispatch(context.Background(), transfer)

	select {
	case res := <-received:
		if res.contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", res.contentType)
		}
		if res.userAgent != "Mock-Bank-Webhook/1.0" {
			t.Errorf("expected User-Agent Mock-Bank-Webhook/1.0, got %s", res.userAgent)
		}
		evt := res.event
		if evt.WebhookType != "TRANSFER" {
			t.Errorf("expected WebhookType TRANSFER, got %s", evt.WebhookType)
		}
		if evt.WebhookCode != "TRANSFER_EVENTS_UPDATE" {
			t.Errorf("expected WebhookCode TRANSFER_EVENTS_UPDATE, got %s", evt.WebhookCode)
		}
		if evt.TransferID != transfer.TransferID {
			t.Errorf("expected TransferID %s, got %s", transfer.TransferID, evt.TransferID)
		}
		if evt.AccountID != transfer.AccountID {
			t.Errorf("expected AccountID %s, got %s", transfer.AccountID, evt.AccountID)
		}
		if evt.EventType != string(transfer.Status) {
			t.Errorf("expected EventType %s, got %s", transfer.Status, evt.EventType)
		}
		if evt.Amount != transfer.Amount {
			t.Errorf("expected Amount %f, got %f", transfer.Amount, evt.Amount)
		}
		if evt.ReturnCode != transfer.ReturnCode {
			t.Errorf("expected ReturnCode %s, got %s", transfer.ReturnCode, evt.ReturnCode)
		}
		if evt.Timestamp.IsZero() {
			t.Errorf("expected non-zero Timestamp")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook dispatch")
	}
}

func TestWebhookDispatcher_Dispatch_Non2xxStatus(t *testing.T) {
	done := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		close(done)
	}))
	defer ts.Close()

	dispatcher := NewWebhookDispatcher(ts.URL)
	transfer := &ledger.ACHTransfer{
		TransferID: "tr_err",
		AccountID:  "acc_err",
		Amount:     50.0,
		Status:     ledger.TransferStatusFailed,
	}

	dispatcher.Dispatch(context.Background(), transfer)

	select {
	case <-done:
		// Delivery attempted and failed with 500 status code as expected
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook dispatch with error status")
	}
}

func TestWebhookDispatcher_Dispatch_UnreachableURL(t *testing.T) {
	// Use an invalid/unreachable address that fails immediately
	dispatcher := NewWebhookDispatcher("http://127.0.0.1:1")
	transfer := &ledger.ACHTransfer{
		TransferID: "tr_unreachable",
		AccountID:  "acc_unreachable",
		Amount:     10.0,
		Status:     ledger.TransferStatusPending,
	}

	dispatcher.Dispatch(context.Background(), transfer)
	// Allow background goroutine to execute and handle transport error safely
	time.Sleep(100 * time.Millisecond)
}
