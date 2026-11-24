// Package cosignverify provides pure-Go cryptographic verification for Cosign
// blob signatures (ECDSA P-256 SHA-256 and Ed25519) and SHA-256 content digests
// used by OCI's OTA firmware pipeline and edge hardware verification.
package cosignverify

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
)

// DefaultPublicKeyEnv is the environment variable pointing to the mounted Cosign public key file.
const DefaultPublicKeyEnv = "COSIGN_PUB_KEY_FILE"

// DefaultPublicKeyPath is the standard Kubernetes ConfigMap mount path for cosign.pub.
const DefaultPublicKeyPath = "/etc/oci/cosign/cosign.pub"

// ParsePublicKeyPEM parses a PEM-encoded PKIX public key (ECDSA or Ed25519).
func ParsePublicKeyPEM(pemBytes []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("cosignverify: failed to decode PEM public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("cosignverify: parse PKIX public key: %w", err)
	}
	switch k := pub.(type) {
	case *ecdsa.PublicKey, ed25519.PublicKey:
		return k, nil
	default:
		return nil, fmt.Errorf("cosignverify: unsupported public key type %T", pub)
	}
}

// LoadPublicKeyFromEnvOrPath resolves a Cosign public key PEM from the explicit path,
// COSIGN_PUB_KEY_FILE environment variable, or DefaultPublicKeyPath.
func LoadPublicKeyFromEnvOrPath(explicitPath string) ([]byte, error) {
	candidates := []string{explicitPath, os.Getenv(DefaultPublicKeyEnv), DefaultPublicKeyPath}
	for _, p := range candidates {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if err == nil && len(strings.TrimSpace(string(data))) > 0 {
			return data, nil
		}
	}
	return nil, errors.New("cosignverify: no cosign public key configured or readable")
}

// VerifySHA256Hex verifies that the SHA-256 digest of payload matches expectedHex.
func VerifySHA256Hex(payload []byte, expectedHex string) error {
	expected := strings.ToLower(strings.TrimSpace(expectedHex))
	expected = strings.TrimPrefix(expected, "sha256:")
	if expected == "" {
		return errors.New("cosignverify: empty expected sha256 checksum")
	}
	sum := sha256.Sum256(payload)
	actual := hex.EncodeToString(sum[:])
	if actual != expected {
		return fmt.Errorf("cosignverify: sha256 mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

// VerifyBlobSignature verifies a base64-encoded Cosign signature over payload using pubKeyPEM.
// Supports both ECDSA (ASN.1 DER or raw R||S over SHA-256) and Ed25519 public keys.
func VerifyBlobSignature(pubKeyPEM, payload []byte, signatureBase64 string) error {
	sigB64 := strings.TrimSpace(signatureBase64)
	if sigB64 == "" {
		return errors.New("cosignverify: missing cosign signature")
	}
	sigBytes, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return fmt.Errorf("cosignverify: invalid base64 signature: %w", err)
	}

	pub, err := ParsePublicKeyPEM(pubKeyPEM)
	if err != nil {
		return err
	}

	switch k := pub.(type) {
	case *ecdsa.PublicKey:
		digest := sha256.Sum256(payload)
		if ecdsa.VerifyASN1(k, digest[:], sigBytes) {
			return nil
		}
		// Fallback for raw IEEE P1363 R||S signatures.
		coordLen := (k.Curve.Params().BitSize + 7) / 8
		if len(sigBytes) == 2*coordLen {
			r := new(big.Int).SetBytes(sigBytes[:coordLen])
			s := new(big.Int).SetBytes(sigBytes[coordLen:])
			if ecdsa.Verify(k, digest[:], r, s) {
				return nil
			}
		}
		return errors.New("cosignverify: ECDSA signature verification failed")
	case ed25519.PublicKey:
		if !ed25519.Verify(k, payload, sigBytes) {
			return errors.New("cosignverify: Ed25519 signature verification failed")
		}
		return nil
	default:
		return fmt.Errorf("cosignverify: unsupported public key type %T", pub)
	}
}

// SignBlob signs payload using a PEM-encoded PKCS8 or SEC1 ECDSA/Ed25519 private key
// and returns the base64-encoded signature matching `cosign sign-blob --b64=true`.
func SignBlob(privKeyPEM, payload []byte) (string, error) {
	block, _ := pem.Decode(privKeyPEM)
	if block == nil {
		return "", errors.New("cosignverify: failed to decode PEM private key")
	}

	var priv crypto.PrivateKey
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		priv = k
	} else if k, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		priv = k
	} else {
		return "", errors.New("cosignverify: unsupported private key format (expected PKCS8 or EC)")
	}

	switch k := priv.(type) {
	case *ecdsa.PrivateKey:
		digest := sha256.Sum256(payload)
		sig, err := ecdsa.SignASN1(rand.Reader, k, digest[:])
		if err != nil {
			return "", fmt.Errorf("cosignverify: sign ECDSA: %w", err)
		}
		return base64.StdEncoding.EncodeToString(sig), nil
	case ed25519.PrivateKey:
		sig := ed25519.Sign(k, payload)
		return base64.StdEncoding.EncodeToString(sig), nil
	default:
		return "", fmt.Errorf("cosignverify: unsupported private key type %T", priv)
	}
}
