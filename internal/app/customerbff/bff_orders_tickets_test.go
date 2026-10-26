package customerbff

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerBFF_DynamicCustomerRemediation(t *testing.T) {
	_, mux := setupTestBFF()
	csrfVal := "remediation-csrf-token"

	// 1. Dynamic Login with arbitrary new customer ID (not Chloe or Arthur)
	loginBody := bytes.NewBufferString(`{"customer_id":"cust-dynamo-999","name":"Diana Prince","email":"diana@oci.local"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/auth/login", loginBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var loginResp struct {
		Session *CustomerSession `json:"session"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	require.NotNil(t, loginResp.Session)
	assert.Equal(t, "cust-dynamo-999", loginResp.Session.CustomerID)
	assert.Equal(t, "Diana Prince", loginResp.Session.Name)

	sessToken := loginResp.Session.SessionID

	// 2. Dynamic portfolio fallback for Diana: Must NOT return Chloe or Arthur's holdings
	portReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio", nil)
	portReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	portRec := httptest.NewRecorder()
	mux.ServeHTTP(portRec, portReq)
	assert.Equal(t, http.StatusOK, portRec.Code)

	var portResp map[string]interface{}
	err = json.Unmarshal(portRec.Body.Bytes(), &portResp)
	require.NoError(t, err)
	assert.Equal(t, "cust-dynamo-999", portResp["customer_id"])
	assert.Equal(t, "Diana Prince", portResp["customer_name"])
	assert.Equal(t, "port-cust-dynamo-999", portResp["portfolio_id"])
	assert.Equal(t, 0.0, portResp["total_balance_usd"])

	// 3. API Keys: Diana starts with 0 keys (not Chloe Momentum Bot)
	keysReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/api-keys", nil)
	keysReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	keysRec := httptest.NewRecorder()
	mux.ServeHTTP(keysRec, keysReq)
	assert.Equal(t, http.StatusOK, keysRec.Code)

	var keys []map[string]interface{}
	err = json.Unmarshal(keysRec.Body.Bytes(), &keys)
	require.NoError(t, err)
	assert.Empty(t, keys, "new customer should start with empty API keys list, not hardcoded bot keys")

	// Create an API key for Diana
	createKeyBody := bytes.NewBufferString(`{"name":"Diana Alpha Bot","permissions":["execute_orders"]}`)
	createKeyReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/api-keys", createKeyBody)
	createKeyReq.Header.Set("Content-Type", "application/json")
	createKeyReq.Header.Set("X-CSRF-Token", csrfVal)
	createKeyReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	createKeyReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	createKeyRec := httptest.NewRecorder()
	mux.ServeHTTP(createKeyRec, createKeyReq)
	assert.Equal(t, http.StatusCreated, createKeyRec.Code)

	var createdKey map[string]interface{}
	err = json.Unmarshal(createKeyRec.Body.Bytes(), &createdKey)
	require.NoError(t, err)
	assert.Equal(t, "Diana Alpha Bot", createdKey["name"])
	assert.Equal(t, "cust-dynamo-999", createdKey["customer_id"])

	// Diana now has 1 key
	keysReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/customer/api-keys", nil)
	keysReq2.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	keysRec2 := httptest.NewRecorder()
	mux.ServeHTTP(keysRec2, keysReq2)
	assert.Equal(t, http.StatusOK, keysRec2.Code)

	var keys2 []map[string]interface{}
	err = json.Unmarshal(keysRec2.Body.Bytes(), &keys2)
	require.NoError(t, err)
	require.Len(t, keys2, 1)
	assert.Equal(t, "Diana Alpha Bot", keys2[0]["name"])

	// 4. Webhooks: Diana starts with 0 webhooks (not Arthur's or Chloe's webhook)
	whReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/webhooks", nil)
	whReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	whRec := httptest.NewRecorder()
	mux.ServeHTTP(whRec, whReq)
	assert.Equal(t, http.StatusOK, whRec.Code)

	var webhooks []map[string]interface{}
	err = json.Unmarshal(whRec.Body.Bytes(), &webhooks)
	require.NoError(t, err)
	assert.Empty(t, webhooks, "new customer should start with empty webhooks list")

	// Create a webhook for Diana
	createWhBody := bytes.NewBufferString(`{"target_url":"https://diana-bot.example.com/webhook","events":["events.trade.executed.v1"]}`)
	createWhReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/webhooks", createWhBody)
	createWhReq.Header.Set("Content-Type", "application/json")
	createWhReq.Header.Set("X-CSRF-Token", csrfVal)
	createWhReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	createWhReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	createWhRec := httptest.NewRecorder()
	mux.ServeHTTP(createWhRec, createWhReq)
	assert.Equal(t, http.StatusCreated, createWhRec.Code)

	// Diana now has 1 webhook
	whReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/customer/webhooks", nil)
	whReq2.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	whRec2 := httptest.NewRecorder()
	mux.ServeHTTP(whRec2, whReq2)
	assert.Equal(t, http.StatusOK, whRec2.Code)

	var webhooks2 []map[string]interface{}
	err = json.Unmarshal(whRec2.Body.Bytes(), &webhooks2)
	require.NoError(t, err)
	require.Len(t, webhooks2, 1)
	assert.Equal(t, "https://diana-bot.example.com/webhook", webhooks2[0]["target_url"])

	// 5. Order placement dynamically resolves Diana's portfolio
	orderBody := bytes.NewBufferString(`{"symbol":"AAPL","side":"buy","quantity":2}`)
	orderReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", orderBody)
	orderReq.Header.Set("Content-Type", "application/json")
	orderReq.Header.Set("X-CSRF-Token", csrfVal)
	orderReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	orderReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	orderRec := httptest.NewRecorder()
	mux.ServeHTTP(orderRec, orderReq)
	assert.Equal(t, http.StatusOK, orderRec.Code)

	var orderResp map[string]interface{}
	err = json.Unmarshal(orderRec.Body.Bytes(), &orderResp)
	require.NoError(t, err)
	assert.Equal(t, "port-cust-dynamo-999", orderResp["portfolio_id"])
}

func TestCustomerBFF_CustomerSupport_ListAndCreateTickets(t *testing.T) {
	_, mux := setupTestBFF()
	csrfVal := "support-csrf-token"

	// 1. Unauthenticated request to /api/v1/customer/support/tickets should be rejected
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/support/tickets", nil)
	unauthRec := httptest.NewRecorder()
	mux.ServeHTTP(unauthRec, unauthReq)
	assert.Equal(t, http.StatusUnauthorized, unauthRec.Code)

	// 2. Authenticated request for Arthur
	loginBody := bytes.NewBufferString(`{"customer_id":"cust-longterm-arthur","name":"Arthur Pendelton","email":"arthur@oci.local"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/auth/login", loginBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var loginResp struct {
		Session *CustomerSession `json:"session"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &loginResp))
	sessToken := loginResp.Session.SessionID

	// 3. Create a ticket without CSRF should fail
	createTicketBody := bytes.NewBufferString(`{"category":"Trading","title":"Execution Delay Question","body":"Why did my order take 30s to fill?"}`)
	badCSRFReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/support/tickets", createTicketBody)
	badCSRFReq.Header.Set("Content-Type", "application/json")
	badCSRFReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	badCSRFRec := httptest.NewRecorder()
	mux.ServeHTTP(badCSRFRec, badCSRFReq)
	assert.Equal(t, http.StatusForbidden, badCSRFRec.Code)

	// 4. Create a ticket with valid CSRF
	createTicketBody = bytes.NewBufferString(`{"category":"Trading","title":"Execution Delay Question","body":"Why did my order take 30s to fill?"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/support/tickets", createTicketBody)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-CSRF-Token", csrfVal)
	createReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	createReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)
	assert.Equal(t, http.StatusCreated, createRec.Code)

	var createdTicket map[string]interface{}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createdTicket))
	ticketID, ok := createdTicket["ticket_id"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, ticketID)
	assert.Equal(t, "Execution Delay Question", createdTicket["title"])
	assert.Equal(t, "cust-longterm-arthur", createdTicket["customer_id"])

	// 5. List tickets for Arthur
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/support/tickets", nil)
	listReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	assert.Equal(t, http.StatusOK, listRec.Code)

	var listResp struct {
		Items []map[string]interface{} `json:"items"`
		Total int                      `json:"total"`
	}
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listResp))
	assert.GreaterOrEqual(t, listResp.Total, 1)

	// 6. Get single ticket
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/support/tickets/"+ticketID, nil)
	getReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	assert.Equal(t, http.StatusOK, getRec.Code)

	// 7. Add customer comment with CSRF
	commentBody := bytes.NewBufferString(`{"comment":"Following up on this, any updates from the desk?"}`)
	commentReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/support/tickets/"+ticketID+"/comments", commentBody)
	commentReq.Header.Set("Content-Type", "application/json")
	commentReq.Header.Set("X-CSRF-Token", csrfVal)
	commentReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	commentReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	commentRec := httptest.NewRecorder()
	mux.ServeHTTP(commentRec, commentReq)
	assert.Equal(t, http.StatusCreated, commentRec.Code)

	// 8. List comments (sanitized)
	commentsReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/support/tickets/"+ticketID+"/comments", nil)
	commentsReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	commentsRec := httptest.NewRecorder()
	mux.ServeHTTP(commentsRec, commentsReq)
	assert.Equal(t, http.StatusOK, commentsRec.Code)

	var commentsResp struct {
		Items []map[string]interface{} `json:"items"`
		Total int                      `json:"total"`
	}
	require.NoError(t, json.Unmarshal(commentsRec.Body.Bytes(), &commentsResp))
	assert.GreaterOrEqual(t, commentsResp.Total, 1)
}

func TestCustomerBFF_CustomerSupport_CommentScrubbing(t *testing.T) {
	bff, mux := setupTestBFF()
	csrfVal := "scrub-csrf-token"

	// Login as Arthur
	loginBody := bytes.NewBufferString(`{"customer_id":"cust-longterm-arthur","name":"Arthur Pendelton","email":"arthur@oci.local"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/auth/login", loginBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var loginResp struct {
		Session *CustomerSession `json:"session"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &loginResp))
	sessToken := loginResp.Session.SessionID

	// Create customer ticket
	createTicketBody := bytes.NewBufferString(`{"category":"UI Bug","title":"Chart zoom bug","body":"Zooming in resets axis"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/support/tickets", createTicketBody)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-CSRF-Token", csrfVal)
	createReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	createReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)
	assert.Equal(t, http.StatusCreated, createRec.Code)

	var createdTicket map[string]interface{}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createdTicket))
	ticketID := createdTicket["ticket_id"].(string)

	// Add internal staff comment directly via bllClient
	_, err := bff.BLLClient().AddCustomerTicketComment(context.Background(), &customerbllv1.AddCustomerTicketCommentRequest{
		CustomerID:     "cust-longterm-arthur",
		TicketID:       ticketID,
		AuthorUsername: "frank_infra",
		Comment:        "INTERNAL ONLY: Database lock contention occurred on server node 3.",
	})
	require.NoError(t, err)

	// Add public support response via bllClient
	_, err = bff.BLLClient().AddCustomerTicketComment(context.Background(), &customerbllv1.AddCustomerTicketCommentRequest{
		CustomerID:     "cust-longterm-arthur",
		TicketID:       ticketID,
		AuthorUsername: "support_agent",
		Comment:        "Thank you for reporting. Our engineering team is reviewing this.",
	})
	require.NoError(t, err)

	// Customer fetches comments via Customer BFF
	commentsReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/support/tickets/"+ticketID+"/comments", nil)
	commentsReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	commentsRec := httptest.NewRecorder()
	mux.ServeHTTP(commentsRec, commentsReq)
	assert.Equal(t, http.StatusOK, commentsRec.Code)

	var commentsResp struct {
		Items []map[string]interface{} `json:"items"`
		Total int                      `json:"total"`
	}
	require.NoError(t, json.Unmarshal(commentsRec.Body.Bytes(), &commentsResp))
	assert.Equal(t, 1, commentsResp.Total)
	assert.Equal(t, "support_agent", commentsResp.Items[0]["author_username"])
	assert.Equal(t, "Thank you for reporting. Our engineering team is reviewing this.", commentsResp.Items[0]["comment"])

	// Assert the internal note was completely scrubbed!
	for _, c := range commentsResp.Items {
		assert.False(t, c["is_internal"].(bool))
		assert.NotContains(t, c["comment"].(string), "INTERNAL ONLY")
	}
}
