package mtls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
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

	_, err = NewDynamicCAPoolReloader("")
	assert.Error(t, err)
}

func generateCA(t *testing.T, commonName string) ([]byte, []byte, *x509.Certificate, *rsa.PrivateKey) {
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
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	parsed, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return certPEM, keyPEM, parsed, priv
}

func generateSignedCert(t *testing.T, commonName string, parent *x509.Certificate, parentKey *rsa.PrivateKey) ([]byte, []byte) {
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
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost", "oci.local"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, parent, &priv.PublicKey, parentKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	return certPEM, keyPEM
}

func TestDynamicCAPoolReloader(t *testing.T) {
	tempDir := t.TempDir()
	caFile := filepath.Join(tempDir, "ca.crt")

	ca1PEM, _, _, _ := generateCA(t, "root-ca-v1")
	require.NoError(t, os.WriteFile(caFile, ca1PEM, 0644))

	reloader, err := NewDynamicCAPoolReloader(caFile)
	require.NoError(t, err)
	require.NotNil(t, reloader)

	pool1, err := reloader.GetPool()
	require.NoError(t, err)
	require.NotNil(t, pool1)

	// Sleep briefly to ensure filesystem modtime ticks
	time.Sleep(50 * time.Millisecond)

	// Rotate CA bundle on disk
	ca2PEM, _, _, _ := generateCA(t, "root-ca-v2")
	require.NoError(t, os.WriteFile(caFile, ca2PEM, 0644))

	pool2, err := reloader.ReloadIfNeeded()
	require.NoError(t, err)
	require.NotNil(t, pool2)
	assert.NotSame(t, pool1, pool2)
}

func TestDynamicMTLSEndToEndRotation(t *testing.T) {
	tempDir := t.TempDir()
	caFile := filepath.Join(tempDir, "ca.crt")
	serverCertFile := filepath.Join(tempDir, "server.crt")
	serverKeyFile := filepath.Join(tempDir, "server.key")
	clientCertFile := filepath.Join(tempDir, "client.crt")
	clientKeyFile := filepath.Join(tempDir, "client.key")

	// Phase 1: Initialize PKI with Root CA v1
	ca1PEM, _, ca1Cert, ca1Key := generateCA(t, "root-ca-v1")
	sCert1PEM, sKey1PEM := generateSignedCert(t, "server-v1", ca1Cert, ca1Key)
	cCert1PEM, cKey1PEM := generateSignedCert(t, "client-v1", ca1Cert, ca1Key)

	require.NoError(t, os.WriteFile(caFile, ca1PEM, 0644))
	require.NoError(t, os.WriteFile(serverCertFile, sCert1PEM, 0600))
	require.NoError(t, os.WriteFile(serverKeyFile, sKey1PEM, 0600))
	require.NoError(t, os.WriteFile(clientCertFile, cCert1PEM, 0600))
	require.NoError(t, os.WriteFile(clientKeyFile, cKey1PEM, 0600))

	serverTLS, err := NewServerTLSConfig(serverCertFile, serverKeyFile, caFile)
	require.NoError(t, err)

	clientTLS, err := NewClientTLSConfig(clientCertFile, clientKeyFile, caFile)
	require.NoError(t, err)

	// Start an HTTPS test server enforcing mTLS
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	server := &http.Server{
		Handler:   mux,
		TLSConfig: serverTLS,
	}

	go func() {
		_ = server.ServeTLS(listener, "", "")
	}()
	defer server.Close()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: clientTLS,
		},
		Timeout: 2 * time.Second,
	}

	url := "https://" + listener.Addr().String() + "/ping"

	// 1. Initial request under CA v1 succeeds
	resp, err := client.Get(url)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Phase 2: Rotate CA and leaf certificates on disk
	time.Sleep(50 * time.Millisecond)
	ca2PEM, _, ca2Cert, ca2Key := generateCA(t, "root-ca-v2")
	sCert2PEM, sKey2PEM := generateSignedCert(t, "server-v2", ca2Cert, ca2Key)
	cCert2PEM, cKey2PEM := generateSignedCert(t, "client-v2", ca2Cert, ca2Key)

	require.NoError(t, os.WriteFile(caFile, ca2PEM, 0644))
	require.NoError(t, os.WriteFile(serverCertFile, sCert2PEM, 0600))
	require.NoError(t, os.WriteFile(serverKeyFile, sKey2PEM, 0600))
	require.NoError(t, os.WriteFile(clientCertFile, cCert2PEM, 0600))
	require.NoError(t, os.WriteFile(clientKeyFile, cKey2PEM, 0600))

	// Close idle keepalive connections to force a new TLS handshake
	client.Transport.(*http.Transport).CloseIdleConnections()

	// 2. Request under CA v2 succeeds dynamically without restarting server or reconstructing client!
	resp2, err := client.Get(url)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	resp2.Body.Close()

	// Phase 3: Rogue client certificate signed by untrusted CA should be rejected
	_, _, rogueCACert, rogueCAKey := generateCA(t, "rogue-ca")
	rogueCertPEM, rogueKeyPEM := generateSignedCert(t, "rogue-client", rogueCACert, rogueCAKey)

	rogueCertFile := filepath.Join(tempDir, "rogue.crt")
	rogueKeyFile := filepath.Join(tempDir, "rogue.key")
	require.NoError(t, os.WriteFile(rogueCertFile, rogueCertPEM, 0600))
	require.NoError(t, os.WriteFile(rogueKeyFile, rogueKeyPEM, 0600))

	rogueTLS, err := NewClientTLSConfig(rogueCertFile, rogueKeyFile, caFile)
	require.NoError(t, err)

	rogueClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: rogueTLS,
		},
		Timeout: 2 * time.Second,
	}

	_, err = rogueClient.Get(url)
	assert.Error(t, err, "server must reject client signed by untrusted CA")
}

func TestStartHealthListener(t *testing.T) {
	// Find free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port)
	_ = l.Close()

	srv := StartHealthListener(port, "50051", nil, "test-service")
	require.NotNil(t, srv)
	defer srv.Close()

	// Wait briefly for listener to be active
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:" + port + "/healthz")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	respLive, err := http.Get("http://127.0.0.1:" + port + "/livez")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, respLive.StatusCode)
	respLive.Body.Close()

	// When healthPort == primaryPort, it should return nil
	assert.Nil(t, StartHealthListener("50051", "50051", nil, "test-service"))
	// When healthPort == "", it should return nil
	assert.Nil(t, StartHealthListener("", "50051", nil, "test-service"))
}
