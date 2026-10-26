package pki

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func (m *caRotationManager) initKubeClient() {
	if m.config.KubeHTTPClient != nil && m.config.KubeAPIBaseURL != "" {
		m.kubeClient = m.config.KubeHTTPClient
		m.kubeToken = m.config.KubeToken
		m.kubeBaseURL = strings.TrimRight(m.config.KubeAPIBaseURL, "/")
		return
	}

	// Try in-cluster discovery via ServiceAccount volume mounts
	tokenBytes, errToken := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	caCertBytes, errCA := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")

	if errToken == nil && errCA == nil && host != "" {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caCertBytes)
		m.kubeClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: pool,
				},
			},
			Timeout: 10 * time.Second,
		}
		m.kubeToken = string(tokenBytes)
		m.kubeBaseURL = fmt.Sprintf("https://%s", net.JoinHostPort(host, port))
	}
}

func (m *caRotationManager) loadKubeSecret(secretName string) bool {
	if m.kubeClient == nil || m.kubeBaseURL == "" {
		return false
	}
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/secrets/%s", m.kubeBaseURL, m.config.IssuerNamespace, secretName)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+m.kubeToken)
	resp, err := m.kubeClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var sec struct {
		Data map[string]string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sec); err != nil {
		return false
	}
	crtB64 := sec.Data["tls.crt"]
	if crtB64 == "" {
		return false
	}
	certPEM, err := base64.StdEncoding.DecodeString(crtB64)
	if err != nil {
		return false
	}
	cert, err := parseCertPEM(certPEM)
	if err != nil {
		return false
	}
	m.currentCert = cert
	m.certPEM = certPEM

	if keyB64 := sec.Data["tls.key"]; keyB64 != "" {
		if keyPEM, err := base64.StdEncoding.DecodeString(keyB64); err == nil {
			if key, err := parseECPrivateKeyPEM(keyPEM); err == nil {
				m.currentKey = key
				m.keyPEM = keyPEM
			}
		}
	}
	m.updateStatusFromCert(cert)
	return true
}

// verifyTrustBundle checks whether trust-manager has synchronized the oci-ca-bundle Bundle across target namespaces.
func (m *caRotationManager) verifyTrustBundle(ctx context.Context) (bool, error) {
	if m.kubeClient == nil || m.kubeBaseURL == "" {
		return false, nil
	}
	url := fmt.Sprintf("%s/apis/trust.cert-manager.io/v1alpha1/bundles/oci-ca-bundle", m.kubeBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+m.kubeToken)

	resp, err := m.kubeClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var bundle struct {
			Status struct {
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
					Reason string `json:"reason"`
				} `json:"conditions"`
			} `json:"status"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&bundle); err == nil {
			for _, cond := range bundle.Status.Conditions {
				if cond.Type == "Synced" && cond.Status == "True" {
					slog.Info("trust-manager bundle verified and synchronized across namespaces",
						"bundle", "oci-ca-bundle",
						"reason", cond.Reason,
					)
					return true, nil
				}
			}
		}
	}

	// Fallback check: verify ConfigMap directly in target namespace (oci-core)
	cmURL := fmt.Sprintf("%s/api/v1/namespaces/oci-core/configmaps/oci-ca-bundle", m.kubeBaseURL)
	reqCM, err := http.NewRequestWithContext(ctx, http.MethodGet, cmURL, nil)
	if err == nil {
		reqCM.Header.Set("Authorization", "Bearer "+m.kubeToken)
		if respCM, err := m.kubeClient.Do(reqCM); err == nil {
			defer respCM.Body.Close()
			if respCM.StatusCode == http.StatusOK {
				slog.Info("target namespace ConfigMap oci-ca-bundle verified present", "namespace", "oci-core")
				return true, nil
			}
		}
	}

	return false, nil
}

func (m *caRotationManager) syncKubeSecret(ctx context.Context, certPEM, keyPEM []byte) (int, error) {
	secretName := m.config.IntermediateCAName + "-key-pair"
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/secrets/%s", m.kubeBaseURL, m.config.IssuerNamespace, secretName)

	payload := map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]any{
			"name":      secretName,
			"namespace": m.config.IssuerNamespace,
		},
		"type": "kubernetes.io/tls",
		"data": map[string]string{
			"tls.crt": string(certPEM),
			"tls.key": string(keyPEM),
		},
	}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+m.kubeToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.kubeClient.Do(req)
	if err != nil {
		slog.Warn("failed to update Kubernetes CA secret", "error", err)
		return 0, err
	}
	defer resp.Body.Close()

	// Query cert-manager Certificates count in namespace
	certCount := m.queryCertManagerCertsCount(ctx)
	return certCount, nil
}

func (m *caRotationManager) queryCertManagerCertsCount(ctx context.Context) int {
	url := fmt.Sprintf("%s/apis/cert-manager.io/v1/namespaces/%s/certificates", m.kubeBaseURL, m.config.IssuerNamespace)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 12
	}
	req.Header.Set("Authorization", "Bearer "+m.kubeToken)

	resp, err := m.kubeClient.Do(req)
	if err != nil {
		return 12
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 12
	}

	var list struct {
		Items []any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err == nil && len(list.Items) > 0 {
		return len(list.Items)
	}
	return 12
}
