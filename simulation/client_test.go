package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type samplePayload struct {
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Author    string   `json:"author"`
	ShareWith []string `json:"share_with"`
}

func BenchmarkRequestBodyCreation(b *testing.B) {
	smallPayload := samplePayload{
		Title:     "Zero-Trust Microservices & PKI Security Standard",
		Content:   "This is a detailed document outlining security specifications for microservices.",
		Author:    "emp-human-frank",
		ShareWith: []string{"emp-human-alice", "emp-human-rick", "emp-human-bob"},
	}

	largeContent := make([]byte, 10000)
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	largePayload := samplePayload{
		Title:     "Large System Architecture Document",
		Content:   string(largeContent),
		Author:    "emp-human-frank",
		ShareWith: []string{"emp-human-alice", "emp-human-rick", "emp-human-bob"},
	}

	b.Run("SmallPayload_JsonMarshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data, err := json.Marshal(smallPayload)
			if err != nil {
				b.Fatal(err)
			}
			_ = bytes.NewReader(data)
		}
	})

	b.Run("SmallPayload_BytesBufferEncoder", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(smallPayload); err != nil {
				b.Fatal(err)
			}
			_ = &buf
		}
	})

	b.Run("LargePayload_JsonMarshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data, err := json.Marshal(largePayload)
			if err != nil {
				b.Fatal(err)
			}
			_ = bytes.NewReader(data)
		}
	})

	b.Run("LargePayload_BytesBufferEncoder", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(largePayload); err != nil {
				b.Fatal(err)
			}
			_ = &buf
		}
	})
}

func TestClientDoWithBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "emp-human-frank")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	c := &Client{
		serverURL:  server.URL,
		httpClient: server.Client(),
	}

	payload := samplePayload{
		Title:     "Zero-Trust Microservices & PKI Security Standard",
		Content:   "This is a detailed document outlining security specifications for microservices.",
		Author:    "emp-human-frank",
		ShareWith: []string{"emp-human-alice", "emp-human-rick", "emp-human-bob"},
	}

	resp, respBytes, err := c.Do(context.Background(), false, "POST", "/test", payload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"status":"ok"}`, string(respBytes))
}
