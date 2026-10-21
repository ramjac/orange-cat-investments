package simulation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllDefaultPersonasRegistrationAndStructure(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)

	personas := r.List()
	assert.Len(t, personas, 13)

	expectedPersonas := map[string]int{
		"emp-feline-garfield":  4,
		"emp-feline-barneby":   3,
		"emp-human-alice":      9,
		"emp-human-bob":        6,
		"emp-human-carol":      2,
		"emp-human-david":      3,
		"emp-human-elena":      4,
		"emp-human-frank":      4,
		"cust-longterm-arthur": 11,
		"cust-active-chloe":    8,
		"emp-human-rick":       5,
		"emp-human-elise":      10,
		"emp-human-marcus":     4,
	}

	for id, expectedActionCount := range expectedPersonas {
		p, ok := r.Get(id)
		require.True(t, ok, "persona %s should exist", id)
		assert.Equal(t, id, p.ID)
		assert.NotEmpty(t, p.Name)
		assert.NotEmpty(t, p.Type)
		assert.NotEmpty(t, p.RoleTitle)
		assert.NotEmpty(t, p.Department)
		assert.Len(t, p.Actions, expectedActionCount, "persona %s action count mismatch", id)

		for _, act := range p.Actions {
			assert.NotEmpty(t, act.Name)
			assert.NotEmpty(t, act.Description)
			assert.NotNil(t, act.Execute)
		}
	}
}

func TestRandomAction(t *testing.T) {
	// 1. Nil persona
	actNil := RandomAction(nil)
	assert.Empty(t, actNil.Name)

	// 2. Persona with empty actions
	pEmpty := &Persona{
		ID:            "emp-test-empty",
		ActivityLevel: 1.0,
		Actions:       []Action{},
	}
	actEmpty := RandomAction(pEmpty)
	assert.Empty(t, actEmpty.Name)

	// 3. ActivityLevel = 0.0
	pZero := &Persona{
		ID:            "emp-test-zero",
		ActivityLevel: 0.0,
		Actions: []Action{
			{Name: "action_1", Description: "Desc 1"},
			{Name: "action_2", Description: "Desc 2"},
		},
	}
	for i := 0; i < 50; i++ {
		act := RandomAction(pZero)
		assert.Empty(t, act.Name, "expected empty action when ActivityLevel is 0.0")
	}

	// 4. ActivityLevel = 1.0
	pActive := &Persona{
		ID:            "emp-test-active",
		ActivityLevel: 1.0,
		Actions: []Action{
			{Name: "action_1", Description: "Desc 1"},
			{Name: "action_2", Description: "Desc 2"},
		},
	}
	seenActions := make(map[string]bool)
	for i := 0; i < 100; i++ {
		act := RandomAction(pActive)
		assert.NotEmpty(t, act.Name)
		seenActions[act.Name] = true
	}
	assert.True(t, seenActions["action_1"])
	assert.True(t, seenActions["action_2"])

	// 5. ActivityLevel = 0.5 (probabilistic check)
	pHalf := &Persona{
		ID:            "emp-test-half",
		ActivityLevel: 0.5,
		Actions: []Action{
			{Name: "action_1", Description: "Desc 1"},
		},
	}
	sawActive := false
	sawEmpty := false
	for i := 0; i < 200; i++ {
		act := RandomAction(pHalf)
		if act.Name == "action_1" {
			sawActive = true
		} else if act.Name == "" {
			sawEmpty = true
		}
		if sawActive && sawEmpty {
			break
		}
	}
	assert.True(t, sawActive, "expected to see active action selected when r < ActivityLevel")
	assert.True(t, sawEmpty, "expected to see empty action when r >= ActivityLevel")
}

func TestGarfieldSleepInHallwayAction_Branches(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	r := NewRegistry()
	garfield, ok := r.Get("emp-feline-garfield")
	require.True(t, ok)

	var sleepAction *Action
	for _, act := range garfield.Actions {
		if act.Name == "garfield_sleep_in_hallway" {
			sleepAction = &act
			break
		}
	}
	require.NotNil(t, sleepAction)

	sawUntripped := false
	sawTripped := false

	ctx := context.Background()
	for i := 0; i < 100; i++ {
		res, err := sleepAction.Execute(ctx, client)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Success)

		if res.Details == "Garfield slept peacefully in the middle of the hallway. Nobody tripped." {
			sawUntripped = true
		} else {
			sawTripped = true
			assert.Contains(t, res.Details, "TRIPPED!")
		}

		if sawUntripped && sawTripped {
			break
		}
	}

	assert.True(t, sawUntripped, "expected to observe untripped Garfield sleep outcome")
	assert.True(t, sawTripped, "expected to observe tripped Garfield sleep outcome")
}

