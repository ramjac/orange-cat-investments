package mtls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestCert(t *testing.T, commonName string) ([]byte, []byte) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost", "oci.local"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	return certPEM, keyPEM
}

func TestDynamicCertReloader(t *testing.T) {
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "tls.crt")
	keyFile := filepath.Join(tempDir, "tls.key")

	cert1PEM, key1PEM := generateTestCert(t, "service-v1")
	require.NoError(t, os.WriteFile(certFile, cert1PEM, 0600))
	require.NoError(t, os.WriteFile(keyFile, key1PEM, 0600))

	reloader, err := NewDynamicCertReloader(certFile, keyFile)
	require.NoError(t, err)
	require.NotNil(t, reloader)

	// Test GetCertificate
	cert, err := reloader.GetCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, cert)
	parsed1, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	assert.Equal(t, "service-v1", parsed1.Subject.CommonName)

	// Test GetClientCertificate
	clientCert, err := reloader.GetClientCertificate(nil)
	require.NoError(t, err)
	assert.Equal(t, cert, clientCert)

	// Sleep slightly to guarantee modtime difference on filesystem
	time.Sleep(50 * time.Millisecond)

	// Simulate cert-manager rotating certificate on disk
	cert2PEM, key2PEM := generateTestCert(t, "service-v2")
	require.NoError(t, os.WriteFile(certFile, cert2PEM, 0600))
	require.NoError(t, os.WriteFile(keyFile, key2PEM, 0600))

	// Next GetCertificate call should dynamically reload the rotated certificate without restart
	reloadedCert, err := reloader.GetCertificate(nil)
	require.NoError(t, err)
	parsed2, err := x509.ParseCertificate(reloadedCert.Certificate[0])
	require.NoError(t, err)
	assert.Equal(t, "service-v2", parsed2.Subject.CommonName)
}

func TestServerAndClientTLSConfig(t *testing.T) {
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "tls.crt")
	keyFile := filepath.Join(tempDir, "tls.key")
	caFile := filepath.Join(tempDir, "ca.crt")

	certPEM, keyPEM := generateTestCert(t, "mtls-node")
	require.NoError(t, os.WriteFile(certFile, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyFile, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caFile, certPEM, 0600))

	serverTLS, err := NewServerTLSConfig(certFile, keyFile, caFile)
	require.NoError(t, err)
	assert.NotNil(t, serverTLS.GetCertificate)
	assert.NotNil(t, serverTLS.ClientCAs)

	clientTLS, err := NewClientTLSConfig(certFile, keyFile, caFile)
	require.NoError(t, err)
	assert.NotNil(t, clientTLS.GetClientCertificate)
	assert.NotNil(t, clientTLS.RootCAs)

	httpClient, err := NewDynamicHTTPClient(certFile, keyFile, caFile, 5*time.Second)
	require.NoError(t, err)
	assert.NotNil(t, httpClient)

	// Validate failing closed when required certs/CAs are missing
	_, err = NewServerTLSConfig("", keyFile, caFile)
	assert.Error(t, err)
	_, err = NewServerTLSConfig(certFile, keyFile, "")
	assert.Error(t, err)

	_, err = NewClientTLSConfig(certFile, "", caFile)
	assert.Error(t, err)
	_, err = NewClientTLSConfig(certFile, keyFile, "")
	assert.Error(t, err)
}
