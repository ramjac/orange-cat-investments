package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// RandomAction selects a random action for the given persona based on its ActivityLevel.
// If the persona is nil, has no actions, or the probability roll exceeds ActivityLevel, an empty Action is returned.
func RandomAction(p *Persona) Action {
	if p == nil || len(p.Actions) == 0 {
		return Action{}
	}
	r := rand.Float64()
	if r < p.ActivityLevel {
		return p.Actions[rand.Intn(len(p.Actions))]
	}
	return Action{}
}

func (r *Registry) registerAllDefaultPersonas() {
	r.Register(r.createGarfieldPersona())
	r.Register(r.createBarnebyPersona())
	r.Register(r.createBobPersona())
	r.Register(r.createAlicePersona())
	r.Register(r.createCarolPersona())
	r.Register(r.createDavidPersona())
	r.Register(r.createElenaPersona())
	r.Register(r.createFrankPersona())
	r.Register(r.createArthurPersona())
	r.Register(r.createChloePersona())
	r.Register(r.createRickPersona())
	r.Register(r.createElisePersona())
	r.Register(r.createMarcusPersona())
}

// 1. Garfield - Chief Observation Officer (COO)
func (r *Registry) createGarfieldPersona() *Persona {
	return &Persona{
		ID:            "emp-feline-garfield",
		Name:          "Garfield",
		Type:          "feline",
		RoleTitle:     "Chief Observation Officer (COO)",
		Department:    "Core Executive Observation Plane",
		Role:          "admin",
		Email:         "garfield@oci.local",
		ActivityLevel: 1.0,
		Actions: []Action{
			r.createGarfieldMorningZoomiesAction(),
			r.createGarfieldPerchInspectionAction(),
			r.createGarfieldCareScheduleAuditAction(),
			r.createGarfieldSleepInHallwayAction(),
		},
	}
}