func TestGarfieldSleepInHallwayAction_Errors(t *testing.T) {
	r := NewRegistry()
	garfield, _ := r.Get("emp-feline-garfield")
	var sleepAction Action
	for _, a := range garfield.Actions {
		if a.Name == "garfield_sleep_in_hallway" {
			sleepAction = a
		}
	}

	// 1. Initial GET network error
	closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedServer.Close()
	c1 := &Client{serverURL: closedServer.URL, httpClient: closedServer.Client()}

	_, err := sleepAction.Execute(context.Background(), c1)
	assert.Error(t, err)

	// 2. Error when vet medical hold call fails
	// Server responds OK to initial GET /facilities/assets, but closes on /medical-hold
	vetFailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/facilities/assets" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		// Force connection drop on vet hold to trigger err != nil
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	defer vetFailServer.Close()

	c2 := &Client{serverURL: vetFailServer.URL, bffURL: vetFailServer.URL, httpClient: vetFailServer.Client()}
	for i := 0; i < 50; i++ {
		res, err := sleepAction.Execute(context.Background(), c2)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to toggle vet medical hold")
			break
		}
		_ = res
	}

	// 3. Error when workplace injury ticket POST fails
	injuryFailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/facilities/assets" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		if r.URL.Path == "/api/v1/workforce/care-schedules/emp-feline-garfield/medical-hold" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		if r.URL.Path == "/ops/tickets" {
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
			return
		}
	}))
	defer injuryFailServer.Close()

	c3 := &Client{serverURL: injuryFailServer.URL, bffURL: injuryFailServer.URL, httpClient: injuryFailServer.Client()}
	for i := 0; i < 50; i++ {
		res, err := sleepAction.Execute(context.Background(), c3)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to create workplace injury ticket")
			break
		}
		_ = res
	}
}

func TestAliceApproveTimesheetAction_Paths(t *testing.T) {
	r := NewRegistry()
	alice, _ := r.Get("emp-human-alice")
	var approveAction Action
	for _, act := range alice.Actions {
		if act.Name == "alice_approve_timesheet" {
			approveAction = act
			break
		}
	}

	// 1. Path A: Fallback creation when no submitted timesheets exist
	client1, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client1.Close()

	res1, err := approveAction.Execute(context.Background(), client1)
	require.NoError(t, err)
	assert.True(t, res1.Success)

	// 2. Path B: When submitted timesheet list already contains an item
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"items":[{"timesheet_id":"ts-existing-101","employee_id":"emp-bob"}]}`))
			return
		}
		if r.Method == "PUT" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"timesheet_id":"ts-existing-101","status":"approved"}`))
			return
		}
	}))
	defer srv.Close()

	client2 := &Client{bffURL: srv.URL, httpClient: srv.Client()}
	res2, err := approveAction.Execute(context.Background(), client2)
	require.NoError(t, err)
	assert.True(t, res2.Success)
	assert.Contains(t, res2.Details, "Approved workforce timesheet ts-existing-101")
}

