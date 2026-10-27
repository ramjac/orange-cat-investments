package versioning

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
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

type versionCandidate struct {
	version string
	q       float64
	order   int
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
			requestedVersion, supported := negotiateVersion(r, cfg, supportedMap)

			// Validate if negotiated version is supported
			if !supported {
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

// negotiateVersion resolves the requested API version considering explicit headers,
// Accept vendor MIME types, quality (q) parameters, and supported version constraints.
func negotiateVersion(r *http.Request, cfg Config, supportedMap map[string]bool) (string, bool) {
	// 1. Check explicit X-API-Version header (e.g. "v1" or "1")
	if v := strings.TrimSpace(r.Header.Get("X-API-Version")); v != "" {
		norm := normalizeVersion(v)
		return norm, supportedMap[norm]
	}

	// 2. Check Accept header for vendor MIME type (e.g. application/vnd.oci.v1+json)
	if accept := r.Header.Get("Accept"); accept != "" {
		candidates := parseAcceptCandidates(accept)
		if len(candidates) > 0 {
			return selectCandidate(candidates, supportedMap)
		}
	}

	// 3. Fallback to default configured version
	def := normalizeVersion(cfg.DefaultVersion)
	return def, supportedMap[def]
}

// parseQualityFactor extracts the q-value parameter from MIME parameters (e.g. "q=0.8").
func parseQualityFactor(params []string) float64 {
	q := 1.0
	for _, param := range params {
		param = strings.TrimSpace(param)
		if strings.HasPrefix(strings.ToLower(param), "q=") {
			if parsedQ, err := strconv.ParseFloat(param[2:], 64); err == nil {
				q = parsedQ
			}
		}
	}
	return q
}

// parseAcceptCandidates parses candidate version options from an Accept header string.
func parseAcceptCandidates(acceptHeader string) []versionCandidate {
	parts := strings.Split(acceptHeader, ",")
	var candidates []versionCandidate
	for i, part := range parts {
		elements := strings.Split(part, ";")
		mime := strings.TrimSpace(elements[0])
		matches := vendorMediaRegex.FindStringSubmatch(mime)
		if len(matches) != 2 {
			continue
		}
		q := parseQualityFactor(elements[1:])
		candidates = append(candidates, versionCandidate{
			version: normalizeVersion("v" + matches[1]),
			q:       q,
			order:   i,
		})
	}
	return candidates
}

// selectCandidate selects the best supported version candidate from sorted candidates.
func selectCandidate(candidates []versionCandidate, supportedMap map[string]bool) (string, bool) {
	// Sort candidates by quality descending, preserving order for matching q
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].q != candidates[j].q {
			return candidates[i].q > candidates[j].q
		}
		return candidates[i].order < candidates[j].order
	})

	// Negotiate: check acceptable candidates (q > 0) against supported versions
	for _, c := range candidates {
		if c.q > 0 && supportedMap[c.version] {
			return c.version, true
		}
	}

	// If no supported candidate with q > 0 was found, return the top candidate as rejected
	firstVersion := ""
	if len(candidates) > 0 {
		firstVersion = candidates[0].version
	}
	return firstVersion, false
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
