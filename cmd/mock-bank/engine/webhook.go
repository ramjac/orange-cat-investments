package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-bank/ledger"
)

// WebhookEvent represents an asynchronous event emitted by the mock bank.
type WebhookEvent struct {
	WebhookType string    `json:"webhook_type"`
	WebhookCode string    `json:"webhook_code"`
	TransferID  string    `json:"transfer_id"`
	AccountID   string    `json:"account_id"`
	EventType   string    `json:"event_type"` // e.g. "pending", "posted", "settled", "returned", "failed"
	Amount      float64   `json:"amount"`
	ReturnCode  string    `json:"return_code,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// WebhookDispatcher dispatches webhook notifications to OCI or external listeners.
type WebhookDispatcher struct {
	webhookURL string
	httpClient *http.Client
}

// NewWebhookDispatcher creates a new dispatcher with configured target URL.
func NewWebhookDispatcher(webhookURL string) *WebhookDispatcher {
	return &WebhookDispatcher{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Dispatch sends a webhook event if a webhook URL is configured.
func (d *WebhookDispatcher) Dispatch(ctx context.Context, transfer *ledger.ACHTransfer) {
	if d.webhookURL == "" {
		return
	}

	event := WebhookEvent{
		WebhookType: "TRANSFER",
		WebhookCode: "TRANSFER_EVENTS_UPDATE",
		TransferID:  transfer.TransferID,
		AccountID:   transfer.AccountID,
		EventType:   string(transfer.Status),
		Amount:      transfer.Amount,
		ReturnCode:  transfer.ReturnCode,
		Timestamp:   time.Now().UTC(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("[mock-bank] error marshaling webhook event: %v", err)
		return
	}

	go func() {
		reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, d.webhookURL, bytes.NewReader(payload))
		if err != nil {
			log.Printf("[mock-bank] error creating webhook request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mock-Bank-Webhook/1.0")

		resp, err := d.httpClient.Do(req)
		if err != nil {
			log.Printf("[mock-bank] failed to deliver webhook to %s: %v", d.webhookURL, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Printf("[mock-bank] webhook endpoint returned non-2xx status: %d", resp.StatusCode)
		}
	}()
}
