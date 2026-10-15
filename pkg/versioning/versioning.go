package versioning

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	// vendorMediaRegex matches MIME types like "application/vnd.oci.v1+json"
	vendorMediaRegex = regexp.MustCompile(`^application/vnd\.oci\.v([0-9]+)\+json$`)
)

// Config configures API versioning middleware.
type Config struct {
	DefaultVersion    string
	SupportedVersions []string
}

// DefaultConfig returns default configuration supporting v1.
func DefaultConfig() Config {
	return Config{
		DefaultVersion:    "v1",
		SupportedVersions: []string{"v1"},
	}
}

// Middleware returns an HTTP middleware enforcing API versioning and content negotiation.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	if cfg.DefaultVersion == "" {
		cfg.DefaultVersion = "v1"
	}
	if len(cfg.SupportedVersions) == 0 {
		cfg.SupportedVersions = []string{"v1"}
	}

	supportedMap := make(map[string]bool)
	for _, v := range cfg.SupportedVersions {
		norm := normalizeVersion(v)
		supportedMap[norm] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedVersion := extractRequestedVersion(r)

			// If no explicit version requested, default to configured default version
			if requestedVersion == "" {
				requestedVersion = normalizeVersion(cfg.DefaultVersion)
			} else {
				requestedVersion = normalizeVersion(requestedVersion)
			}

			// Validate if requested version is supported
			if !supportedMap[requestedVersion] {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Vary", "Accept, X-API-Version")
				w.WriteHeader(http.StatusNotAcceptable)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error":              fmt.Sprintf("unsupported API version: %s", requestedVersion),
					"supported_versions": cfg.SupportedVersions,
				})
				return
			}

			// Set response headers indicating negotiated API version
			w.Header().Set("X-API-Version", requestedVersion)
			w.Header().Add("Vary", "Accept, X-API-Version")

			next.ServeHTTP(w, r)
		})
	}
}

// extractRequestedVersion checks X-API-Version header and Accept header.
func extractRequestedVersion(r *http.Request) string {
	// 1. Check explicit X-API-Version header (e.g. "v1" or "1")
	if v := strings.TrimSpace(r.Header.Get("X-API-Version")); v != "" {
		return v
	}

	// 2. Check Accept header for vendor MIME type (e.g. application/vnd.oci.v1+json)
	accept := r.Header.Get("Accept")
	if accept != "" {
		parts := strings.Split(accept, ",")
		for _, part := range parts {
			mime := strings.TrimSpace(strings.Split(part, ";")[0])
			if matches := vendorMediaRegex.FindStringSubmatch(mime); len(matches) == 2 {
				return "v" + matches[1]
			}
		}
	}

	return ""
}

// normalizeVersion converts "1", "V1", "v1" to canonical "v1".
func normalizeVersion(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// Deprecate applies RFC 8594 standard deprecation headers to the response.
// Deprecation: true
// Sunset: <HTTP-date>
// Link: <successorLink>; rel="successor-version"
func Deprecate(w http.ResponseWriter, sunset time.Time, successorLink string) {
	w.Header().Set("Deprecation", "true")
	if !sunset.IsZero() {
		w.Header().Set("Sunset", sunset.UTC().Format(http.TimeFormat))
	}
	if successorLink != "" {
		linkHeader := fmt.Sprintf("<%s>; rel=\"successor-version\"", successorLink)
		w.Header().Set("Link", linkHeader)
	}
}
