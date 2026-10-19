package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// DynamicCertReloader dynamically reloads X.509 certificates from disk
// when cert-manager or intermediate CA rotation updates the files,
// avoiding pod restarts and satisfying the zero static cert invariant.
type DynamicCertReloader struct {
	certFile    string
	keyFile     string
	mu          sync.RWMutex
	cachedCert  *tls.Certificate
	lastCertMod time.Time
	lastKeyMod  time.Time
}

// NewDynamicCertReloader creates a new certificate reloader for given file paths.
func NewDynamicCertReloader(certFile, keyFile string) (*DynamicCertReloader, error) {
	reloader := &DynamicCertReloader{
		certFile: certFile,
		keyFile:  keyFile,
	}

	// Verify we can load initial certificate if files already exist
	if certFile != "" && keyFile != "" {
		if _, err := reloader.ReloadIfNeeded(); err != nil {
			return nil, fmt.Errorf("failed initial certificate load (%s, %s): %w", certFile, keyFile, err)
		}
	}

	return reloader, nil
}

// ReloadIfNeeded checks whether certificate files have changed on disk and reloads them.
func (r *DynamicCertReloader) ReloadIfNeeded() (*tls.Certificate, error) {
	r.mu.RLock()
	certStat, errCert := os.Stat(r.certFile)
	keyStat, errKey := os.Stat(r.keyFile)
	isCurrent := errCert == nil && errKey == nil &&
		certStat.ModTime().Equal(r.lastCertMod) &&
		keyStat.ModTime().Equal(r.lastKeyMod) &&
		r.cachedCert != nil
	cached := r.cachedCert
	r.mu.RUnlock()

	if isCurrent {
		return cached, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double check under write lock
	certStat, errCert = os.Stat(r.certFile)
	if errCert != nil {
		return nil, fmt.Errorf("stat cert file %s: %w", r.certFile, errCert)
	}
	keyStat, errKey = os.Stat(r.keyFile)
	if errKey != nil {
		return nil, fmt.Errorf("stat key file %s: %w", r.keyFile, errKey)
	}

	if certStat.ModTime().Equal(r.lastCertMod) && keyStat.ModTime().Equal(r.lastKeyMod) && r.cachedCert != nil {
		return r.cachedCert, nil
	}

	cert, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	if err != nil {
		return nil, fmt.Errorf("load keypair (%s, %s): %w", r.certFile, r.keyFile, err)
	}

	r.cachedCert = &cert
	r.lastCertMod = certStat.ModTime()
	r.lastKeyMod = keyStat.ModTime()

	return r.cachedCert, nil
}

// GetCertificate implements the tls.Config GetCertificate callback for TLS servers.
func (r *DynamicCertReloader) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return r.ReloadIfNeeded()
}

// GetClientCertificate implements the tls.Config GetClientCertificate callback for TLS clients.
func (r *DynamicCertReloader) GetClientCertificate(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return r.ReloadIfNeeded()
}

// NewServerTLSConfig creates a server-side tls.Config enforcing mTLS with dynamic certificate reloading.
func NewServerTLSConfig(certFile, keyFile, clientCAFile string) (*tls.Config, error) {
	if certFile == "" || keyFile == "" || clientCAFile == "" {
		return nil, fmt.Errorf("certFile, keyFile, and clientCAFile are required to enforce mTLS")
	}

	reloader, err := NewDynamicCertReloader(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	caCertPEM, err := os.ReadFile(clientCAFile)
	if err != nil {
		return nil, fmt.Errorf("read client ca file %s: %w", clientCAFile, err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse client ca pem in %s", clientCAFile)
	}

	tlsConfig := &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: reloader.GetCertificate,
		ClientCAs:      caPool,
		ClientAuth:     tls.RequireAndVerifyClientCert,
	}

	return tlsConfig, nil
}

// NewClientTLSConfig creates a client-side tls.Config for mTLS with dynamic client cert reloading.
func NewClientTLSConfig(certFile, keyFile, serverCAFile string) (*tls.Config, error) {
	if certFile == "" || keyFile == "" || serverCAFile == "" {
		return nil, fmt.Errorf("certFile, keyFile, and serverCAFile are required for mTLS")
	}
	reloader, err := NewDynamicCertReloader(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		MinVersion:           tls.VersionTLS12,
		GetClientCertificate: reloader.GetClientCertificate,
	}

	if serverCAFile != "" {
		caCertPEM, err := os.ReadFile(serverCAFile)
		if err != nil {
			return nil, fmt.Errorf("read server ca file %s: %w", serverCAFile, err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCertPEM) {
			return nil, fmt.Errorf("failed to parse server ca pem in %s", serverCAFile)
		}
		tlsConfig.RootCAs = caPool
	}

	return tlsConfig, nil
}

// NewDynamicHTTPClient returns an *http.Client configured with dynamic mTLS client certificates.
func NewDynamicHTTPClient(certFile, keyFile, serverCAFile string, timeout time.Duration) (*http.Client, error) {
	tlsConfig, err := NewClientTLSConfig(certFile, keyFile, serverCAFile)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		TLSClientConfig:       tlsConfig,
		ResponseHeaderTimeout: timeout,
		ForceAttemptHTTP2:     true,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, nil
}
