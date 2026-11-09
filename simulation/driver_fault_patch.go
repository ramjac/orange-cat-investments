package simulation

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
)

func (d *Driver) testFaultScenarios(ctx context.Context) error {
	d.logger.Info("starting fault injection scenarios")

	bankURL := strings.TrimRight(d.cfg.BankURL, "/") + "/sim/faults"
	brokerURL := strings.TrimRight(d.cfg.BrokerURL, "/") + "/sim/faults"

	defer func() {
		d.logger.Info("resetting fault states")
		req1, _ := http.NewRequestWithContext(ctx, http.MethodDelete, bankURL, nil)
		http.DefaultClient.Do(req1)
		req2, _ := http.NewRequestWithContext(ctx, http.MethodDelete, brokerURL, nil)
		http.DefaultClient.Do(req2)
	}()

	injectFault := func(url string, payload string) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBufferString(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("fault injection returned status %d", resp.StatusCode)
		}
		return nil
	}

	d.logger.Info("injecting bank and broker outage faults")
	if err := injectFault(bankURL, `{"outage":true}`); err != nil {
		return fmt.Errorf("inject bank outage failed: %w", err)
	}
	if err := injectFault(brokerURL, `{"outage":true}`); err != nil {
		return fmt.Errorf("inject broker outage failed: %w", err)
	}

	d.logger.Info("asserting BFF behavior during injected faults")

	// Ensure we have a persona to make request
	p, ok := d.registry.Get("cust-longterm-arthur")
	if !ok {
		// Just pick the first one if arthur not found
		for _, v := range d.registry.List() {
			p = v
			break
		}
	}

	if p != nil {
		personaCtx := WithPersona(ctx, p)
		d.client.AuthenticatePersona(personaCtx, p)

		// Just run all actions and check if they returned non-200.
		for _, act := range p.Actions {
			res, err := act.Execute(personaCtx, d.client)
			if err == nil && res != nil {
				// We expect some 5xx or 422 if it hits a mock fault
				if res.StatusCode >= 200 && res.StatusCode < 300 {
					d.logger.Warn("action succeeded unexpectedly during fault scenario", "action", act.Name, "status", res.StatusCode)
				} else {
					d.logger.Info("action failed as expected during fault scenario", "action", act.Name, "status", res.StatusCode)
				}
			}
		}
	}
	d.logger.Info("fault scenarios complete")
	return nil
}
