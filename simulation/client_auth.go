package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// AuthenticatePersona executes the automated authentication handshake for the specified persona.
func (c *Client) AuthenticatePersona(ctx context.Context, p *Persona) error {
	if p == nil {
		return errors.New("cannot authenticate nil persona")
	}
	personaCtx := WithPersona(ctx, p)
	if p.Type == "customer" || strings.HasPrefix(p.ID, "cust-") {
		return c.authenticateCustomer(personaCtx, p)
	}
	return c.authenticateEmployee(personaCtx, p)
}

func (c *Client) authenticateCustomer(ctx context.Context, p *Persona) error {
	email := p.AuthEmail()
	loginPayload := map[string]string{
		"customer_id": p.ID,
		"name":        p.Name,
		"email":       email,
		"persona":     p.ID,
	}

	endpoint := "/api/v1/customer/auth/login"
	resp, respBytes, err := c.DoCustomer(ctx, "POST", endpoint, loginPayload)
	if err != nil {
		return fmt.Errorf("customer authentication handshake failed for %s (%s): %w", p.Name, p.ID, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("customer authentication rejected for %s (%s): status %d body %s", p.Name, p.ID, resp.StatusCode, string(respBytes))
	}

	var res struct {
		Status    string `json:"status"`
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(respBytes, &res); err == nil && res.CSRFToken != "" {
		c.SetCSRFToken(p.ID, res.CSRFToken)
	}
	return nil
}

func (c *Client) authenticateEmployee(ctx context.Context, p *Persona) error {
	email := p.AuthEmail()
	role := p.AuthRole()

	loginPayload := map[string]string{
		"employee_id": p.ID,
		"name":        p.Name,
		"email":       email,
		"role":        role,
		"department":  p.Department,
	}

	endpoint := "/api/v1/employee/auth/login"
	resp, respBytes, err := c.Do(ctx, "POST", endpoint, loginPayload)
	if err == nil && resp.StatusCode == http.StatusOK {
		var res struct {
			Status    string `json:"status"`
			CSRFToken string `json:"csrf_token"`
		}
		if err := json.Unmarshal(respBytes, &res); err == nil && res.CSRFToken != "" {
			c.SetCSRFToken(p.ID, res.CSRFToken)
		}
		return nil
	}

	// In production mode, self-asserted login is disabled, so try Forgejo login
	if resp != nil && (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound) {
		forgejoUsername := strings.ToLower(strings.Split(p.Name, " ")[0])
		forgejoPayload := map[string]string{
			"username": forgejoUsername,
			"password": "password123",
		}
		fResp, fBytes, fErr := c.Do(ctx, "POST", "/api/v1/employee/auth/forgejo/login", forgejoPayload)
		if fErr == nil && fResp.StatusCode == http.StatusOK {
			var res struct {
				Status    string `json:"status"`
				CSRFToken string `json:"csrf_token"`
			}
			if err := json.Unmarshal(fBytes, &res); err == nil && res.CSRFToken != "" {
				c.SetCSRFToken(p.ID, res.CSRFToken)
			}
			return nil
		}
	}

	if err != nil {
		return fmt.Errorf("employee authentication handshake failed for %s (%s): %w", p.Name, p.ID, err)
	}
	return fmt.Errorf("employee authentication rejected for %s (%s): status %d body %s", p.Name, p.ID, resp.StatusCode, string(respBytes))
}

// LogoutPersona logs out the specified persona from their active BFF session.
func (c *Client) LogoutPersona(ctx context.Context, p *Persona) error {
	if p == nil {
		return errors.New("cannot logout nil persona")
	}
	personaCtx := WithPersona(ctx, p)
	if p.Type == "customer" || strings.HasPrefix(p.ID, "cust-") {
		endpoint := "/api/v1/customer/auth/logout"
		resp, _, err := c.DoCustomer(personaCtx, "POST", endpoint, nil)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("customer logout failed: status %d", resp.StatusCode)
		}
		return nil
	}

	endpoint := "/api/v1/employee/auth/logout"
	resp, _, err := c.Do(personaCtx, "POST", endpoint, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("employee logout failed: status %d", resp.StatusCode)
	}
	return nil
}