func (r *Registry) createGarfieldMorningZoomiesAction() Action {
	return Action{
		Name:        "garfield_morning_zoomies",
		Description: "Emits observation event during 3 AM morning zoomies via optical camera stream",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/facilities/assets?asset_type=edge_camera"
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
				Details:     fmt.Sprintf("Morning zoomies observation generated via optical camera stream. Cameras queried: %d bytes", len(bytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

func (r *Registry) createGarfieldPerchInspectionAction() Action {
	return Action{
		Name:        "garfield_perch_inspection",
		Description: "Inspects primary habitat active hardware assets",
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
				Details:     fmt.Sprintf("Primary habitat active hardware inspection verified. Response length: %d", len(bytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

func (r *Registry) createGarfieldCareScheduleAuditAction() Action {
	return Action{
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
	}
}

func (r *Registry) createGarfieldSleepInHallwayAction() Action {
	return Action{
		Name:        "garfield_sleep_in_hallway",
		Description: "Sleeps in the middle of a hallway, with a 1/3 chance of tripping a human employee, sending Garfield to the vet and triggering a workplace injury report",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/facilities/assets?status=active"
			resp, _, err := client.Do(ctx, false, "GET", endpoint, nil)
			if err != nil {
				return nil, err
			}

			tripped := rand.Intn(3) == 0
			if !tripped {
				return &ActionResult{
					PersonaID:   "emp-feline-garfield",
					PersonaName: "Garfield",
					ActionName:  "garfield_sleep_in_hallway",
					Endpoint:    endpoint,
					Success:     resp.StatusCode == http.StatusOK,
					StatusCode:  resp.StatusCode,
					Details:     "Garfield slept peacefully in the middle of the hallway. Nobody tripped.",
					Timestamp:   time.Now().UTC(),
				}, nil
			}

			vetEndpoint := "/api/v1/workforce/care-schedules/emp-feline-garfield/medical-hold"
			vetResp, vetBytes, vetErr := client.Do(ctx, true, "POST", vetEndpoint, map[string]any{
				"emergency_medical_hold": true,
			})
			if vetErr != nil {
				return nil, fmt.Errorf("failed to toggle vet medical hold: %w", vetErr)
			}

			injuryEndpoint := "/ops/tickets"
			injuryPayload := map[string]any{
				"forgejo_repo":    "oci/workforce-safety",
				"title":           "Workplace Injury Report: Tripped over Garfield in Hallway B",
				"body":            "Human employee tripped over Garfield while sleeping in hallway. Garfield admitted to vet for checkup.",
				"author_username": "emp-human-bob",
			}
			injResp, injBytes, injErr := client.Do(ctx, false, "POST", injuryEndpoint, injuryPayload)
			if injErr != nil {
				return nil, fmt.Errorf("failed to create workplace injury ticket: %w", injErr)
			}

			return &ActionResult{
				PersonaID:   "emp-feline-garfield",
				PersonaName: "Garfield",
				ActionName:  "garfield_sleep_in_hallway",
				Endpoint:    injuryEndpoint,
				Success:     (vetResp.StatusCode == http.StatusOK) && (injResp.StatusCode == http.StatusCreated || injResp.StatusCode == http.StatusOK),
				StatusCode:  injResp.StatusCode,
				Details:     fmt.Sprintf("TRIPPED! Garfield sent to vet (Hold status %d: %s). Workplace injury ticket created (Status %d: %s)", vetResp.StatusCode, string(vetBytes), injResp.StatusCode, string(injBytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

// 2. Barneby - Senior Alpha Perch Analyst
func (r *Registry) createBarnebyPersona() *Persona {
	return &Persona{
		ID:            "emp-feline-barneby",
		Name:          "Barneby",
		Type:          "feline",
		RoleTitle:     "Senior Alpha Perch Analyst",
		Department:    "Habitat Intelligence & Telemetry",
		Role:          "staff",
		Email:         "barneby@oci.local",
		ActivityLevel: 1.0,
		Actions: []Action{
			r.createBarnebySunbeamTrackingAction(),
			r.createBarnebyCareScheduleReviewAction(),
			r.createBarnebySlapWaterGlassAction(),
		},
	}
}

func (r *Registry) createBarnebySunbeamTrackingAction() Action {
	return Action{
		Name:        "barneby_sunbeam_tracking",
		Description: "Monitors habitat optical camera streams and tracks high-frequency sunbeam alignment",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/facilities/assets?asset_type=edge_camera"
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
				Details:     fmt.Sprintf("Sunbeam tracking telemetry synced via optical camera stream. Cameras read: %d bytes", len(bytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

func (r *Registry) createBarnebyCareScheduleReviewAction() Action {
	return Action{
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
	}
}

func (r *Registry) createBarnebySlapWaterGlassAction() Action {
	return Action{
		Name:        "barneby_slap_water_glass",
		Description: "Slaps a glass of water off table, triggering facilities alert ticket, Pebble watch notification, and mobile field app cleanup log sync",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			// Step 1: Trigger operations alert ticket for water spill
			maintEndpoint := "/ops/tickets"
			maintPayload := map[string]any{
				"forgejo_repo":    "oci/facilities",
				"title":           "Water Spill Incident: Glass slapped off desk in Sector 4",
				"body":            "Barneby slapped a full glass of water onto active server control console desk. Immediate spill response required.",
				"author_username": "emp-feline-barneby",
			}
			maintResp, maintBytes, err := client.Do(ctx, false, "POST", maintEndpoint, maintPayload)
			if err != nil {
				return nil, fmt.Errorf("failed to create maintenance ticket: %w", err)
			}

			var ticketData struct {
				TicketID string `json:"ticket_id"`
			}
			_ = json.Unmarshal(maintBytes, &ticketData)
			ticketID := ticketData.TicketID
			if ticketID == "" {
				ticketID = "018e0000-0000-7000-8000-000000000003"
			}

			// Step 2: Query Pebble companion alerts
			pebbleAlertEndpoint := "/ops/pebble/alerts"
			pebbleResp, pebbleBytes, pebbleErr := client.Do(ctx, false, "GET", pebbleAlertEndpoint, nil)
			if pebbleErr != nil {
				return nil, fmt.Errorf("failed to query Pebble watch alerts: %w", pebbleErr)
			}

			// Step 3: Send Pebble watch ACK for facilities alert
			pebbleAckEndpoint := "/ops/pebble/ack"
			pebbleAckPayload := map[string]any{
				"ticket_id":       ticketID,
				"acknowledged_by": "emp-human-bob",
			}
			ackResp, ackBytes, ackErr := client.Do(ctx, false, "POST", pebbleAckEndpoint, pebbleAckPayload)
			if ackErr != nil {
				return nil, fmt.Errorf("failed to acknowledge Pebble watch alert: %w", ackErr)
			}

			// Step 4: Facilities employee completes maintenance work and syncs via Flutter mobile app
			syncEndpoint := "/facilities/sync/maintenance-logs"
			syncPayload := []map[string]any{
				{
					"asset_id":        "018e0000-0000-7000-8000-000000000001",
					"qr_code_scanned": "QR-ZONE-4-DESK-01",
					"action_taken":    "Cleaned up water spill, wiped control console electronics, placed spill prevention mug coaster.",
					"notes":           "Barneby supervised cleanup from upper perch.",
					"created_at":      time.Now().UTC().Format(time.RFC3339),
				},
			}
			syncResp, syncBytes, syncErr := client.Do(ctx, false, "POST", syncEndpoint, syncPayload)
			if syncErr != nil {
				return nil, fmt.Errorf("failed to sync mobile app maintenance log: %w", syncErr)
			}

			success := (maintResp.StatusCode == http.StatusCreated || maintResp.StatusCode == http.StatusOK) &&
				pebbleResp.StatusCode == http.StatusOK &&
				(ackResp.StatusCode == http.StatusOK || ackResp.StatusCode == http.StatusCreated) &&
				(syncResp.StatusCode == http.StatusOK || syncResp.StatusCode == http.StatusCreated)

			return &ActionResult{
				PersonaID:   "emp-feline-barneby",
				PersonaName: "Barneby",
				ActionName:  "barneby_slap_water_glass",
				Endpoint:    syncEndpoint,
				Success:     success,
				StatusCode:  syncResp.StatusCode,
				Details:     fmt.Sprintf("WATER GLASS SLAPPED! Maint Ticket: %d (%s) -> Pebble Watch Alert: %d (%d bytes) -> Pebble ACK: %d (%s) -> Mobile App Log Sync: %d (%s)", maintResp.StatusCode, string(maintBytes), pebbleResp.StatusCode, len(pebbleBytes), ackResp.StatusCode, string(ackBytes), syncResp.StatusCode, string(syncBytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

// 3. Alice Vance - Head of Human & Feline Resources
func (r *Registry) createAlicePersona() *Persona {
	return &Persona{
		ID:            "emp-human-alice",
		Name:          "Alice Vance",
		Type:          "human",
		RoleTitle:     "Head of Human & Feline Resources",
		Department:    "Workforce Operations",
		Role:          "hr",
		Email:         "alice.vance@oci.local",
		ActivityLevel: 1.0,
		Actions: []Action{
			r.createAliceWorkforceDirectoryAction(),
			r.createAliceTriggerOnboardingSagaAction(),
			r.createAliceReviewLeaveRequestsAction(),
			r.createAliceApproveTimesheetAction(),
			r.createAliceProcessPayrollEntryAction(),
			r.createAliceOffboardEmployeeAction(),
			r.createAliceAssignLaptopAction(),
			r.createAlicePublishOnboardingDocAction(),
			r.createAliceGenerateExecutiveReportAction(),
		},
	}
}

func (r *Registry) createAliceWorkforceDirectoryAction() Action {
	return Action{
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
	}
}

func (r *Registry) createAliceTriggerOnboardingSagaAction() Action {
	return Action{
		Name:        "alice_trigger_onboarding_saga",
		Description: "Simulates Frappe HR webhook triggering automated employee onboarding saga",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/api/v1/webhooks/frappe-hr"
			payload := map[string]any{
				"event":         "employee_created",
				"employee_id":   "018f3a9a-e999-7000-8000-000000000099",
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
	}
}

func (r *Registry) createAliceReviewLeaveRequestsAction() Action {
	return Action{
		Name:        "alice_review_leave_requests",
		Description: "Audits pending staff leave requests and feline catnip break compliance",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/api/v1/workforce/leave-requests"
			resp, bytes, err := client.Do(ctx, true, "GET", endpoint, nil)
			if err != nil {
				return nil, err
			}
			return &ActionResult{
				PersonaID:   "emp-human-alice",
				PersonaName: "Alice Vance",
				ActionName:  "alice_review_leave_requests",
				Endpoint:    endpoint,
				Success:     resp.StatusCode == http.StatusOK,
				StatusCode:  resp.StatusCode,
				Details:     fmt.Sprintf("Audited leave requests & feline catnip breaks. Payload: %d bytes", len(bytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

func (r *Registry) createAliceApproveTimesheetAction() Action {
	return Action{
		Name:        "alice_approve_timesheet",
		Description: "Reviews and approves pending workforce timesheets for Frappe HR synchronization",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			listEndpoint := "/api/v1/workforce/timesheets?status=submitted"
			_, bytes, err := client.Do(ctx, true, "GET", listEndpoint, nil)
			if err != nil {
				return nil, err
			}

			var listResp struct {
				Items []struct {
					TimesheetID string `json:"timesheet_id"`
					EmployeeID  string `json:"employee_id"`
				} `json:"items"`
			}
			_ = json.Unmarshal(bytes, &listResp)

			var timesheetID string
			if len(listResp.Items) > 0 {
				timesheetID = listResp.Items[0].TimesheetID
			} else {
				// Fallback: If no submitted timesheet exists yet, submit one first
				today := time.Now().UTC().Format("2006-01-02")
				_, createBytes, createErr := client.Do(ctx, true, "POST", "/api/v1/workforce/timesheets", map[string]any{
					"employee_id":   "emp-human-bob",
					"shift_date":    today,
					"start_time":    fmt.Sprintf("%sT08:00:00Z", today),
					"end_time":      fmt.Sprintf("%sT16:30:00Z", today),
					"break_minutes": 30,
					"task_code":     "habitat_maintenance",
					"billable":      true,
				})
				if createErr != nil {
					return nil, createErr
				}
				var created struct {
					TimesheetID string `json:"timesheet_id"`
				}
				_ = json.Unmarshal(createBytes, &created)
				timesheetID = created.TimesheetID
			}

			if timesheetID == "" {
				return nil, errors.New("no pending timesheet found to approve")
			}

			approveEndpoint := fmt.Sprintf("/api/v1/workforce/timesheets/%s/status", timesheetID)
			payload := map[string]any{
				"status":      "approved",
				"approver_id": "emp-human-alice",
			}
			approveResp, approveBytes, approveErr := client.Do(ctx, true, "PUT", approveEndpoint, payload)
			if approveErr != nil {
				return nil, approveErr
			}

			return &ActionResult{
				PersonaID:   "emp-human-alice",
				PersonaName: "Alice Vance",
				ActionName:  "alice_approve_timesheet",
				Endpoint:    approveEndpoint,
				Success:     approveResp.StatusCode == http.StatusOK,
				StatusCode:  approveResp.StatusCode,
				Details:     fmt.Sprintf("Approved workforce timesheet %s: %s", timesheetID, string(approveBytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

func (r *Registry) createAliceAssignLaptopAction() Action {
	return Action{
		Name:        "alice_assign_laptop_to_rick",
		Description: "Provisions a new MacBook Pro laptop asset in hardware inventory assigned to new hire Rick",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/facilities/assets"
			payload := map[string]any{
				"serial_number": fmt.Sprintf("MBP-RICK-%d", time.Now().UnixNano()),
				"asset_type":    "laptop",
				"model":         "MacBook Pro 16-inch M3 Max",
				"zone_id":       "018f3a9a-1111-7000-8000-000000000001",
				"assigned_to":   "emp-human-rick",
				"notes":         "Assigned to Rick Newhire during onboarding checklist execution",
			}
			resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
			if err != nil {
				return nil, err
			}
			return &ActionResult{
				PersonaID:   "emp-human-alice",
				PersonaName: "Alice Vance",
				ActionName:  "alice_assign_laptop_to_rick",
				Endpoint:    endpoint,
				Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
				StatusCode:  resp.StatusCode,
				Details:     fmt.Sprintf("Assigned new hire laptop asset to Rick: %s", string(bytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

func (r *Registry) createAlicePublishOnboardingDocAction() Action {
	return Action{
		Name:        "alice_publish_onboarding_doc",
		Description: "Publishes employee handbook in Nextcloud and notifies new hire Rick via Nextcloud Chat",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			docEndpoint := "/nextcloud/api/v1/documents"
			docPayload := map[string]any{
				"title":      "OCI New Employee & Feline Care Onboarding Handbook",
				"content":    "Comprehensive guide on OCI culture, catnip safety compliance, and IT asset allocation.",
				"author":     "emp-human-alice",
				"share_with": []string{"emp-human-rick", "emp-human-frank"},
			}
			docResp, docBytes, docErr := client.Do(ctx, false, "POST", docEndpoint, docPayload)
			if docErr != nil {
				return nil, fmt.Errorf("failed to create Nextcloud document: %w", docErr)
			}

			chatEndpoint := "/nextcloud/api/v1/chat/messages"
			chatPayload := map[string]any{
				"sender":    "emp-human-alice",
				"recipient": "emp-human-rick",
				"room":      "onboarding-general",
				"message":   "Welcome to OCI, Rick! I published the Onboarding Handbook in Nextcloud and assigned your laptop asset.",
			}
			chatResp, chatBytes, chatErr := client.Do(ctx, false, "POST", chatEndpoint, chatPayload)
			if chatErr != nil {
				return nil, fmt.Errorf("failed to send Nextcloud chat message: %w", chatErr)
			}

			success := (docResp.StatusCode == http.StatusCreated || docResp.StatusCode == http.StatusOK) &&
				(chatResp.StatusCode == http.StatusCreated || chatResp.StatusCode == http.StatusOK)

			return &ActionResult{
				PersonaID:   "emp-human-alice",
				PersonaName: "Alice Vance",
				ActionName:  "alice_publish_onboarding_doc",
				Endpoint:    chatEndpoint,
				Success:     success,
				StatusCode:  chatResp.StatusCode,
				Details:     fmt.Sprintf("ONBOARDING DOC & CHAT SENT! Doc created: %d (%s) -> Nextcloud Chat: %d (%s)", docResp.StatusCode, string(docBytes), chatResp.StatusCode, string(chatBytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

func (r *Registry) createAliceGenerateExecutiveReportAction() Action {
	return Action{
		Name:        "alice_generate_executive_report",
		Description: "Generates monthly executive management summary aggregating ERPNext financials and workforce KPIs, archiving to Nextcloud WebDAV",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/api/v1/workforce/reports/monthly/generate"
			payload := map[string]any{
				"period":       "2026-10",
				"generated_by": "emp-human-alice",
			}
			resp, bytes, err := client.Do(ctx, true, "POST", endpoint, payload)
			if err != nil {
				return nil, err
			}
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				return &ActionResult{
					PersonaID:   "emp-human-alice",
					PersonaName: "Alice Vance",
					ActionName:  "alice_generate_executive_report",
					Endpoint:    endpoint,
					Success:     false,
					StatusCode:  resp.StatusCode,
					Details:     fmt.Sprintf("Failed to generate executive report: %s", string(bytes)),
					Timestamp:   time.Now().UTC(),
				}, nil
			}

			// Verify report is listed in historical reports
			listEndpoint := "/api/v1/workforce/reports/monthly"
			listResp, listBytes, listErr := client.Do(ctx, true, "GET", listEndpoint, nil)
			if listErr != nil {
				return nil, listErr
			}

			success := (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated) &&
				listResp.StatusCode == http.StatusOK

			return &ActionResult{
				PersonaID:   "emp-human-alice",
				PersonaName: "Alice Vance",
				ActionName:  "alice_generate_executive_report",
				Endpoint:    endpoint,
				Success:     success,
				StatusCode:  resp.StatusCode,
				Details:     fmt.Sprintf("Generated and archived monthly executive report: %s (History verified: %d bytes)", string(bytes), len(listBytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

func (r *Registry) createAliceProcessPayrollEntryAction() Action {
	return Action{
		Name:        "alice_process_payroll_entry",
		Description: "Submits monthly payroll run for processing and salary slip generation",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/api/v1/workforce/payroll/run"
			today := time.Now().UTC().Format("2006-01-02")
			startOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
			payload := map[string]any{
				"company":           "Orange Cat Investments",
				"posting_date":      today,
				"start_date":        startOfMonth,
				"end_date":          today,
				"payroll_frequency": "Monthly",
				"currency":          "USD",
			}
			resp, bytes, err := client.Do(ctx, true, "POST", endpoint, payload)
			if err != nil {
				return nil, err
			}
			return &ActionResult{
				PersonaID:   "emp-human-alice",
				PersonaName: "Alice Vance",
				ActionName:  "alice_process_payroll_entry",
				Endpoint:    endpoint,
				Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
				StatusCode:  resp.StatusCode,
				Details:     fmt.Sprintf("Processed payroll entry run: %s", string(bytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

func (r *Registry) createAliceOffboardEmployeeAction() Action {
	return Action{
		Name:        "alice_offboard_employee",
		Description: "Executes automated deprovisioning engine for separating employee across VCS, Nextcloud, and Snipe-IT",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/api/v1/workforce/employees/018f3a9a-e999-7000-8000-000000000099/offboard"
			payload := map[string]any{
				"employee_id":   "018f3a9a-e999-7000-8000-000000000099",
				"reason":        "Seasonal habitat monitoring contract concluded",
				"target_status": "separated",
			}
			resp, bytes, err := client.Do(ctx, true, "POST", endpoint, payload)
			if err != nil {
				return nil, err
			}
			return &ActionResult{
				PersonaID:   "emp-human-alice",
				PersonaName: "Alice Vance",
				ActionName:  "alice_offboard_employee",
				Endpoint:    endpoint,
				Success:     resp.StatusCode == http.StatusOK,
				StatusCode:  resp.StatusCode,
				Details:     fmt.Sprintf("Executed offboarding engine: %s", string(bytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

// 4. Bob Builder - Lead Facilities & Edge Telemetry Engineer
func (r *Registry) createBobPersona() *Persona {
	return &Persona{
		ID:            "emp-human-bob",
		Name:          "Bob Builder",
		Type:          "human",
		RoleTitle:     "Lead Facilities & Edge Telemetry Engineer",
		Department:    "Habitat Facilities & Infrastructure",
		Role:          "facilities",
		Email:         "bob.builder@oci.local",
		ActivityLevel: 1.0,
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
			{
				Name:        "bob_verify_pebble_wearables",
				Description: "Verifies assigned Pebble watch hardware assets for on-call alerting",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/facilities/assets?asset_type=pebble_watch"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-bob",
						PersonaName: "Bob Builder",
						ActionName:  "bob_verify_pebble_wearables",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Verified Pebble watch wearable assets: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			r.createBobLogDailyTimesheetAction(),
			r.createBobViewPaystubsAction(),
			r.createBobReviewCalendarAction(),
		},
	}
}

func (r *Registry) createBobLogDailyTimesheetAction() Action {
	return Action{
		Name:        "bob_log_daily_timesheet",
		Description: "Submits daily 8-hour shift timesheet for habitat maintenance",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/api/v1/workforce/timesheets"
			today := time.Now().UTC().Format("2006-01-02")
			payload := map[string]any{
				"employee_id":   "emp-human-bob",
				"shift_date":    today,
				"start_time":    fmt.Sprintf("%sT08:00:00Z", today),
				"end_time":      fmt.Sprintf("%sT16:30:00Z", today),
				"break_minutes": 30,
				"task_code":     "habitat_maintenance",
				"billable":      true,
				"notes":         "Daily habitat perimeter maintenance, 4K camera rig alignment & HVAC telemetry inspection",
			}
			resp, bytes, err := client.Do(ctx, true, "POST", endpoint, payload)
			if err != nil {
				return nil, err
			}
			return &ActionResult{
				PersonaID:   "emp-human-bob",
				PersonaName: "Bob Builder",
				ActionName:  "bob_log_daily_timesheet",
				Endpoint:    endpoint,
				Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
				StatusCode:  resp.StatusCode,
				Details:     fmt.Sprintf("Submitted daily timesheet shift: %s", string(bytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

func (r *Registry) createBobViewPaystubsAction() Action {
	return Action{
		Name:        "bob_view_paystubs",
		Description: "Queries historical salary slips and itemized paystubs for compensation review",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/api/v1/workforce/payroll/slips?employee_id=emp-human-bob"
			resp, bytes, err := client.Do(ctx, true, "GET", endpoint, nil)
			if err != nil {
				return nil, err
			}
			return &ActionResult{
				PersonaID:   "emp-human-bob",
				PersonaName: "Bob Builder",
				ActionName:  "bob_view_paystubs",
				Endpoint:    endpoint,
				Success:     resp.StatusCode == http.StatusOK,
				StatusCode:  resp.StatusCode,
				Details:     fmt.Sprintf("Retrieved Bob paystubs/salary slips: %s", string(bytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

func (r *Registry) createBobReviewCalendarAction() Action {
	return Action{
		Name:        "bob_review_maintenance_calendar",
		Description: "Reviews scheduled habitat maintenance windows and shift rotations in the shared operations calendar",
		Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
			endpoint := "/api/v1/workforce/calendar/events?calendar=Habitat%20Maintenance%20Windows"
			resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
			if err != nil {
				return nil, err
			}
			return &ActionResult{
				PersonaID:   "emp-human-bob",
				PersonaName: "Bob Builder",
				ActionName:  "bob_review_maintenance_calendar",
				Endpoint:    endpoint,
				Success:     resp.StatusCode == http.StatusOK,
				StatusCode:  resp.StatusCode,
				Details:     fmt.Sprintf("Retrieved maintenance calendar events: %s", string(bytes)),
				Timestamp:   time.Now().UTC(),
			}, nil
		},
	}
}

// 5. Carol Danvers - Chief Investment Officer (CIO)
func (r *Registry) createCarolPersona() *Persona {
	return &Persona{
		ID:            "emp-human-carol",
		Name:          "Carol Danvers",
		Type:          "human",
		RoleTitle:     "Chief Investment Officer (CIO)",
		Department:    "Core Investment Management",
		Role:          "investment",
		Email:         "carol.danvers@oci.local",
		ActivityLevel: 1.0,
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
						"quantity":     10.0,
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
		ID:            "emp-human-david",
		Name:          "David Quant",
		Type:          "human",
		RoleTitle:     "Feline Behavioral Data Scientist",
		Department:    "Quantitative Analytics & AI",
		Role:          "investment",
		Email:         "david.quant@oci.local",
		ActivityLevel: 1.0,
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
			{
				Name:        "david_fetch_market_candlesticks",
				Description: "Fetches historical OHLCV candlestick bars from mock broker data engine to train quantitative pricing models",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/v2/stocks/NVDA/bars?timeframe=1Min&limit=50"
					resp, bytes, err := client.DoBroker(ctx, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-david",
						PersonaName: "David Quant",
						ActionName:  "david_fetch_market_candlesticks",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Fetched market candlestick bars for NVDA: %d bytes", len(bytes)),
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
		ID:            "emp-human-elena",
		Name:          "Dr. Elena Rostova",
		Type:          "human",
		RoleTitle:     "Chief Veterinary Officer & Habitat Care Specialist",
		Department:    "Feline Health & Habitat Welfare",
		Role:          "veterinary",
		Email:         "elena.rostova@oci.local",
		ActivityLevel: 1.0,
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
			{
				Name:        "elena_review_health_assessments",
				Description: "Audits scheduled feline health assessments and records clinical observation metrics",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/workforce/review-cycles?review_type=feline_health_assessment"
					resp, bytes, err := client.Do(ctx, true, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-elena",
						PersonaName: "Dr. Elena Rostova",
						ActionName:  "elena_review_health_assessments",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Audited feline health assessments: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "elena_conduct_wellness_audit",
				Description: "Conducts periodic feline clinical wellness audit with habitat optimization recommendations",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/workforce/review-cycles/018e0000-0000-7000-8000-000000000021/vet-audit"
					payload := map[string]any{
						"clinical_auditor_id":     "018f3a9a-e004-7000-8000-000000000004",
						"wellness_audit_score":    4.90,
						"habitat_recommendations": "Recalibrate perch climb angle to reduce joint strain and install secondary scratching post",
					}
					resp, bytes, err := client.Do(ctx, true, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-elena",
						PersonaName: "Dr. Elena Rostova",
						ActionName:  "elena_conduct_wellness_audit",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Recorded clinical wellness audit: %s", string(bytes)),
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
		ID:            "emp-human-frank",
		Name:          "Frank Operations",
		Type:          "human",
		RoleTitle:     "Platform Security & K8s Infrastructure Lead",
		Department:    "Systems Platform & DevSecOps",
		Role:          "ops",
		Email:         "frank.ops@oci.local",
		ActivityLevel: 1.0,
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
			{
				Name:        "frank_publish_security_doc",
				Description: "Writes security architecture standard document in Nextcloud and notifies team via Nextcloud Chat",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					docEndpoint := "/nextcloud/api/v1/documents"
					docPayload := map[string]any{
						"title":      "Zero-Trust Microservices & SPIFFE mTLS Security Standard",
						"content":    "Architecture decision record on zero-static certificates, cert-manager dynamic reloaders, and Valkey session security.",
						"author":     "emp-human-frank",
						"share_with": []string{"emp-human-elise", "emp-human-rick", "emp-human-bob"},
					}
					docResp, docBytes, docErr := client.Do(ctx, false, "POST", docEndpoint, docPayload)
					if docErr != nil {
						return nil, fmt.Errorf("failed to write security doc in Nextcloud: %w", docErr)
					}

					chatEndpoint := "/nextcloud/api/v1/chat/messages"
					chatPayload := map[string]any{
						"sender":    "emp-human-frank",
						"recipient": "all-devs",
						"room":      "devsecops-alerts",
						"message":   "Published the updated SPIFFE mTLS and Zero-Trust standard document in Nextcloud. Please review!",
					}
					chatResp, chatBytes, chatErr := client.Do(ctx, false, "POST", chatEndpoint, chatPayload)
					if chatErr != nil {
						return nil, fmt.Errorf("failed to send Nextcloud chat message: %w", chatErr)
					}

					success := (docResp.StatusCode == http.StatusCreated || docResp.StatusCode == http.StatusOK) &&
						(chatResp.StatusCode == http.StatusCreated || chatResp.StatusCode == http.StatusOK)

					return &ActionResult{
						PersonaID:   "emp-human-frank",
						PersonaName: "Frank Operations",
						ActionName:  "frank_publish_security_doc",
						Endpoint:    chatEndpoint,
						Success:     success,
						StatusCode:  chatResp.StatusCode,
						Details:     fmt.Sprintf("SECURITY DOC & CHAT SENT! Doc created: %d (%s) -> Nextcloud Chat: %d (%s)", docResp.StatusCode, string(docBytes), chatResp.StatusCode, string(chatBytes)),
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
		ID:            "cust-longterm-arthur",
		Name:          "Arthur Pendelton",
		Type:          "customer",
		RoleTitle:     "Long-Term Value Investor",
		Department:    "Retail Investor Community",
		Role:          "customer",
		Email:         "arthur@oci.local",
		ActivityLevel: 1.0,
		Actions: []Action{
			{
				Name:        "arthur_review_performance_reports",
				Description: "Audits historical quantitative backtest performance and strategy returns",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/trade-logs"
					resp, bytes, err := client.DoCustomer(ctx, "GET", endpoint, nil)
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
					endpoint := "/api/v1/customer/trading/orders"
					payload := map[string]any{
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
						"broker_name":  "Interactive Brokers",
						"symbol":       "SPY",
						"side":         "buy",
						"quantity":     10.0,
						"price":        510.50,
					}
					resp, bytes, err := client.DoCustomer(ctx, "POST", endpoint, payload)
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
					endpoint := "/api/v1/customer/streams"
					resp, bytes, err := client.DoCustomer(ctx, "GET", endpoint, nil)
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
			{
				Name:        "arthur_schedule_recurring_ach_deposit",
				Description: "Schedules automated monthly recurring ACH deposit from linked Chase checking account",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/deposits"
					payload := map[string]any{
						"customer_id":  "cust-longterm-arthur",
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
						"amount_usd":   500.00,
						"frequency":    "monthly",
						"bank_account": "Chase Checking (****4821)",
					}
					resp, bytes, err := client.DoCustomer(ctx, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-longterm-arthur",
						PersonaName: "Arthur Pendelton",
						ActionName:  "arthur_schedule_recurring_ach_deposit",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Scheduled recurring ACH deposit: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "arthur_audit_linked_bank_accounts",
				Description: "Audits linked commercial bank accounts and available balances via Customer BFF",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/bank-accounts?customer_id=cust-longterm-arthur"
					resp, bytes, err := client.DoCustomer(ctx, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-longterm-arthur",
						PersonaName: "Arthur Pendelton",
						ActionName:  "arthur_audit_linked_bank_accounts",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Audited linked bank accounts: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "arthur_verify_portfolio_holdings",
				Description: "Verifies equity asset allocation and real-time portfolio market value via Customer BFF",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/portfolio?customer_id=cust-longterm-arthur"
					resp, bytes, err := client.DoCustomer(ctx, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-longterm-arthur",
						PersonaName: "Arthur Pendelton",
						ActionName:  "arthur_verify_portfolio_holdings",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Verified portfolio holdings and valuation: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "arthur_link_new_bank_account",
				Description: "Initiates Plaid Link token workflow with linked financial institution via Bank API",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/link/token/create"
					payload := map[string]any{
						"client_user_id": "cust-longterm-arthur",
						"client_name":    "Arthur Pendelton",
					}
					resp, bytes, err := client.DoBank(ctx, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-longterm-arthur",
						PersonaName: "Arthur Pendelton",
						ActionName:  "arthur_link_new_bank_account",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Initiated Plaid Link token session: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "arthur_submit_customer_support_ticket",
				Description: "Submits customer support inquiry and verifies sanitized comment thread retrieval",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/support/tickets"
					payload := map[string]any{
						"customer_id": "cust-longterm-arthur",
						"category":    "Trading",
						"title":       "Delayed Fill Confirmation for SPY Order",
						"body":        "I noticed an order submitted at market open took longer than anticipated to confirm fill status.",
					}
					resp, bytes, err := client.DoCustomer(ctx, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
						return &ActionResult{
							PersonaID:   "cust-longterm-arthur",
							PersonaName: "Arthur Pendelton",
							ActionName:  "arthur_submit_customer_support_ticket",
							Endpoint:    endpoint,
							Success:     false,
							StatusCode:  resp.StatusCode,
							Details:     fmt.Sprintf("Failed to submit support ticket: %s", string(bytes)),
							Timestamp:   time.Now().UTC(),
						}, nil
					}

					var created struct {
						TicketID string `json:"ticket_id"`
					}
					_ = json.Unmarshal(bytes, &created)

					// Retrieve sanitized comments thread
					commentsEndpoint := fmt.Sprintf("/api/v1/customer/support/tickets/%s/comments", created.TicketID)
					commentsResp, commentsBytes, commentsErr := client.DoCustomer(ctx, "GET", commentsEndpoint, nil)
					if commentsErr != nil {
						return nil, commentsErr
					}

					success := (resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK) &&
						commentsResp.StatusCode == http.StatusOK

					return &ActionResult{
						PersonaID:   "cust-longterm-arthur",
						PersonaName: "Arthur Pendelton",
						ActionName:  "arthur_submit_customer_support_ticket",
						Endpoint:    commentsEndpoint,
						Success:     success,
						StatusCode:  commentsResp.StatusCode,
						Details:     fmt.Sprintf("Created support ticket %s, retrieved sanitized comments: %s", created.TicketID, string(commentsBytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "arthur_self_register_onboarding",
				Description: "Registers new retail investor account with NIST-compliant password and linked bank account",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/auth/register"
					payload := map[string]any{
						"full_name":           "Arthur Pendelton Jr",
						"name":                "Arthur Pendelton Jr",
						"email":               "arthur.jr@oci.local",
						"password":            "NIST_Compliant#2026!Strong",
						"bank_account":        "Chase Premier Checking (****4821)",
						"initial_deposit_usd": 5000.00,
					}
					resp, bytes, err := client.DoCustomer(ctx, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					success := resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK
					return &ActionResult{
						PersonaID:   "cust-longterm-arthur",
						PersonaName: "Arthur Pendelton",
						ActionName:  "arthur_self_register_onboarding",
						Endpoint:    endpoint,
						Success:     success,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Registered new customer account with zero-balance portfolio: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "arthur_generate_monthly_investment_statement",
				Description: "Generates pure Go PDF 1.4 monthly portfolio summary statement and verifies document download",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/statements/generate"
					payload := map[string]any{
						"customer_id":  "018e0000-0000-7000-8000-000000000099",
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
						"type":         "monthly",
						"period_month": 3,
						"period_year":  2026,
					}
					resp, bytes, err := client.DoCustomer(ctx, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
						return &ActionResult{
							PersonaID:   "cust-longterm-arthur",
							PersonaName: "Arthur Pendelton",
							ActionName:  "arthur_generate_monthly_investment_statement",
							Endpoint:    endpoint,
							Success:     false,
							StatusCode:  resp.StatusCode,
							Details:     fmt.Sprintf("Failed to generate monthly statement: %s", string(bytes)),
							Timestamp:   time.Now().UTC(),
						}, nil
					}

					var created struct {
						StatementID string `json:"statement_id"`
					}
					_ = json.Unmarshal(bytes, &created)

					// Verify PDF binary download
					downloadEndpoint := fmt.Sprintf("/api/v1/customer/statements/%s/download", created.StatementID)
					dlResp, dlBytes, dlErr := client.DoCustomer(ctx, "GET", downloadEndpoint, nil)
					if dlErr != nil {
						return nil, dlErr
					}
					isPDF := dlResp.StatusCode == http.StatusOK && len(dlBytes) > 4 && string(dlBytes[:5]) == "%PDF-"

					return &ActionResult{
						PersonaID:   "cust-longterm-arthur",
						PersonaName: "Arthur Pendelton",
						ActionName:  "arthur_generate_monthly_investment_statement",
						Endpoint:    downloadEndpoint,
						Success:     isPDF,
						StatusCode:  dlResp.StatusCode,
						Details:     fmt.Sprintf("Generated and downloaded pure Go PDF 1.4 statement (ID: %s, %d bytes)", created.StatementID, len(dlBytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "arthur_generate_tax_form_1099b",
				Description: "Generates IRS Form 1099-B tax return report and audits statement list repository",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/statements/generate"
					payload := map[string]any{
						"customer_id":  "018e0000-0000-7000-8000-000000000099",
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
						"type":         "1099_b",
						"tax_year":     2025,
					}
					resp, bytes, err := client.DoCustomer(ctx, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
						return &ActionResult{
							PersonaID:   "cust-longterm-arthur",
							PersonaName: "Arthur Pendelton",
							ActionName:  "arthur_generate_tax_form_1099b",
							Endpoint:    endpoint,
							Success:     false,
							StatusCode:  resp.StatusCode,
							Details:     fmt.Sprintf("Failed to generate Form 1099-B: %s", string(bytes)),
							Timestamp:   time.Now().UTC(),
						}, nil
					}

					// Audit list statements
					listEndpoint := "/api/v1/customer/statements?customer_id=018e0000-0000-7000-8000-000000000099"
					listResp, listBytes, listErr := client.DoCustomer(ctx, "GET", listEndpoint, nil)
					if listErr != nil {
						return nil, listErr
					}

					return &ActionResult{
						PersonaID:   "cust-longterm-arthur",
						PersonaName: "Arthur Pendelton",
						ActionName:  "arthur_generate_tax_form_1099b",
						Endpoint:    listEndpoint,
						Success:     listResp.StatusCode == http.StatusOK,
						StatusCode:  listResp.StatusCode,
						Details:     fmt.Sprintf("Form 1099-B compiled; statements audited: %s", string(listBytes)),
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
		ID:            "cust-active-chloe",
		Name:          "Chloe Spark",
		Type:          "customer",
		RoleTitle:     "Momentum Alpha Trader",
		Department:    "Retail Investor Community",
		Role:          "customer",
		Email:         "chloe@oci.local",
		ActivityLevel: 1.0,
		Actions: []Action{
			{
				Name:        "chloe_scan_activity_streams",
				Description: "Scans real-time camera streams for high-activity feline behavioral telemetry",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/streams"
					resp, bytes, err := client.DoCustomer(ctx, "GET", endpoint, nil)
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
					endpoint := "/api/v1/customer/trading/orders"
					payload := map[string]any{
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
						"symbol":       "NVDA",
						"side":         "buy",
						"quantity":     50.0,
						"price":        125.75,
					}
					resp, bytes, err := client.DoCustomer(ctx, "POST", endpoint, payload)
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
					endpoint := "/api/v1/customer/trading/orders"
					payload := map[string]any{
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
						"symbol":       "NVDA",
						"side":         "sell",
						"quantity":     50.0,
						"price":        131.25,
					}
					resp, bytes, err := client.DoCustomer(ctx, "POST", endpoint, payload)
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
					endpoint := "/api/v1/customer/trade-logs"
					resp, bytes, err := client.DoCustomer(ctx, "GET", endpoint, nil)
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
			{
				Name:        "chloe_query_live_market_quotes",
				Description: "Queries real-time market data quotes and prices across tracked symbols via Customer BFF",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/ticker"
					resp, bytes, err := client.DoCustomer(ctx, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "cust-active-chloe",
						PersonaName: "Chloe Spark",
						ActionName:  "chloe_query_live_market_quotes",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Queried real-time ticker quotes: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "chloe_margin_deposit_nsf_check",
				Description: "Attempts over-limit ACH capital transfer triggering bank NSF R01 pre-authorization rejection",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/deposits"
					payload := map[string]any{
						"customer_id":  "cust-active-chloe",
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000003",
						"amount_usd":   50000.00,
						"frequency":    "one_time",
						"bank_account": "Wells Fargo Checking (****9912)",
					}
					resp, bytes, err := client.DoCustomer(ctx, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					// $50,000 exceeds Chloe's $12,450.00 balance, so Bank API pre-authorization returns declined (R01 Insufficient Funds)
					isNSFRejection := resp.StatusCode == http.StatusBadRequest
					return &ActionResult{
						PersonaID:   "cust-active-chloe",
						PersonaName: "Chloe Spark",
						ActionName:  "chloe_margin_deposit_nsf_check",
						Endpoint:    endpoint,
						Success:     isNSFRejection,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Verified Bank API NSF R01 rejection as expected: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "chloe_close_account_and_liquidate",
				Description: "Initiates full portfolio liquidation selling all equity positions and disbursing cash via Plaid ACH",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/account/close"
					payload := map[string]any{
						"portfolio_id": "018f3a9a-2222-7000-8000-000000000003",
						"reason":       "Customer migrating portfolio to private feline alpha vehicle",
					}
					resp, bytes, err := client.DoCustomer(ctx, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					success := resp.StatusCode == http.StatusOK
					return &ActionResult{
						PersonaID:   "cust-active-chloe",
						PersonaName: "Chloe Spark",
						ActionName:  "chloe_close_account_and_liquidate",
						Endpoint:    endpoint,
						Success:     success,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Executed account closure & liquidation: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "chloe_request_gdpr_data_erasure",
				Description: "Executes GDPR Article 17 Right to be Forgotten data erasure and receives HMAC-SHA256 certificate",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/customer/privacy/gdpr-erasure"
					payload := map[string]any{
						"customer_id":  "018e0000-0000-7000-8000-000000000098",
						"confirmation": "DELETE",
						"reason":       "Customer privacy request under GDPR Article 17",
					}
					resp, bytes, err := client.DoCustomer(ctx, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					if resp.StatusCode != http.StatusOK {
						return &ActionResult{
							PersonaID:   "cust-active-chloe",
							PersonaName: "Chloe Spark",
							ActionName:  "chloe_request_gdpr_data_erasure",
							Endpoint:    endpoint,
							Success:     false,
							StatusCode:  resp.StatusCode,
							Details:     fmt.Sprintf("Failed to execute GDPR erasure: %s", string(bytes)),
							Timestamp:   time.Now().UTC(),
						}, nil
					}

					var res struct {
						Certificate struct {
							CertificateID    string `json:"certificate_id"`
							DigitalSignature string `json:"digital_signature"`
						} `json:"certificate"`
						CertificateID    string `json:"certificate_id"`
						DigitalSignature string `json:"digital_signature"`
					}
					_ = json.Unmarshal(bytes, &res)
					certID := res.Certificate.CertificateID
					if certID == "" {
						certID = res.CertificateID
					}
					sig := res.Certificate.DigitalSignature
					if sig == "" {
						sig = res.DigitalSignature
					}

					// Verify downloadable PDF certificate
					dlEndpoint := fmt.Sprintf("/api/v1/customer/privacy/gdpr-certificate/%s/download", certID)
					dlResp, dlBytes, dlErr := client.DoCustomer(ctx, "GET", dlEndpoint, nil)
					if dlErr != nil {
						return nil, dlErr
					}
					isPDF := dlResp.StatusCode == http.StatusOK && len(dlBytes) > 4 && string(dlBytes[:5]) == "%PDF-"

					return &ActionResult{
						PersonaID:   "cust-active-chloe",
						PersonaName: "Chloe Spark",
						ActionName:  "chloe_request_gdpr_data_erasure",
						Endpoint:    dlEndpoint,
						Success:     isPDF && sig != "",
						StatusCode:  dlResp.StatusCode,
						Details:     fmt.Sprintf("GDPR erasure executed, HMAC-SHA256 signature verified, PDF cert downloaded (%d bytes)", len(dlBytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 11. Rick Newhire - Junior Operations Associate
func (r *Registry) createRickPersona() *Persona {
	return &Persona{
		ID:            "emp-human-rick",
		Name:          "Rick Newhire",
		Type:          "human",
		RoleTitle:     "Junior Operations Associate",
		Department:    "Workforce Operations",
		Role:          "staff",
		Email:         "rick.newhire@oci.local",
		ActivityLevel: 1.0,
		Actions: []Action{
			{
				Name:        "rick_onboarding_checklist",
				Description: "Verifies workforce directory status during onboarding checklist execution",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/workforce/employees"
					resp, bytes, err := client.Do(ctx, true, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-rick",
						PersonaName: "Rick Newhire",
						ActionName:  "rick_onboarding_checklist",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Onboarding directory check completed: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "rick_read_nextcloud_docs",
				Description: "Reads onboarding and security policy documents shared in Nextcloud",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/nextcloud/api/v1/documents"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-rick",
						PersonaName: "Rick Newhire",
						ActionName:  "rick_read_nextcloud_docs",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Read shared Nextcloud documents: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "rick_send_nextcloud_chat_ack",
				Description: "Sends Nextcloud Chat response acknowledging receipt of laptop and onboarding documentation",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/nextcloud/api/v1/chat/messages"
					payload := map[string]any{
						"sender":    "emp-human-rick",
						"recipient": "emp-human-alice",
						"room":      "onboarding-general",
						"message":   "Thanks Alice! I set up my new MacBook Pro laptop asset and finished reading the Nextcloud handbook.",
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-rick",
						PersonaName: "Rick Newhire",
						ActionName:  "rick_send_nextcloud_chat_ack",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Sent Nextcloud chat ACK to HR: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "rick_check_onboarding_status",
				Description: "Queries employee onboarding status from workforce directory",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/workforce/employees?status=onboarding"
					resp, bytes, err := client.Do(ctx, true, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-rick",
						PersonaName: "Rick Newhire",
						ActionName:  "rick_check_onboarding_status",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Onboarding status queried. Payload: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "rick_trigger_self_onboarding_webhook",
				Description: "Simulates Frappe HR onboarding saga webhook for new hire lifecycle testing",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/webhooks/frappe-hr"
					payload := map[string]any{
						"event":         "employee_created",
						"employee_id":   "018f3a9a-e005-7000-8000-000000000005",
						"employee_type": "human",
						"first_name":    "Rick",
						"last_name":     "Newhire",
						"email":         "rick.newhire@oci.local",
						"role_title":    "Junior Operations Associate",
						"department":    "Workforce Operations",
						"status":        "onboarding",
					}
					resp, bytes, err := client.Do(ctx, true, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-rick",
						PersonaName: "Rick Newhire",
						ActionName:  "rick_trigger_self_onboarding_webhook",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Self onboarding saga webhook response: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 12. Elise Dev - Software Developer
func (r *Registry) createElisePersona() *Persona {
	return &Persona{
		ID:            "emp-human-elise",
		Name:          "Elise Dev",
		Type:          "human",
		RoleTitle:     "Software Developer",
		Department:    "Engineering & Platform Development",
		Role:          "engineering",
		Email:         "elise.dev@oci.local",
		ActivityLevel: 1.0,
		Actions: []Action{
			{
				Name:        "elise_claim_kanban_card",
				Description: "Claims and assigns engineering backlog card #1 on Forgejo kanban board to herself",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/repos/oci/monorepo/issues/1"
					payload := map[string]any{
						"assignee": "emp-human-elise",
						"state":    "open",
					}
					resp, bytes, err := client.DoForgejoAs(ctx, "emp-human-elise", "PATCH", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-elise",
						PersonaName: "Elise Dev",
						ActionName:  "elise_claim_kanban_card",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Claimed Kanban Card #1 in Forgejo: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "elise_create_feature_branch",
				Description: "Creates feature branch 'feature/42-gitops-promotion-pipeline' from main branch in Forgejo",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/repos/oci/monorepo/branches"
					payload := map[string]any{
						"new_branch_name": "feature/42-gitops-promotion-pipeline",
						"old_ref_name":    "main",
					}
					resp, bytes, err := client.DoForgejoAs(ctx, "emp-human-elise", "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					success := resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict
					return &ActionResult{
						PersonaID:   "emp-human-elise",
						PersonaName: "Elise Dev",
						ActionName:  "elise_create_feature_branch",
						Endpoint:    endpoint,
						Success:     success,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Created feature branch in Forgejo: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "elise_push_code_commit",
				Description: "Pushes git commit to feature branch with multi-stage Dockerfiles and promotion manifests",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/repos/oci/monorepo/contents/Dockerfile"
					payload := map[string]any{
						"branch":  "feature/42-gitops-promotion-pipeline",
						"content": "RlJPTSBnb2xhbmc6MS4yMi1hbHBpbmUgQVMgYnVpbGRlcg==",
						"message": "feat(gitops): multi-stage containerization & gated promotion pipeline (#42)",
					}
					resp, bytes, err := client.DoForgejoAs(ctx, "emp-human-elise", "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					success := resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusUnprocessableEntity
					return &ActionResult{
						PersonaID:   "emp-human-elise",
						PersonaName: "Elise Dev",
						ActionName:  "elise_push_code_commit",
						Endpoint:    endpoint,
						Success:     success,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Pushed commit to feature branch in Forgejo: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "elise_open_pull_request",
				Description: "Opens Pull Request in Forgejo linking Card #1 for automated CI validation and peer review",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/repos/oci/monorepo/pulls"
					payload := map[string]any{
						"title": "feat(gitops): multi-stage containerization & gated promotion pipeline",
						"head":  "feature/42-gitops-promotion-pipeline",
						"base":  "main",
						"body":  "Resolves #1. Multi-stage Dockerfiles, in-cluster Forgejo runner, and live staging gates.",
					}
					resp, bytes, err := client.DoForgejoAs(ctx, "emp-human-elise", "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					success := resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict
					return &ActionResult{
						PersonaID:   "emp-human-elise",
						PersonaName: "Elise Dev",
						ActionName:  "elise_open_pull_request",
						Endpoint:    endpoint,
						Success:     success,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Opened Pull Request in Forgejo: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "elise_write_nextcloud_doc",
				Description: "Creates Vue 3 & Employee BFF architecture specification document in Nextcloud",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/nextcloud/api/v1/documents"
					payload := map[string]any{
						"title":      "OCI Employee Portal Vue 3 & BFF Architecture Spec",
						"content":    "Detailed technical breakdown of Vue 3 SPA components, Valkey session caching, and double-submit CSRF headers.",
						"author":     "emp-human-elise",
						"share_with": []string{"emp-human-frank", "emp-human-rick"},
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-elise",
						PersonaName: "Elise Dev",
						ActionName:  "elise_write_nextcloud_doc",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Published engineering doc in Nextcloud: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "elise_send_nextcloud_chat",
				Description: "Sends Nextcloud Chat message with document link to Rick and the development team",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/nextcloud/api/v1/chat/messages"
					payload := map[string]any{
						"sender":    "emp-human-elise",
						"recipient": "emp-human-rick",
						"room":      "engineering-onboarding",
						"message":   "Hey Rick, welcome! I posted the Vue 3 & Employee BFF architecture doc in Nextcloud for your onboarding reading.",
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-elise",
						PersonaName: "Elise Dev",
						ActionName:  "elise_send_nextcloud_chat",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Sent engineering onboarding message via Nextcloud Chat: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "elise_monitor_it_tickets",
				Description: "Monitors IT helpdesk queue for developer tooling and CI/CD workflow issues",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/ops/tickets"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-elise",
						PersonaName: "Elise Dev",
						ActionName:  "elise_monitor_it_tickets",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Checked IT tickets queue for dev tooling requests: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "elise_create_code_review_ticket",
				Description: "Submits a Forgejo code review & CI/CD workflow ticket for platform builds",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/ops/tickets"
					payload := map[string]any{
						"forgejo_repo":    "oci/monorepo",
						"title":           "PR #42: Enhance Go mTLS cert auto-reloader & CI pipeline",
						"body":            "Automated build and test suite run successfully in Forgejo Actions runner.",
						"author_username": "emp-human-elise",
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-elise",
						PersonaName: "Elise Dev",
						ActionName:  "elise_create_code_review_ticket",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Code review ticket submitted: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "elise_query_developer_tickets",
				Description: "Audits active developer IT tickets and CI build issues",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/ops/tickets"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-elise",
						PersonaName: "Elise Dev",
						ActionName:  "elise_query_developer_tickets",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Audited developer tickets response size: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "elise_submit_habitat_helpdesk_ticket",
				Description: "Submits an urgent priority habitat IT helpdesk ticket via Employee BFF",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/api/v1/ops/tickets"
					payload := map[string]any{
						"title":           "Habitat Alpha Thermostat Calibration Drift",
						"body":            "Environmental monitoring sensor reports temperature delta exceeding +3.5C in Habitat Alpha.",
						"author_username": "emp-human-elise",
						"category":        "habitat",
						"severity":        "urgent",
						"is_urgent":       true,
						"forgejo_repo":    "oci-internal/helpdesk",
					}
					resp, bytes, err := client.Do(ctx, true, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-elise",
						PersonaName: "Elise Dev",
						ActionName:  "elise_submit_habitat_helpdesk_ticket",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Submitted urgent habitat helpdesk ticket: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

// 13. Marcus Lead - Principal Systems Architect & Senior Developer
func (r *Registry) createMarcusPersona() *Persona {
	return &Persona{
		ID:            "emp-human-marcus",
		Name:          "Marcus Lead",
		Type:          "human",
		RoleTitle:     "Principal Systems Architect & Senior Developer",
		Department:    "Engineering & Platform Development",
		Role:          "engineering",
		Email:         "marcus.lead@oci.local",
		ActivityLevel: 1.0,
		Actions: []Action{
			{
				Name:        "senior_dev_author_adr",
				Description: "Authors and publishes ADR 0003 for GitOps and containerization pipeline in Nextcloud",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/nextcloud/api/v1/documents"
					payload := map[string]any{
						"title":      "ADR 0003: Self-Hosted GitOps, Multi-Stage Packaging & Gated Promotion Pipeline",
						"content":    "Defines multi-stage container builds, local registry caching on localhost:32000, Forgejo Actions runner execution, and live integration testing gate before production promotion.",
						"author":     "emp-human-marcus",
						"share_with": []string{"emp-human-elise", "emp-human-frank"},
					}
					resp, bytes, err := client.Do(ctx, false, "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-marcus",
						PersonaName: "Marcus Lead",
						ActionName:  "senior_dev_author_adr",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Published ADR 0003 in Nextcloud: %s", string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "senior_dev_review_pull_request",
				Description: "Performs architectural peer review and approves Elise Dev's Pull Request in Forgejo",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					prNumber := 13
					listResp, listBytes, listErr := client.DoForgejo(ctx, "GET", "/api/v1/repos/oci/monorepo/pulls?state=all&limit=10", nil)
					if listErr == nil && listResp.StatusCode == http.StatusOK {
						var prs []struct {
							Number int    `json:"number"`
							State  string `json:"state"`
						}
						if err := json.Unmarshal(listBytes, &prs); err == nil && len(prs) > 0 {
							prNumber = prs[0].Number
							for _, p := range prs {
								if p.State == "open" {
									prNumber = p.Number
									break
								}
							}
						}
					}
					endpoint := fmt.Sprintf("/api/v1/repos/oci/monorepo/pulls/%d/reviews", prNumber)
					payload := map[string]any{
						"event": "APPROVED",
						"body":  "Architecture and security invariants verified. Distroless/alpine multi-stage builds and Forgejo Actions CI pass all tests. Approved for staging deployment.",
					}
					resp, bytes, err := client.DoForgejoAs(ctx, "emp-human-marcus", "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					success := resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusUnprocessableEntity
					return &ActionResult{
						PersonaID:   "emp-human-marcus",
						PersonaName: "Marcus Lead",
						ActionName:  "senior_dev_review_pull_request",
						Endpoint:    endpoint,
						Success:     success,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Submitted approved peer review on PR #%d in Forgejo: %s", prNumber, string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "senior_dev_merge_pull_request",
				Description: "Merges Pull Request in Forgejo, triggering automated CI build and container push to localhost:32000",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					prNumber := 13
					listResp, listBytes, listErr := client.DoForgejo(ctx, "GET", "/api/v1/repos/oci/monorepo/pulls?state=all&limit=10", nil)
					if listErr == nil && listResp.StatusCode == http.StatusOK {
						var prs []struct {
							Number int    `json:"number"`
							State  string `json:"state"`
						}
						if err := json.Unmarshal(listBytes, &prs); err == nil && len(prs) > 0 {
							prNumber = prs[0].Number
							for _, p := range prs {
								if p.State == "open" {
									prNumber = p.Number
									break
								}
							}
						}
					}
					endpoint := fmt.Sprintf("/api/v1/repos/oci/monorepo/pulls/%d/merge", prNumber)
					payload := map[string]any{
						"Do":                "merge",
						"MergeMessageField": fmt.Sprintf("Merge pull request #%d: feat(gitops): multi-stage containerization & gated promotion pipeline", prNumber),
					}
					resp, bytes, err := client.DoForgejoAs(ctx, "emp-human-marcus", "POST", endpoint, payload)
					if err != nil {
						return nil, err
					}
					success := resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusMethodNotAllowed
					return &ActionResult{
						PersonaID:   "emp-human-marcus",
						PersonaName: "Marcus Lead",
						ActionName:  "senior_dev_merge_pull_request",
						Endpoint:    endpoint,
						Success:     success,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Merged PR #%d in Forgejo: %s", prNumber, string(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
			{
				Name:        "senior_dev_audit_ci_pipeline",
				Description: "Audits active developer tickets and CI/CD workflow status across platform deployments",
				Execute: func(ctx context.Context, client *Client) (*ActionResult, error) {
					endpoint := "/ops/tickets"
					resp, bytes, err := client.Do(ctx, false, "GET", endpoint, nil)
					if err != nil {
						return nil, err
					}
					return &ActionResult{
						PersonaID:   "emp-human-marcus",
						PersonaName: "Marcus Lead",
						ActionName:  "senior_dev_audit_ci_pipeline",
						Endpoint:    endpoint,
						Success:     resp.StatusCode == http.StatusOK,
						StatusCode:  resp.StatusCode,
						Details:     fmt.Sprintf("Audited CI/CD tickets queue: %d bytes", len(bytes)),
						Timestamp:   time.Now().UTC(),
					}, nil
				},
			},
		},
	}
}

