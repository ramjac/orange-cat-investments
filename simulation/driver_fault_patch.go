package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (d *Driver) testFaultScenarios(ctx context.Context) error {
	d.logger.Info("starting fault injection scenarios")

	client := &http.Client{Timeout: 30 * time.Second}

	bankURL := strings.TrimRight(d.cfg.BankURL, "/") + "/sim/faults"
	brokerURL := strings.TrimRight(d.cfg.BrokerURL, "/") + "/sim/faults"

	defer func() {
		for _, url := range []string{bankURL, brokerURL} {
			req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
			if err != nil {
				d.logger.Error("fault reset failed", "url", url, "error", err)
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				d.logger.Error("fault reset failed", "url", url, "error", err)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				d.logger.Error("fault reset left mock unhealthy", "url", url, "status", resp.StatusCode)
				continue
			}
			d.logger.Info("fault state reset", "url", url)
		}
	}()

	injectFault := func(url string, payload string) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBufferString(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("fault injection at %s returned status %d", url, resp.StatusCode)
		}
		return nil
	}

	if err := injectFault(bankURL, `{"outage":true}`); err != nil {
		return fmt.Errorf("inject bank outage failed: %w", err)
	}
	if err := injectFault(brokerURL, `{"outage":true}`); err != nil {
		return fmt.Errorf("inject broker outage failed: %w", err)
	}
	d.logger.Info("bank and broker outage faults injected")

	p, ok := d.registry.Get("cust-longterm-arthur")
	if !ok {
		for _, candidate := range d.registry.List() {
			if candidate.Type == "customer" {
				p = candidate
				break
			}
		}
	}
	if p == nil {
		return fmt.Errorf("no customer persona available for fault scenario")
	}

	personaCtx := WithPersona(ctx, p)
	if err := d.client.AuthenticatePersona(personaCtx, p); err != nil {
		return fmt.Errorf("authenticate persona for fault scenario: %w", err)
	}

	// Probe a customer BFF endpoint that depends on the broker. During the
	// outage it must answer with a structured JSON error (4xx/5xx), never a
	// plain 2xx success built on stale data.
	orderBody := map[string]any{
		"symbol":     "NVDA",
		"side":       "buy",
		"quantity":   1,
		"order_type": "market",
	}
	resp, body, err := d.client.DoCustomer(personaCtx, http.MethodPost, "/api/v1/customer/trading/orders", orderBody)
	if err != nil {
		return fmt.Errorf("BFF order request during outage failed at transport level: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode < 400 {
		return fmt.Errorf("BFF returned %d for an order during injected broker outage; fault did not propagate", resp.StatusCode)
	}
	var structured map[string]any
	if err := json.Unmarshal(body, &structured); err != nil {
		return fmt.Errorf("BFF error response during outage is not structured JSON: %s", string(body))
	}
	d.logger.Info("BFF returned structured error during injected outage", "status", resp.StatusCode)

	// Sweep the persona actions while the outage is active and record which
	// of them degrade. Actions unrelated to the mocks may legitimately
	// succeed, so only log them.
	for _, act := range p.Actions {
		res, actErr := act.Execute(personaCtx, d.client)
		if actErr != nil {
			d.logger.Warn("action errored during fault scenario", "action", act.Name, "error", actErr)
			continue
		}
		if res != nil && res.StatusCode >= 400 {
			d.logger.Info("action degraded as expected during fault scenario", "action", act.Name, "status", res.StatusCode)
		}
	}

	d.logger.Info("fault scenarios complete")
	return nil
}
