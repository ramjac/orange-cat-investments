package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
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

// DynamicCAPoolReloader dynamically reloads X.509 CA certificate bundles from disk
// when cert-manager, trust-manager, or CA rotation updates the files,
// avoiding pod restarts and satisfying the zero static cert invariant.
type DynamicCAPoolReloader struct {
	caFile     string
	mu         sync.RWMutex
	cachedPool *x509.CertPool
	lastMod    time.Time
}

// NewDynamicCAPoolReloader creates a new CA pool reloader for a given file path.
func NewDynamicCAPoolReloader(caFile string) (*DynamicCAPoolReloader, error) {
	if caFile == "" {
		return nil, fmt.Errorf("caFile path cannot be empty")
	}
	reloader := &DynamicCAPoolReloader{
		caFile: caFile,
	}
	if _, err := reloader.ReloadIfNeeded(); err != nil {
		return nil, fmt.Errorf("failed initial CA pool load (%s): %w", caFile, err)
	}
	return reloader, nil
}

// ReloadIfNeeded checks whether the CA bundle file has changed on disk and reloads the x509.CertPool.
func (r *DynamicCAPoolReloader) ReloadIfNeeded() (*x509.CertPool, error) {
	r.mu.RLock()
	caStat, errStat := os.Stat(r.caFile)
	isCurrent := errStat == nil &&
		caStat.ModTime().Equal(r.lastMod) &&
		r.cachedPool != nil
	cached := r.cachedPool
	r.mu.RUnlock()

	if isCurrent {
		return cached, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	caStat, errStat = os.Stat(r.caFile)
	if errStat != nil {
		return nil, fmt.Errorf("stat ca file %s: %w", r.caFile, errStat)
	}

	if caStat.ModTime().Equal(r.lastMod) && r.cachedPool != nil {
		return r.cachedPool, nil
	}

	caCertPEM, err := os.ReadFile(r.caFile)
	if err != nil {
		return nil, fmt.Errorf("read ca file %s: %w", r.caFile, err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse valid CA certificates from PEM in %s", r.caFile)
	}

	r.cachedPool = pool
	r.lastMod = caStat.ModTime()

	return r.cachedPool, nil
}

// GetPool returns the latest active x509.CertPool.
func (r *DynamicCAPoolReloader) GetPool() (*x509.CertPool, error) {
	return r.ReloadIfNeeded()
}

// NewServerTLSConfig creates a server-side tls.Config enforcing mTLS with dynamic certificate reloading
// and dynamic CA pool hot-swapping for zero-downtime certificate rollovers.
func NewServerTLSConfig(certFile, keyFile, clientCAFile string) (*tls.Config, error) {
	if certFile == "" || keyFile == "" || clientCAFile == "" {
		return nil, fmt.Errorf("certFile, keyFile, and clientCAFile are required to enforce mTLS")
	}

	reloader, err := NewDynamicCertReloader(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	caReloader, err := NewDynamicCAPoolReloader(clientCAFile)
	if err != nil {
		return nil, err
	}

	initialPool, err := caReloader.GetPool()
	if err != nil {
		return nil, err
	}

	baseConfig := &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: reloader.GetCertificate,
		ClientAuth:     tls.RequireAndVerifyClientCert,
	}

	tlsConfig := baseConfig.Clone()
	tlsConfig.ClientCAs = initialPool
	tlsConfig.GetConfigForClient = func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
		pool, err := caReloader.ReloadIfNeeded()
		if err != nil {
			return nil, fmt.Errorf("reload client CAs: %w", err)
		}
		cfg := baseConfig.Clone()
		cfg.ClientCAs = pool
		return cfg, nil
	}

	return tlsConfig, nil
}

// NewClientTLSConfig creates a client-side tls.Config for mTLS with dynamic client cert reloading
// and dynamic server CA pool hot-swapping for zero-downtime certificate rollovers.
func NewClientTLSConfig(certFile, keyFile, serverCAFile string) (*tls.Config, error) {
	if certFile == "" || keyFile == "" || serverCAFile == "" {
		return nil, fmt.Errorf("certFile, keyFile, and serverCAFile are required for mTLS")
	}
	reloader, err := NewDynamicCertReloader(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	caReloader, err := NewDynamicCAPoolReloader(serverCAFile)
	if err != nil {
		return nil, err
	}

	initialPool, err := caReloader.GetPool()
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		MinVersion:           tls.VersionTLS12,
		GetClientCertificate: reloader.GetClientCertificate,
		RootCAs:              initialPool,
		InsecureSkipVerify:   true, // Dynamic chain verification handled strictly in VerifyConnection
		VerifyConnection: func(cs tls.ConnectionState) error {
			pool, err := caReloader.ReloadIfNeeded()
			if err != nil {
				return fmt.Errorf("reload server CA pool: %w", err)
			}
			if len(cs.PeerCertificates) == 0 {
				return fmt.Errorf("no peer certificates presented by server")
			}
			intermediates := x509.NewCertPool()
			for _, c := range cs.PeerCertificates[1:] {
				intermediates.AddCert(c)
			}
			opts := x509.VerifyOptions{
				Roots:         pool,
				Intermediates: intermediates,
			}
			if _, err := cs.PeerCertificates[0].Verify(opts); err != nil {
				return fmt.Errorf("mTLS peer certificate chain verification failed: %w", err)
			}
			if cs.ServerName != "" {
				if err := cs.PeerCertificates[0].VerifyHostname(cs.ServerName); err != nil {
					return fmt.Errorf("mTLS hostname verification failed for %s: %w", cs.ServerName, err)
				}
			}
			return nil
		},
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

// StartHealthListener starts an optional unauthenticated plaintext health and liveness listener on healthPort
// if healthPort is specified and differs from the primary mTLS business port. This decouples Kubernetes
// node probes from internal mTLS client certificate requirements.
func StartHealthListener(healthPort, primaryPort string, logger *slog.Logger, serviceName string) *http.Server {
	if healthPort == "" || healthPort == primaryPort {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"ok","service":"%s"}`+"\n", serviceName)
	})
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	srv := &http.Server{
		Addr:         ":" + healthPort,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	go func() {
		if logger != nil {
			logger.Info("unauthenticated health probe listener started", "service", serviceName, "port", healthPort)
		}
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			if logger != nil {
				logger.Error("health probe server error", "service", serviceName, "error", err)
			}
		}
	}()
	return srv
}