func TestAliceApproveTimesheetAction_Errors(t *testing.T) {
	r := NewRegistry()
	alice, _ := r.Get("emp-human-alice")
	var approveAction Action
	for _, act := range alice.Actions {
		if act.Name == "alice_approve_timesheet" {
			approveAction = act
			break
		}
	}

	// 1. GET list network error
	closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedServer.Close()
	c1 := &Client{bffURL: closedServer.URL, httpClient: closedServer.Client()}

	_, err := approveAction.Execute(context.Background(), c1)
	assert.Error(t, err)

	// 2. POST create fallback network error when list is empty
	failPostServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"items":[]}`))
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer failPostServer.Close()

	c2 := &Client{bffURL: failPostServer.URL, httpClient: failPostServer.Client()}
	_, err = approveAction.Execute(context.Background(), c2)
	assert.Error(t, err)

	// 3. Empty timesheet ID returned
	emptyIDServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"items":[]}`))
			return
		}
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}
	}))
	defer emptyIDServer.Close()

	c3 := &Client{bffURL: emptyIDServer.URL, httpClient: emptyIDServer.Client()}
	_, err = approveAction.Execute(context.Background(), c3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pending timesheet found")

	// 4. PUT approve network error
	failPutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"items":[{"timesheet_id":"ts-123"}]}`))
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer failPutServer.Close()

	c4 := &Client{bffURL: failPutServer.URL, httpClient: failPutServer.Client()}
	_, err = approveAction.Execute(context.Background(), c4)
	assert.Error(t, err)
}

func TestAliceGenerateExecutiveReportAction_Paths(t *testing.T) {
	// Success path tested via mock client
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	r := NewRegistry()
	alice, _ := r.Get("emp-human-alice")
	var execReportAction Action
	for _, act := range alice.Actions {
		if act.Name == "alice_generate_executive_report" {
			execReportAction = act
			break
		}
	}

	res, err := execReportAction.Execute(context.Background(), client)
	require.NoError(t, err)
	assert.True(t, res.Success)

	// Failure status code path on report generation POST
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "generation error", http.StatusBadRequest)
	})
	server1 := httptest.NewServer(handler1)
	defer server1.Close()

	c1 := &Client{bffURL: server1.URL, httpClient: server1.Client()}
	res1, err := execReportAction.Execute(context.Background(), c1)
	require.NoError(t, err)
	assert.False(t, res1.Success)
	assert.Contains(t, res1.Details, "Failed to generate executive report")

	// History list GET network error
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer server2.Close()

	c2 := &Client{bffURL: server2.URL, httpClient: server2.Client()}
	_, err = execReportAction.Execute(context.Background(), c2)
	assert.Error(t, err)
}

func TestBarnebySlapWaterGlassAction_MultiStepErrors(t *testing.T) {
	r := NewRegistry()
	barneby, _ := r.Get("emp-feline-barneby")
	var slapAction Action
	for _, act := range barneby.Actions {
		if act.Name == "barneby_slap_water_glass" {
			slapAction = act
			break
		}
	}

	closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedServer.Close()

	// 1. Error on maintenance ticket creation
	c1 := &Client{serverURL: closedServer.URL, httpClient: closedServer.Client()}
	_, err := slapAction.Execute(context.Background(), c1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create maintenance ticket")

	// 2. Error on Pebble watch alerts query
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ops/tickets" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ticket_id":"t-100"}`))
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer s2.Close()
	c2 := &Client{serverURL: s2.URL, httpClient: s2.Client()}

	_, err = slapAction.Execute(context.Background(), c2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query Pebble watch alerts")

	// 3. Error on Pebble watch ACK
	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ops/tickets" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ticket_id":"t-100"}`))
			return
		}
		if r.URL.Path == "/ops/pebble/alerts" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer s3.Close()
	c3 := &Client{serverURL: s3.URL, httpClient: s3.Client()}

	_, err = slapAction.Execute(context.Background(), c3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to acknowledge Pebble watch alert")

	// 4. Error on maintenance log sync
	s4 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ops/tickets" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ticket_id":"t-100"}`))
			return
		}
		if r.URL.Path == "/ops/pebble/alerts" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		if r.URL.Path == "/ops/pebble/ack" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer s4.Close()
	c4 := &Client{serverURL: s4.URL, httpClient: s4.Client()}

	_, err = slapAction.Execute(context.Background(), c4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync mobile app maintenance log")
}

