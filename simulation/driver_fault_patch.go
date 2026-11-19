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
		resetCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, url := range []string{bankURL, brokerURL} {
			req, err := http.NewRequestWithContext(resetCtx, http.MethodDelete, url, nil)
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

	orderBody := map[string]any{
		"symbol":     "NVDA",
		"side":       "buy",
		"quantity":   1,
		"order_type": "market",
	}

	// Scenario 1: Broker & Bank outage -> BFF must return structured 5xx JSON error.
	if err := injectFault(bankURL, `{"outage":true}`); err != nil {
		return fmt.Errorf("inject bank outage failed: %w", err)
	}
	if err := injectFault(brokerURL, `{"outage":true}`); err != nil {
		return fmt.Errorf("inject broker outage failed: %w", err)
	}
	d.logger.Info("bank and broker outage faults injected")

	resp, body, err := d.client.DoCustomer(personaCtx, http.MethodPost, "/api/v1/customer/trading/orders", orderBody)
	if err != nil {
		return fmt.Errorf("BFF order request during outage failed at transport level: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode < 500 || resp.StatusCode >= 600 {
		return fmt.Errorf("BFF returned %d for an order during injected broker outage; expected 5xx", resp.StatusCode)
	}
	var outageErr map[string]any
	if err := json.Unmarshal(body, &outageErr); err != nil || outageErr["error"] == nil {
		return fmt.Errorf("BFF error response during outage is not structured JSON with error field: %s", string(body))
	}
	d.logger.Info("BFF returned structured 5xx error during injected outage", "status", resp.StatusCode)

	// Scenario 2: Halted symbol rejection -> BFF must return structured 422 JSON error.
	if err := injectFault(brokerURL, `{"outage":false,"halted_symbols":["NVDA"]}`); err != nil {
		return fmt.Errorf("inject broker halted_symbols failed: %w", err)
	}
	d.logger.Info("broker halted_symbols fault injected", "symbol", "NVDA")

	haltResp, haltBody, err := d.client.DoCustomer(personaCtx, http.MethodPost, "/api/v1/customer/trading/orders", orderBody)
	if err != nil {
		return fmt.Errorf("BFF order request during symbol halt failed at transport level: %w", err)
	}
	haltResp.Body.Close()

	if haltResp.StatusCode != http.StatusUnprocessableEntity {
		return fmt.Errorf("BFF returned %d for halted symbol order; expected 422", haltResp.StatusCode)
	}
	var haltErr map[string]any
	if err := json.Unmarshal(haltBody, &haltErr); err != nil || haltErr["error"] == nil {
		return fmt.Errorf("BFF error response during symbol halt is not structured JSON with error field: %s", string(haltBody))
	}
	d.logger.Info("BFF returned structured 422 error during halted symbol fault", "status", haltResp.StatusCode)

	// Sweep the persona actions while faults are active and record which
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
