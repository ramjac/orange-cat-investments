package customerbff

import (
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- verifying coturn HMAC-SHA1 TURN REST API credentials in test
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTURNCredentials_HMACSHA1(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	secret := "test-turn-shared-secret"

	username, credential, exp, err := GenerateTURNCredentials(secret, "cust-longterm-arthur", now, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(1700003600), exp.Unix())
	assert.Equal(t, "1700003600:cust-longterm-arthur", username)

	// Recompute expected HMAC-SHA1 base64 digest per coturn REST API spec.
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write([]byte(username))
	expectedCred := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	assert.Equal(t, expectedCred, credential)

	// Missing secret or empty customer ID must return an error.
	_, _, _, err = GenerateTURNCredentials("", "cust-longterm-arthur", now, time.Hour)
	assert.Error(t, err)
	_, _, _, err = GenerateTURNCredentials(secret, "   ", now, time.Hour)
	assert.Error(t, err)
}

func TestCustomerBFF_WebRTCICEServersEndpoint(t *testing.T) {
	bff, mux := setupTestBFF()
	secret := "ephemeral-coturn-secret-01"
	bff.SetTURNConfig("turn.oci.local:3478", secret)

	// 1. Unauthenticated request must return 401 Unauthorized.
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/webrtc/ice-servers", nil)
	unauthRec := httptest.NewRecorder()
	mux.ServeHTTP(unauthRec, unauthReq)
	assert.Equal(t, http.StatusUnauthorized, unauthRec.Code)

	// 2. Authenticated session request returns STUN + TURN URIs with 1h HMAC-SHA1 credential.
	before := time.Now().UTC().Unix()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/webrtc/ice-servers", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	after := time.Now().UTC().Unix()

	require.Equal(t, http.StatusOK, rec.Code)

	var payload ICEServersResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, 3600, payload.TTLSeconds)
	require.Len(t, payload.ICEServers, 2)
	assert.Contains(t, payload.ICEServers[0].URLs, "stun:turn.oci.local:3478")
	assert.Contains(t, payload.ICEServers[1].URLs, "turn:turn.oci.local:3478?transport=udp")
	assert.Contains(t, payload.ICEServers[1].URLs, "turn:turn.oci.local:3478?transport=tcp")

	parts := strings.Split(payload.ICEServers[1].Username, ":")
	require.Len(t, parts, 2)
	assert.Equal(t, "cust-longterm-arthur", parts[1])
	expUnix, err := strconv.ParseInt(parts[0], 10, 64)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, expUnix, before+3600)
	assert.LessOrEqual(t, expUnix, after+3600)

	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write([]byte(payload.ICEServers[1].Username))
	expectedCred := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	assert.Equal(t, expectedCred, payload.ICEServers[1].Credential)
}
