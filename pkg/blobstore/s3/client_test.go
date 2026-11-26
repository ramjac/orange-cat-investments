package s3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type storedObject struct {
	contentType string
	data        []byte
	sha256Meta  string
}

type mockS3Server struct {
	mu      sync.RWMutex
	buckets map[string]bool
	objects map[string]storedObject
}

func newMockS3Server() (*mockS3Server, *httptest.Server) {
	m := &mockS3Server{
		buckets: make(map[string]bool),
		objects: make(map[string]storedObject),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trimmed := strings.TrimPrefix(r.URL.Path, "/")
		parts := strings.SplitN(trimmed, "/", 2)
		bucket := parts[0]
		key := ""
		if len(parts) == 2 {
			key = parts[1]
		}

		// Bucket-level operations
		if key == "" {
			m.mu.Lock()
			defer m.mu.Unlock()
			switch r.Method {
			case http.MethodHead:
				if m.buckets[bucket] {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			case http.MethodPut:
				m.buckets[bucket] = true
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}

		// Object-level operations
		objKey := bucket + "/" + key
		switch r.Method {
		case http.MethodPut:
			if !strings.Contains(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256") || r.Header.Get("x-amz-date") == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			sum := sha256.Sum256(body)
			sumHex := hex.EncodeToString(sum[:])
			if sumHex != r.Header.Get("x-amz-content-sha256") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			m.mu.Lock()
			m.buckets[bucket] = true
			m.objects[objKey] = storedObject{
				contentType: r.Header.Get("Content-Type"),
				data:        body,
				sha256Meta:  r.Header.Get("x-amz-meta-sha256"),
			}
			m.mu.Unlock()
			w.Header().Set("ETag", `"`+sumHex[:16]+`"`)
			w.WriteHeader(http.StatusOK)

		case http.MethodGet:
			// Verify either Authorization header OR pre-signed SigV4 query parameters
			if r.Header.Get("Authorization") == "" {
				q := r.URL.Query()
				if q.Get("X-Amz-Algorithm") != "AWS4-HMAC-SHA256" || q.Get("X-Amz-Signature") == "" || q.Get("X-Amz-Expires") == "" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
			m.mu.RLock()
			obj, ok := m.objects[objKey]
			m.mu.RUnlock()
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if obj.contentType != "" {
				w.Header().Set("Content-Type", obj.contentType)
			}
			if obj.sha256Meta != "" {
				w.Header().Set("x-amz-meta-sha256", obj.sha256Meta)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(obj.data)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	return m, srv
}

func TestClient_BucketsPutGetAndPresign(t *testing.T) {
	_, srv := newMockS3Server()
	defer srv.Close()

	fixedTime := time.Date(2026, 11, 25, 15, 0, 0, 0, time.UTC)
	blobStore, err := NewClient(Config{
		Endpoint:        srv.URL,
		Region:          "garage",
		AccessKeyID:     "GKtestaccesskey001",
		SecretAccessKey: "testsecretkey001",
		Now:             func() time.Time { return fixedTime },
	})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, blobStore.EnsureBuckets(ctx, "oci-statements", "oci-firmware", "oci-backups"))
	// Second EnsureBuckets call is idempotent
	require.NoError(t, blobStore.EnsureBuckets(ctx, "oci-statements", "oci-firmware", "oci-backups"))

	pdfPayload := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF\n")
	meta, err := blobStore.PutBlob(ctx, "oci-statements", "statements/cust-1/stmt-1.pdf", "application/pdf", pdfPayload)
	require.NoError(t, err)
	require.NotNil(t, meta)

	expectedSum := sha256.Sum256(pdfPayload)
	assert.Equal(t, hex.EncodeToString(expectedSum[:]), meta.SHA256Hex)
	assert.Equal(t, int64(len(pdfPayload)), meta.SizeBytes)
	assert.Equal(t, "application/pdf", meta.ContentType)

	fetched, err := blobStore.GetBlob(ctx, "oci-statements", "statements/cust-1/stmt-1.pdf")
	require.NoError(t, err)
	assert.Equal(t, pdfPayload, fetched)

	presignedURL, err := blobStore.PresignGetURL(ctx, "oci-statements", "statements/cust-1/stmt-1.pdf", 10*time.Minute)
	require.NoError(t, err)

	parsed, err := url.Parse(presignedURL)
	require.NoError(t, err)
	assert.Equal(t, "/oci-statements/statements/cust-1/stmt-1.pdf", parsed.Path)
	assert.Equal(t, "AWS4-HMAC-SHA256", parsed.Query().Get("X-Amz-Algorithm"))
	assert.Equal(t, "20261125T150000Z", parsed.Query().Get("X-Amz-Date"))
	assert.Equal(t, "600", parsed.Query().Get("X-Amz-Expires"))
	assert.Equal(t, "host", parsed.Query().Get("X-Amz-SignedHeaders"))
	assert.Equal(t, "GKtestaccesskey001/20261125/garage/s3/aws4_request", parsed.Query().Get("X-Amz-Credential"))
	assert.Len(t, parsed.Query().Get("X-Amz-Signature"), 64)

	// Fetch directly via pre-signed URL without Authorization header
	httpResp, err := http.Get(presignedURL) // #nosec G107 -- test httptest URL
	require.NoError(t, err)
	defer httpResp.Body.Close()
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	dlBytes, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err)
	assert.Equal(t, pdfPayload, dlBytes)
}

func TestClient_SHA256IntegrityMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-amz-meta-sha256", strings.Repeat("0", 64))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-1.4 corrupted"))
	}))
	defer srv.Close()

	blobStore, err := NewClient(Config{
		Endpoint:        srv.URL,
		AccessKeyID:     "GK1",
		SecretAccessKey: "SEC1",
	})
	require.NoError(t, err)

	_, err = blobStore.GetBlob(context.Background(), "oci-statements", "statements/bad.pdf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SHA-256 mismatch")
}

func TestClient_PublicEndpointPresign(t *testing.T) {
	blobStore, err := NewClient(Config{
		Endpoint:        "http://garage-service.oci-dev.svc.cluster.local:3900",
		PublicEndpoint:  "http://dev.s3.oci.local",
		AccessKeyID:     "GKdev",
		SecretAccessKey: "SECdev",
	})
	require.NoError(t, err)

	u, err := blobStore.PresignGetURL(context.Background(), "oci-firmware", "firmware/edge_camera/2.3.0/fw.bin", 5*time.Minute)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(u, "http://dev.s3.oci.local/oci-firmware/firmware/edge_camera/2.3.0/fw.bin?"))
}
