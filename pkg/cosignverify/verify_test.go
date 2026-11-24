package cosignverify

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateECDSAPEMPair(t *testing.T) (privPEM, pubPEM []byte) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)

	privPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})
	pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return privPEM, pubPEM
}

func generateEd25519PEMPair(t *testing.T) (privPEM, pubPEM []byte) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	privPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})
	pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return privPEM, pubPEM
}

func TestVerifyBlobSignature_ECDSA(t *testing.T) {
	privPEM, pubPEM := generateECDSAPEMPair(t)
	_, otherPubPEM := generateECDSAPEMPair(t)

	payload := []byte("OCI-FIRMWARE-EDGE-CAMERA-v2.2.0-BIN")
	sig, err := SignBlob(privPEM, payload)
	require.NoError(t, err)

	assert.NoError(t, VerifyBlobSignature(pubPEM, payload, sig))
	assert.Error(t, VerifyBlobSignature(pubPEM, []byte("TAMPERED-FIRMWARE"), sig))
	assert.Error(t, VerifyBlobSignature(otherPubPEM, payload, sig))
	assert.Error(t, VerifyBlobSignature(pubPEM, payload, ""))
	assert.Error(t, VerifyBlobSignature(pubPEM, payload, "not-valid-base64!!!"))
}

func TestVerifyBlobSignature_Ed25519(t *testing.T) {
	privPEM, pubPEM := generateEd25519PEMPair(t)
	payload := []byte("OCI-PEBBLE-WATCH-v1.5.0-PBW")

	sig, err := SignBlob(privPEM, payload)
	require.NoError(t, err)

	assert.NoError(t, VerifyBlobSignature(pubPEM, payload, sig))
	assert.Error(t, VerifyBlobSignature(pubPEM, []byte("TAMPERED-PBW"), sig))
}

func TestVerifySHA256Hex(t *testing.T) {
	payload := []byte("firmware-binary-content")
	sum := sha256.Sum256(payload)
	expected := hex.EncodeToString(sum[:])

	assert.NoError(t, VerifySHA256Hex(payload, expected))
	assert.NoError(t, VerifySHA256Hex(payload, "sha256:"+expected))
	assert.Error(t, VerifySHA256Hex([]byte("mutated"), expected))
	assert.Error(t, VerifySHA256Hex(payload, ""))
}

func TestLoadPublicKeyFromEnvOrPath(t *testing.T) {
	_, pubPEM := generateECDSAPEMPair(t)
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "cosign.pub")
	require.NoError(t, os.WriteFile(keyPath, pubPEM, 0o600))

	got, err := LoadPublicKeyFromEnvOrPath(keyPath)
	require.NoError(t, err)
	assert.Equal(t, pubPEM, got)

	t.Setenv(DefaultPublicKeyEnv, keyPath)
	gotEnv, err := LoadPublicKeyFromEnvOrPath("")
	require.NoError(t, err)
	assert.Equal(t, pubPEM, gotEnv)
}
