package httputil

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/orange-cat-investments/oci/pkg/mtls"
)

// WriteJSON encodes payload as JSON with the given HTTP status code.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteError writes a standard JSON error response {"error": msg} with the given HTTP status code.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// RenderError writes a standard JSON error response from an error value.
func RenderError(w http.ResponseWriter, status int, err error) {
	if err == nil {
		WriteError(w, status, http.StatusText(status))
		return
	}
	WriteError(w, status, err.Error())
}

// DecodeJSON decodes the JSON request body into dst.
func DecodeJSON[T any](r *http.Request, dst *T) error {
	if r == nil || r.Body == nil {
		return errors.New("empty request body")
	}
	return json.NewDecoder(r.Body).Decode(dst)
}

// HealthzHandler returns a standard HTTP health check handler.
func HealthzHandler(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// NewDynamicMTLSClient returns a dynamic mTLS-enabled *http.Client if TLS env vars are present,
// or falls back to a standard *http.Client with the specified timeout.
func NewDynamicMTLSClient(timeout time.Duration, logger *slog.Logger) *http.Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	tlsCert := os.Getenv("TLS_CERT_FILE")
	tlsKey := os.Getenv("TLS_KEY_FILE")
	clientCA := os.Getenv("CA_CERT_FILE")
	if clientCA == "" {
		clientCA = os.Getenv("CLIENT_CA_FILE")
	}
	if clientCA == "" {
		clientCA = os.Getenv("CA_CERT_PATH")
	}
	if tlsCert != "" && tlsKey != "" && clientCA != "" {
		if c, err := mtls.NewDynamicHTTPClient(tlsCert, tlsKey, clientCA, timeout); err == nil {
			return c
		} else if logger != nil {
			logger.Warn("failed to configure dynamic mTLS HTTP client, using default HTTP client", "error", err)
		}
	}
	return &http.Client{Timeout: timeout}
}

// ListenAndServeGraceful starts an HTTP server with optional mTLS and handles SIGINT/SIGTERM graceful shutdown.
func ListenAndServeGraceful(serviceName, port string, handler http.Handler, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	tlsCert := os.Getenv("TLS_CERT_FILE")
	tlsKey := os.Getenv("TLS_KEY_FILE")
	clientCA := os.Getenv("CLIENT_CA_FILE")
	if clientCA == "" {
		clientCA = os.Getenv("CA_CERT_FILE")
	}

	useTLS := false
	if tlsCert != "" && tlsKey != "" && clientCA != "" {
		if tlsCfg, err := mtls.NewServerTLSConfig(tlsCert, tlsKey, clientCA); err == nil {
			server.TLSConfig = tlsCfg
			useTLS = true
		} else {
			logger.Warn("failed to load server mTLS config, falling back to plaintext HTTP", "service", serviceName, "error", err)
		}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info(serviceName+" listening", "port", port, "mtls", useTLS)
		var err error
		if useTLS {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(serviceName+" HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down " + serviceName + "...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed graceful shutdown", "service", serviceName, "error", err)
	}
	logger.Info(serviceName + " stopped gracefully")
}
