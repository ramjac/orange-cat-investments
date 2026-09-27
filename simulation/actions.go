package simulation

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func (r *Registry) registerAllDefaultPersonas() {
	r.Register(r.createGarfieldPersona())
	r.Register(r.createBarnebyPersona())
	r.Register(r.createAlicePersona())
	r.Register(r.createBobPersona())
	r.Register(r.createCarolPersona())
	r.Register(r.createDavidPersona())
	r.Register(r.createElenaPersona())
	r.Register(r.createFrankPersona())
	r.Register(r.createArthurPersona())
	r.Register(r.createChloePersona())
}

// 1. Garfield - Chief Observation Officer (COO)
func (r *Registry) createGarfieldPersona() *Persona {
	return &Persona{
		ID:         "emp-feline-garfield",
		Name:       "Garfield",
		Type:       "feline",
		RoleTitle:  "Chief Observation Officer (COO)",
		Department: "Core Executive Observation Plane",
		Actions: []Action{
			{
				Name:        "garfield_morning_zoomies",
				Description: "Emits observation event during 3 AM morning zoomies",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/facilities/assets?asset_type=observation_perch"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-feline-garfield",
						PersonaName: "Garfield",
						ActionName:  "garfield_morning_zoomies",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Morning zoomies observation generated. Perches queried: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "garfield_perch_inspection",
				Description: "Inspects primary habitat zone perches and cushion comfort",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/facilities/assets?status=active"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-feline-garfield",
						PersonaName: "Garfield",
						ActionName:  "garfield_perch_inspection",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Primary perch inspection verified. Response length: %d", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "garfield_care_schedule_audit",
				Description: "Queries workforce care schedule to verify lasagna and snack disbursement times",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/workforce/care-schedules/emp-feline-garfield"
					resp, bytes, err := client.Do(ctx, true, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-feline-garfield",
						PersonaName: "Garfield",
						ActionName:  "garfield_care_schedule_audit",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Care schedule audited. Payload: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 2. Barneby - Senior Alpha Perch Analyst
func (r *Registry) createBarnebyPersona() *Persona {
	return &Persona{
		ID:         "emp-feline-barneby",
		Name:       "Barneby",
		Type:       "feline",
		RoleTitle:  "Senior Alpha Perch Analyst",
		Department: "Habitat Intelligence & Telemetry",
		Actions: []Action{
			{
				Name:        "barneby_sunbeam_tracking",
				Description: "Monitors secondary observation perch and tracks high-frequency sunbeam telemetry",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/facilities/assets?asset_type=smart_collar"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-feline-barneby",
						PersonaName: "Barneby",
						ActionName:  "barneby_sunbeam_tracking",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Sunbeam tracking telemetry synced. Smart collar assets read: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "barneby_care_schedule_review",
				Description: "Reviews feline care schedule and health assessment parameters",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/workforce/care-schedules/emp-feline-barneby"
					resp, bytes, err := client.Do(ctx, true, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-feline-barneby",
						PersonaName: "Barneby",
						ActionName:  "barneby_care_schedule_review",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Barneby care schedule checked. Payload: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 3. Alice Vance - Head of Human & Feline Resources
func (r *Registry) createAlicePersona() *Persona {
	return &Persona{
		ID:         "emp-human-alice",
		Name:       "Alice Vance",
		Type:       "human",
		RoleTitle:  "Head of Human & Feline Resources",
		Department: "Workforce Operations",
		Actions: []Action{
			{
				Name:        "alice_workforce_directory",
				Description: "Queries employee directory across human and feline staff",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/workforce/employees"
					resp, bytes, err := client.Do(ctx, true, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-alice",
						PersonaName: "Alice Vance",
						ActionName:  "alice_workforce_directory",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Retrieved workforce directory. Size: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "alice_trigger_onboarding_saga",
				Description: "Simulates Frappe HR webhook triggering automated employee onboarding saga",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/webhooks/frappe-hr"
					payload := map[string]any{
						"event":         "employee_created",
						"employee_id":   "emp-simulated-newhire",
						"employee_type": "human",
						"first_name":    "Nala",
						"last_name":     "Simba",
						"email":         "nala@oci.local",
						"role_title":    "Junior Habitat Specialist",
						"department":    "Workforce Operations",
						"status":        "onboarding",
					}
					resp, bytes, err := client.Do(ctx, true, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-alice",
						PersonaName: "Alice Vance",
						ActionName:  "alice_trigger_onboarding_saga",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Onboarding saga webhook result: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 4. Bob Builder - Lead Facilities & Edge Telemetry Engineer
func (r *Registry) createBobPersona() *Persona {
	return &Persona{
		ID:         "emp-human-bob",
		Name:       "Bob Builder",
		Type:       "human",
		RoleTitle:  "Lead Facilities & Edge Telemetry Engineer",
		Department: "Habitat Facilities & Infrastructure",
		Actions: []Action{
			{
				Name:        "bob_scan_edge_cameras",
				Description: "Scans active edge camera rigs across habitat zones",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/facilities/assets?asset_type=edge_camera"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-bob",
						PersonaName: "Bob Builder",
						ActionName:  "bob_scan_edge_cameras",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Scanned camera assets: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "bob_register_camera_rig",
				Description: "Registers a new 4K edge camera hardware asset",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/facilities/assets"
					payload := map[string]any{
						"serial_number": fmt.Sprintf("CAM-RIG-%d", time.Now().UnixNano()),
						"asset_type":    "edge_camera",
						"model":         "4K-Feline-Scope-V2",
						"zone_id":       "018f3a9a-1111-7000-8000-000000000001",
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-bob",
						PersonaName: "Bob Builder",
						ActionName:  "bob_register_camera_rig",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Registered camera asset response: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 5. Carol Danvers - Chief Investment Officer (CIO)
func (r *Registry) createCarolPersona() *Persona {
	return &Persona{
		ID:         "emp-human-carol",
		Name:       "Carol Danvers",
		Type:       "human",
		RoleTitle:  "Chief Investment Officer (CIO)",
		Department: "Core Investment Management",
		Actions: []Action{
			{
				Name:        "carol_query_brokerage_orders",
				Description: "Audits real-time execution log of brokerage trade orders",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/core-invest/brokerage/orders?limit=20"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-carol",
						PersonaName: "Carol Danvers",
						ActionName:  "carol_query_brokerage_orders",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Audited brokerage orders: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "carol_execute_automated_trade",
				Description: "Submits an algorithmic brokerage trade order triggered by feline activity",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/core-invest/brokerage/orders"
					payload := map[string]any{
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
						"broker_name":  "Interactive Brokers",
						"symbol":       "AAPL",
						"side":         "buy",
						"quantity":     100.0,
						"price":        225.50,
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-carol",
						PersonaName: "Carol Danvers",
						ActionName:  "carol_execute_automated_trade",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Executed trade order: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 6. David Quant - Feline Behavioral Data Scientist
func (r *Registry) createDavidPersona() *Persona {
	return &Persona{
		ID:         "emp-human-david",
		Name:       "David Quant",
		Type:       "human",
		RoleTitle:  "Feline Behavioral Data Scientist",
		Department: "Quantitative Analytics & AI",
		Actions: []Action{
			{
				Name:        "david_register_stream",
				Description: "Registers RTSP/WebRTC camera stream for computer vision activity classification",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/core-invest/streams"
					payload := map[string]any{
						"camera_asset_id": "018f3a9a-3333-7000-8000-000000000003",
						"stream_url":      "rtsp://edge-cam-01.local/live",
						"protocol":        "rtsp",
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-david",
						PersonaName: "David Quant",
						ActionName:  "david_register_stream",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Registered camera stream: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "david_initiate_backtest",
				Description: "Runs historical backtesting model comparing feline activity vs equity market performance",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/core-invest/backtesting/runs"
					payload := map[string]any{
						"strategy_id": "018f3a9a-4444-7000-8000-000000000004",
						"start_date":  time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
						"end_date":    time.Now().Format(time.RFC3339),
						"parameters":  `{"confidence_threshold":0.95,"multiplier":1.5}`,
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-david",
						PersonaName: "David Quant",
						ActionName:  "david_initiate_backtest",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Initiated backtest run: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 7. Dr. Elena Rostova - Chief Veterinary Officer & Habitat Care Specialist
func (r *Registry) createElenaPersona() *Persona {
	return &Persona{
		ID:         "emp-human-elena",
		Name:       "Dr. Elena Rostova",
		Type:       "human",
		RoleTitle:  "Chief Veterinary Officer & Habitat Care Specialist",
		Department: "Feline Health & Habitat Welfare",
		Actions: []Action{
			{
				Name:        "elena_update_garfield_care_schedule",
				Description: "Updates Garfield's dietary plan and feeding schedule in workforce domain",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/workforce/care-schedules/emp-feline-garfield"
					payload := map[string]any{
						"dietary_plan":           "Low-Sodium Organic Salmon & Grain-Free Lasagna Kibble",
						"feeding_times":          []string{"06:00", "12:00", "18:00"},
						"special_medical_needs":  "Routine dental checkup scheduled",
						"preferred_perch_zone":   "Alpha Sunbeam Lounge",
						"emergency_medical_hold": false,
					}
					resp, bytes, err := client.Do(ctx, true, "PUT", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-elena",
						PersonaName: "Dr. Elena Rostova",
						ActionName:  "elena_update_garfield_care_schedule",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Updated care schedule: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "elena_toggle_medical_hold",
				Description: "Verifies emergency medical hold toggle functionality",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/workforce/care-schedules/emp-feline-garfield/medical-hold"
					payload := map[string]any{
						"emergency_medical_hold": false,
					}
					resp, bytes, err := client.Do(ctx, true, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-elena",
						PersonaName: "Dr. Elena Rostova",
						ActionName:  "elena_toggle_medical_hold",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Medical hold checked: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 8. Frank Operations - Platform Security & K8s Infrastructure Lead
func (r *Registry) createFrankPersona() *Persona {
	return &Persona{
		ID:         "emp-human-frank",
		Name:       "Frank Operations",
		Type:       "human",
		RoleTitle:  "Platform Security & K8s Infrastructure Lead",
		Department: "Systems Platform & DevSecOps",
		Actions: []Action{
			{
				Name:        "frank_monitor_it_tickets",
				Description: "Monitors IT helpdesk ticket queue for cluster security or pod alerts",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/ops/tickets"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-frank",
						PersonaName: "Frank Operations",
						ActionName:  "frank_monitor_it_tickets",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Monitored IT tickets response: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "frank_query_pebble_alerts",
				Description: "Queries Pebble companion watch AppMessage alert payloads",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/ops/pebble/alerts"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-frank",
						PersonaName: "Frank Operations",
						ActionName:  "frank_query_pebble_alerts",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Pebble alert payloads retrieved: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "frank_create_it_ticket",
				Description: "Creates an IT infrastructure ticket for intermediate CA key rotation audit",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/ops/tickets"
					payload := map[string]any{
						"forgejo_repo":    "oci/infrastructure",
						"title":           "Routine Intermediate CA Key Rotation Audit",
						"body":            "Cert-manager renewed oci-intermediate-ca. Verify SPIFFE ID mTLS dynamic reloads.",
						"author_username": "emp-human-frank",
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-frank",
						PersonaName: "Frank Operations",
						ActionName:  "frank_create_it_ticket",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Created IT ticket: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 9. Arthur Pendelton - Long-Term Value Investor (Customer)
func (r *Registry) createArthurPersona() *Persona {
	return &Persona{
		ID:         "cust-longterm-arthur",
		Name:       "Arthur Pendelton",
		Type:       "customer",
		RoleTitle:  "Long-Term Value Investor",
		Department: "Retail Investor Community",
		Actions: []Action{
			{
				Name:        "arthur_review_performance_reports",
				Description: "Audits historical quantitative backtest performance and strategy returns",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/core-invest/backtesting/runs"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-longterm-arthur",
						PersonaName: "Arthur Pendelton",
						ActionName:  "arthur_review_performance_reports",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Reviewed strategy backtest reports. Payload size: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "arthur_deposit_investment_capital",
				Description: "Executes steady capital deposit order into OCI Alpha Growth Fund",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/core-invest/brokerage/orders"
					payload := map[string]any{
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
						"broker_name":  "Interactive Brokers",
						"symbol":       "SPY",
						"side":         "buy",
						"quantity":     25,
						"price":        510.50,
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-longterm-arthur",
						PersonaName: "Arthur Pendelton",
						ActionName:  "arthur_deposit_investment_capital",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Deposited capital via trade order: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "arthur_monitor_feline_stream",
				Description: "Passively checks optical camera streams to verify feline habitat comfort",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/core-invest/streams"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-longterm-arthur",
						PersonaName: "Arthur Pendelton",
						ActionName:  "arthur_monitor_feline_stream",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Monitored habitat observation streams. Stream count: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 10. Chloe Spark - Short-Term Momentum Trader (Customer)
func (r *Registry) createChloePersona() *Persona {
	return &Persona{
		ID:         "cust-active-chloe",
		Name:       "Chloe Spark",
		Type:       "customer",
		RoleTitle:  "Momentum Alpha Trader",
		Department: "Retail Investor Community",
		Actions: []Action{
			{
				Name:        "chloe_scan_activity_streams",
				Description: "Scans real-time camera streams for high-activity feline behavioral telemetry",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/core-invest/streams"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-active-chloe",
						PersonaName: "Chloe Spark",
						ActionName:  "chloe_scan_activity_streams",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Scanned real-time camera streams for zoomies signals: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "chloe_momentum_buy_order",
				Description: "Executes aggressive momentum buy order on feline zoomies activity spike",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/core-invest/brokerage/orders"
					payload := map[string]any{
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
						"broker_name":  "Interactive Brokers",
						"symbol":       "NVDA",
						"side":         "buy",
						"quantity":     50,
						"price":        125.75,
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-active-chloe",
						PersonaName: "Chloe Spark",
						ActionName:  "chloe_momentum_buy_order",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Executed momentum buy order on activity spike: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "chloe_take_profit_sell_order",
				Description: "Executes tactical take-profit sell order as cat activity returns to baseline",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/core-invest/brokerage/orders"
					payload := map[string]any{
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
						"broker_name":  "Interactive Brokers",
						"symbol":       "NVDA",
						"side":         "sell",
						"quantity":     50,
						"price":        131.25,
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-active-chloe",
						PersonaName: "Chloe Spark",
						ActionName:  "chloe_take_profit_sell_order",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Executed take-profit sell order on post-zoomies cooldown: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "chloe_audit_execution_log",
				Description: "Audits real-time trade order execution logs and settlement status",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/core-invest/brokerage/orders"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-active-chloe",
						PersonaName: "Chloe Spark",
						ActionName:  "chloe_audit_execution_log",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Audited real-time trade executions: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}
