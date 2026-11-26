package simulation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveMailpitURL(t *testing.T) {
	assert.Equal(t, "http://dev.mail.oci.local", ResolveMailpitURL("", "https://dev.api.oci.local"))
	assert.Equal(t, "http://mail.oci.local", ResolveMailpitURL("", "https://api.oci.local"))
	assert.Equal(t, "http://custom:8025", ResolveMailpitURL("http://custom:8025/", "https://dev.api.oci.local"))
	assert.Equal(t, "", ResolveMailpitURL("", "http://localhost:8081"))
}

func TestClient_QueryMailpitMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/messages", r.URL.Path)
		assert.Equal(t, "to:chloe@oci.local", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"messages":[{"ID":"msg-1","Subject":"OCI Trade Confirmation: BUY NVDA","To":[{"Address":"chloe@oci.local"}]}]}`))
	}))
	defer srv.Close()

	c, err := NewClient(&Config{
		EmployeeBFFURL: "http://localhost:8081",
		CustomerBFFURL: "http://localhost:8083",
		MailpitURL:     srv.URL,
	})
	require.NoError(t, err)

	msgs, err := c.QueryMailpitMessages(context.Background(), "to:chloe@oci.local")
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "msg-1", msgs[0].ID)
	assert.Contains(t, msgs[0].Subject, "Trade Confirmation")
}
