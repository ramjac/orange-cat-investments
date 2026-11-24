package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cosignverify"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

const maxDeviceFirmwareBytes = 64 * 1024 * 1024

// VerifyFirmwarePayload validates both SHA-256 digest and Cosign asymmetric signature
// against pubKeyPEM before allowing the emulated edge camera hardware to apply an OTA update.
func VerifyFirmwarePayload(pubKeyPEM, payload []byte, expectedChecksum, cosignSignature string) error {
	if err := cosignverify.VerifySHA256Hex(payload, expectedChecksum); err != nil {
		return fmt.Errorf("mock-hardware checksum verification failed: %w", err)
	}
	if err := cosignverify.VerifyBlobSignature(pubKeyPEM, payload, cosignSignature); err != nil {
		return fmt.Errorf("mock-hardware cosign verification failed: %w", err)
	}
	return nil
}

func resolveDeviceFirmwareURL(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	internalBase := strings.TrimRight(os.Getenv("FORGEJO_INTERNAL_URL"), "/")
	if internalBase == "" {
		internalBase = "http://forgejo-service.oci-internal.svc.cluster.local:3000"
	}
	for _, ext := range []string{"https://forgejo.oci.local", "http://forgejo.oci.local"} {
		if strings.HasPrefix(trimmed, ext+"/") || trimmed == ext {
			return internalBase + strings.TrimPrefix(trimmed, ext)
		}
	}
	return trimmed
}

// FetchAndVerifyFirmwareRelease downloads a firmware artifact from its package registry URL
// and verifies SHA-256 checksum + Cosign signature against the configured public key.
func FetchAndVerifyFirmwareRelease(ctx context.Context, httpClient *http.Client, pubKeyPEM []byte, rel *facilitiesv1.FirmwareRelease) error {
	if rel == nil {
		return errors.New("mock-hardware: nil firmware release")
	}
	if strings.TrimSpace(rel.CosignSignature) == "" {
		return fmt.Errorf("mock-hardware: firmware release %s (%s) is unsigned", rel.ReleaseID, rel.Version)
	}
	targetURL := resolveDeviceFirmwareURL(rel.FileURL)
	parsed, err := url.Parse(targetURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("mock-hardware: invalid firmware URL %q", rel.FileURL)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil) // #nosec G107 -- URL scheme/host validated; payload verified via SHA-256 + Cosign
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mock-hardware: download firmware %s: %w", parsed.String(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mock-hardware: package registry returned HTTP %d", resp.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxDeviceFirmwareBytes+1))
	if err != nil {
		return err
	}
	if len(payload) > maxDeviceFirmwareBytes {
		return errors.New("mock-hardware: firmware payload exceeds 64 MiB limit")
	}
	return VerifyFirmwarePayload(pubKeyPEM, payload, rel.Checksum, rel.CosignSignature)
}

func runFirmwareVerificationLoop(ctx context.Context, bllClient internalbllv1.InternalBLLServiceClient, logger *slog.Logger) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	httpClient := &http.Client{Timeout: 15 * time.Second}
	verified := make(map[string]bool)

	checkOnce := func() {
		pubKeyPEM, err := cosignverify.LoadPublicKeyFromEnvOrPath("")
		if err != nil {
			return
		}
		reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		resp, err := bllClient.ListFirmwareReleases(reqCtx, &facilitiesv1.ListFirmwareReleasesRequest{DeviceType: "edge_camera"})
		if err != nil || resp == nil {
			return
		}
		for _, rel := range resp.Releases {
			if rel == nil || rel.ReleaseID == "" || verified[rel.ReleaseID] || strings.TrimSpace(rel.CosignSignature) == "" {
				continue
			}
			if err := FetchAndVerifyFirmwareRelease(reqCtx, httpClient, pubKeyPEM, rel); err != nil {
				logger.Warn("edge hardware rejected firmware release during OTA verification",
					"release_id", rel.ReleaseID,
					"version", rel.Version,
					"error", err,
				)
				continue
			}
			verified[rel.ReleaseID] = true
			logger.Info("edge hardware verified signed OTA firmware release",
				"release_id", rel.ReleaseID,
				"version", rel.Version,
				"file_url", rel.FileURL,
			)
		}
	}

	checkOnce()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkOnce()
		}
	}
}
