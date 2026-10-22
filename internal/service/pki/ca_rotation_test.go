package pki

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCARotationManager(t *testing.T) {
	ctx := context.Background()

	t.Run("Default Config Initialization with Real Cryptography", func(t *testing.T) {
		manager := NewCARotationManager(CARotationConfig{})
		assert.NotNil(t, manager)

		status, err := manager.GetCAStatus(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "oci-intermediate-ca", status.CAName)
		assert.NotEmpty(t, status.SerialNumber)
		assert.True(t, status.NeedsRotation)
	})

	t.Run("CheckAndRotateCA When Rotation Needed", func(t *testing.T) {
		manager := NewCARotationManager(CARotationConfig{
			RenewalThresholdDays: 30,
		})

		initialStatus, _ := manager.GetCAStatus(ctx)
		initialSerial := initialStatus.SerialNumber

		res, err := manager.CheckAndRotateCA(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "CA_ROTATED_SUCCESSFULLY", res.Status)
		assert.Equal(t, "oci-intermediate-ca", res.IntermediateCAName)
		assert.NotEqual(t, initialSerial, res.NewSerialNumber)
		assert.Equal(t, 12, res.ReissuedCertificatesCount)

		// Check status after rotation: should be renewed for 1 year and not need rotation
		updatedStatus, errStatus := manager.GetCAStatus(ctx)
		assert.NoError(t, errStatus)
		assert.False(t, updatedStatus.NeedsRotation)
		assert.Equal(t, res.NewSerialNumber, updatedStatus.SerialNumber)
	})

	t.Run("CheckAndRotateCA When No Rotation Needed", func(t *testing.T) {
		manager := NewCARotationManager(CARotationConfig{
			RenewalThresholdDays: 30,
		})

		// Perform initial rotation to make CA fresh
		_, _ = manager.CheckAndRotateCA(ctx)

		// Second check should indicate no rotation needed
		res, err := manager.CheckAndRotateCA(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "NO_ROTATION_NEEDED", res.Status)
	})

	t.Run("Inspect Real Disk CA Certificate", func(t *testing.T) {
		rootCAPath := "../../../k8s/ha/lan/tls/oci-lan-root-ca.crt"
		if _, err := os.Stat(rootCAPath); os.IsNotExist(err) {
			t.Skip("local LAN root CA file not found, skipping disk inspection test")
		}

		manager := NewCARotationManager(CARotationConfig{
			CACertFile:           rootCAPath,
			RenewalThresholdDays: 30,
		})

		status, err := manager.GetCAStatus(ctx)
		require.NoError(t, err)
		assert.Contains(t, status.CAName, "Orange Cat Investments")
		assert.NotEmpty(t, status.SerialNumber)
		// Root CA is valid until 2036, so it must not need rotation today
		assert.False(t, status.NeedsRotation)

		// Rotation check should confirm no rotation needed
		res, err := manager.CheckAndRotateCA(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "NO_ROTATION_NEEDED", res.Status)
	})

	t.Run("File Persistence on Disk Rotation", func(t *testing.T) {
		tmpDir := t.TempDir()
		certPath := filepath.Join(tmpDir, "ca.crt")
		keyPath := filepath.Join(tmpDir, "ca.key")

		manager := NewCARotationManager(CARotationConfig{
			IntermediateCAName:   "oci-test-intermediate-ca",
			RenewalThresholdDays: 30,
			CACertFile:           certPath,
			CAKeyFile:            keyPath,
		})

		res, err := manager.CheckAndRotateCA(ctx)
		require.NoError(t, err)
		assert.Equal(t, "CA_ROTATED_SUCCESSFULLY", res.Status)

		// Files must exist on disk and be valid PEM
		certBytes, err := os.ReadFile(certPath)
		require.NoError(t, err)
		assert.Contains(t, string(certBytes), "-----BEGIN CERTIFICATE-----")

		keyBytes, err := os.ReadFile(keyPath)
		require.NoError(t, err)
		assert.Contains(t, string(keyBytes), "-----BEGIN EC PRIVATE KEY-----")

		// Re-initialize manager from the written files: should verify that no rotation is needed
		freshManager := NewCARotationManager(CARotationConfig{
			IntermediateCAName:   "oci-test-intermediate-ca",
			RenewalThresholdDays: 30,
			CACertFile:           certPath,
			CAKeyFile:            keyPath,
		})
		freshStatus, err := freshManager.GetCAStatus(ctx)
		require.NoError(t, err)
		assert.False(t, freshStatus.NeedsRotation)
		assert.Equal(t, res.NewSerialNumber, freshStatus.SerialNumber)
	})

	t.Run("Kubernetes API Secret, Certificate and Trust Bundle Sync", func(t *testing.T) {
		secretUpdated := false
		certsQueried := false
		bundleQueried := false

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

			if r.Method == http.MethodPut && r.URL.Path == "/api/v1/namespaces/cert-manager/secrets/oci-intermediate-ca-key-pair" {
				secretUpdated = true
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, "kubernetes.io/tls", body["type"])
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
				return
			}

			if r.Method == http.MethodGet && r.URL.Path == "/apis/cert-manager.io/v1/namespaces/cert-manager/certificates" {
				certsQueried = true
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"workforce-api-tls"}},{"metadata":{"name":"facilities-api-tls"}},{"metadata":{"name":"core-invest-api-tls"}}]}`))
				return
			}

			if r.Method == http.MethodGet && r.URL.Path == "/apis/trust.cert-manager.io/v1alpha1/bundles/oci-ca-bundle" {
				bundleQueried = true
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":{"conditions":[{"type":"Synced","status":"True","reason":"Synced"}]}}`))
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		manager := NewCARotationManager(CARotationConfig{
			IntermediateCAName:   "oci-intermediate-ca",
			RenewalThresholdDays: 30,
			IssuerNamespace:      "cert-manager",
			KubeAPIBaseURL:       server.URL,
			KubeToken:            "test-token",
			KubeHTTPClient:       server.Client(),
		})

		res, err := manager.CheckAndRotateCA(ctx)
		require.NoError(t, err)
		assert.Equal(t, "CA_ROTATED_SUCCESSFULLY", res.Status)
		assert.True(t, secretUpdated, "expected Kubernetes Secret to be updated via PUT")
		assert.True(t, certsQueried, "expected cert-manager Certificates to be queried via GET")
		assert.True(t, bundleQueried, "expected trust-manager Bundle to be verified via GET")
		assert.True(t, res.TrustBundleSynced)
		assert.Equal(t, 3, res.ReissuedCertificatesCount)

		status, err := manager.GetCAStatus(ctx)
		require.NoError(t, err)
		assert.True(t, status.TrustBundleSynced)
	})

	t.Run("Cryptographic X509 Certificate Chain Verification", func(t *testing.T) {
		now := time.Now().UTC()
		rootKey, rootCert, _, _, err := generateCAKeyPair("Root CA", now.AddDate(-1, 0, 0), now.AddDate(10, 0, 0), nil, nil)
		require.NoError(t, err)

		_, intermediateCert, _, _, err := generateCAKeyPair("Intermediate CA", now, now.AddDate(1, 0, 0), rootCert, rootKey)
		require.NoError(t, err)

		roots := x509.NewCertPool()
		roots.AddCert(rootCert)

		opts := x509.VerifyOptions{
			Roots:       roots,
			CurrentTime: now.Add(1 * time.Hour),
		}

		chains, err := intermediateCert.Verify(opts)
		require.NoError(t, err)
		require.Len(t, chains, 1)
		assert.Equal(t, "Intermediate CA", chains[0][0].Subject.CommonName)
		assert.Equal(t, "Root CA", chains[0][1].Subject.CommonName)
	})
}
