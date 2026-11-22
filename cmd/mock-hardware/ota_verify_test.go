package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/cosignverify"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockHardware_VerifyAndFetchFirmware(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	payload := []byte("EDGE-CAMERA-FIRMWARE-IMAGE-v2.2.0")
	sum := sha256.Sum256(payload)
	checksum := hex.EncodeToString(sum[:])
	sig, err := cosignverify.SignBlob(privPEM, payload)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/valid.bin":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(payload)
		case "/tampered.bin":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("TAMPERED-EDGE-FIRMWARE"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	// Valid signed firmware succeeds
	err = FetchAndVerifyFirmwareRelease(context.Background(), srv.Client(), pubPEM, &facilitiesv1.FirmwareRelease{
		ReleaseID:       "rel-1",
		Version:         "2.2.0",
		FileURL:         srv.URL + "/valid.bin",
		Checksum:        checksum,
		CosignSignature: sig,
	})
	assert.NoError(t, err)

	// Unsigned firmware rejected
	err = FetchAndVerifyFirmwareRelease(context.Background(), srv.Client(), pubPEM, &facilitiesv1.FirmwareRelease{
		ReleaseID: "rel-unsigned",
		Version:   "2.2.0",
		FileURL:   srv.URL + "/valid.bin",
		Checksum:  checksum,
	})
	assert.Error(t, err)

	// Tampered payload rejected
	err = FetchAndVerifyFirmwareRelease(context.Background(), srv.Client(), pubPEM, &facilitiesv1.FirmwareRelease{
		ReleaseID:       "rel-tampered",
		Version:         "2.2.0",
		FileURL:         srv.URL + "/tampered.bin",
		Checksum:        checksum,
		CosignSignature: sig,
	})
	assert.Error(t, err)

	// Forged signature rejected
	err = VerifyFirmwarePayload(pubPEM, payload, checksum, "Zm9yZ2VkLXNpZ25hdHVyZQ==")
	assert.Error(t, err)
}