func TestAliceAndFrankPublishDocActions_Errors(t *testing.T) {
	r := NewRegistry()
	alice, _ := r.Get("emp-human-alice")
	var publishDocAction Action
	for _, act := range alice.Actions {
		if act.Name == "alice_publish_onboarding_doc" {
			publishDocAction = act
			break
		}
	}

	closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedServer.Close()

	// Document creation error
	c1 := &Client{serverURL: closedServer.URL, httpClient: closedServer.Client()}
	_, err := publishDocAction.Execute(context.Background(), c1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create Nextcloud document")

	// Chat message error
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/nextcloud/api/v1/documents" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer s2.Close()
	c2 := &Client{serverURL: s2.URL, httpClient: s2.Client()}

	_, err = publishDocAction.Execute(context.Background(), c2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send Nextcloud chat message")

	// Test Frank security doc errors
	frank, _ := r.Get("emp-human-frank")
	var frankSecurityDocAction Action
	for _, act := range frank.Actions {
		if act.Name == "frank_publish_security_doc" {
			frankSecurityDocAction = act
			break
		}
	}

	_, err = frankSecurityDocAction.Execute(context.Background(), c1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write security doc in Nextcloud")

	_, err = frankSecurityDocAction.Execute(context.Background(), c2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send Nextcloud chat message")
}

func TestArthurCustomerActions_PathsAndErrors(t *testing.T) {
	r := NewRegistry()
	arthur, _ := r.Get("cust-longterm-arthur")

	var supportTicketAction, monthlyStmtAction Action
	for _, act := range arthur.Actions {
		if act.Name == "arthur_submit_customer_support_ticket" {
			supportTicketAction = act
		}
		if act.Name == "arthur_generate_monthly_investment_statement" {
			monthlyStmtAction = act
		}
	}

	// 1. Support ticket failure status path
	h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	s1 := httptest.NewServer(h1)
	defer s1.Close()
	c1 := &Client{custBFFURL: s1.URL, httpClient: s1.Client()}

	res1, err := supportTicketAction.Execute(context.Background(), c1)
	require.NoError(t, err)
	assert.False(t, res1.Success)
	assert.Contains(t, res1.Details, "Failed to submit support ticket")

	// 2. Support ticket comments fetch error
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"ticket_id":"t-55"}`))
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer s2.Close()
	c2 := &Client{custBFFURL: s2.URL, httpClient: s2.Client()}

	_, err = supportTicketAction.Execute(context.Background(), c2)
	assert.Error(t, err)

	// 3. Monthly statement failure status path
	res2, err := monthlyStmtAction.Execute(context.Background(), c1)
	require.NoError(t, err)
	assert.False(t, res2.Success)
	assert.Contains(t, res2.Details, "Failed to generate monthly statement")

	// 4. Monthly statement download error
	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"statement_id":"s-10"}`))
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer s3.Close()
	c3 := &Client{custBFFURL: s3.URL, httpClient: s3.Client()}

	_, err = monthlyStmtAction.Execute(context.Background(), c3)
	assert.Error(t, err)
}

func TestArthurTaxForm1099B_Errors(t *testing.T) {
	r := NewRegistry()
	arthur, _ := r.Get("cust-longterm-arthur")

	var taxFormAction Action
	for _, act := range arthur.Actions {
		if act.Name == "arthur_generate_tax_form_1099b" {
			taxFormAction = act
			break
		}
	}

	// 1. Failure on POST generate tax form
	h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "failed tax form", http.StatusBadRequest)
	})
	s1 := httptest.NewServer(h1)
	defer s1.Close()
	c1 := &Client{custBFFURL: s1.URL, httpClient: s1.Client()}

	res1, err := taxFormAction.Execute(context.Background(), c1)
	require.NoError(t, err)
	assert.False(t, res1.Success)
	assert.Contains(t, res1.Details, "Failed to generate Form 1099-B")

	// 2. Error on GET list statements
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"statement_id":"tax-1"}`))
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer s2.Close()
	c2 := &Client{custBFFURL: s2.URL, httpClient: s2.Client()}

	_, err = taxFormAction.Execute(context.Background(), c2)
	assert.Error(t, err)
}

func TestChloeGDPRDataErasure_PathsAndErrors(t *testing.T) {
	r := NewRegistry()
	chloe, _ := r.Get("cust-active-chloe")

	var gdprAction Action
	for _, act := range chloe.Actions {
		if act.Name == "chloe_request_gdpr_data_erasure" {
			gdprAction = act
			break
		}
	}

	// 1. Failure status code on erasure endpoint
	h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gdpr forbidden", http.StatusForbidden)
	})
	s1 := httptest.NewServer(h1)
	defer s1.Close()
	c1 := &Client{custBFFURL: s1.URL, httpClient: s1.Client()}

	res1, err := gdprAction.Execute(context.Background(), c1)
	require.NoError(t, err)
	assert.False(t, res1.Success)
	assert.Contains(t, res1.Details, "Failed to execute GDPR erasure")

	// 2. Download error on certificate download
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"certificate_id":"cert-99","digital_signature":"sig-xyz"}`))
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer s2.Close()
	c2 := &Client{custBFFURL: s2.URL, httpClient: s2.Client()}

	_, err = gdprAction.Execute(context.Background(), c2)
	assert.Error(t, err)
}

func TestAllPersonaActions_TransportErrorHandling(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	s.Close()

	client := &Client{
		serverURL:  s.URL,
		bffURL:     s.URL,
		custBFFURL: s.URL,
		brokerURL:  s.URL,
		bankURL:    s.URL,
		forgejoURL: s.URL,
		httpClient: s.Client(),
	}

	r := NewRegistry()
	for _, persona := range r.List() {
		for _, act := range persona.Actions {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			res, err := act.Execute(ctx, client)
			cancel()
			if err == nil {
				assert.False(t, res.Success, "action %s should fail when network errors occur", act.Name)
			}
		}
	}
}
