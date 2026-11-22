// Package waf implements an edge Web Application Firewall (WAF) HTTP middleware
// driven by Coraza / OWASP Core Rule Set (CRS) SecRule directives tuned for
// OCI's JSON REST APIs, BFF authentication flows, and SSE event streams.
package waf

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/orange-cat-investments/oci/pkg/telemetry"
)

// DefaultMaxBodyBytes bounds request body inspection memory in BFF middleware.
const DefaultMaxBodyBytes = 1 << 20 // 1 MiB

// DefaultOWASPCRSDirectives embeds the Coraza / ModSecurity OWASP CRS ruleset
// tuned for OCI REST JSON payloads and SSE streams.
const DefaultOWASPCRSDirectives = `
SecRuleEngine On

# OWASP CRS 930xxx: Path Traversal & Local File Inclusion (LFI)
SecRule REQUEST_URI|ARGS|REQUEST_BODY "@rx (?i)(?:\.\.[/\\]|%2e%2e(?:%2f|%5c|/)|/(?:etc/(?:passwd|shadow|group)|proc/self/(?:environ|cmdline)))" "id:930100,phase:2,deny,status:403,tag:'attack-lfi',msg:'Path Traversal / Local File Inclusion Attack'"

# OWASP CRS 932xxx: Remote Command Execution (RCE) / Shell Injection
SecRule ARGS|REQUEST_HEADERS|REQUEST_BODY "@rx (?i)(?:[;|&` + "`" + `]\s*(?:cat|ls|id|whoami|uname|nc|netcat|bash|sh|curl|wget)\s+(?:/etc/|/bin/|-[a-z]|https?://)|\$\([^)]*(?:cat|curl|wget|id|whoami|bash|sh)\b)" "id:932100,phase:2,deny,status:403,tag:'attack-rce',msg:'Remote Command Injection Attempt'"

# OWASP CRS 941xxx: Cross-Site Scripting (XSS)
SecRule ARGS|REQUEST_HEADERS|REQUEST_BODY "@rx (?i)<\s*/?\s*script\b[^>]*>" "id:941100,phase:2,deny,status:403,tag:'attack-xss',msg:'XSS Attack Detected via script tag'"
SecRule ARGS|REQUEST_HEADERS|REQUEST_BODY "@rx (?i)(?:<\s*[a-z0-9]+[^>]*\bon(?:error|load|click|mouseover|focus|submit|mouseenter)\s*=|\bon(?:error|load)\s*=\s*(?:alert|prompt|confirm|eval|fetch)\s*\(|javascript\s*:|<\s*iframe\b[^>]*>|<\s*svg\b[^>]*on[a-z]+\s*=)" "id:941110,phase:2,deny,status:403,tag:'attack-xss',msg:'XSS Event Handler or Scripting URI Detected'"

# OWASP CRS 942xxx: SQL Injection (SQLi)
SecRule ARGS|REQUEST_HEADERS|REQUEST_BODY "@rx (?i)(?:'\s*(?:or|and)\s+(?:[0-9]+\s*=\s*[0-9]+|true\b|'[^']*'\s*=\s*'[^']*')|\"\s*(?:or|and)\s+(?:[0-9]+\s*=\s*[0-9]+|true\b)|\bor\s+1\s*=\s*1\s*(?:--|#|\z))" "id:942100,phase:2,deny,status:403,tag:'attack-sqli',msg:'SQL Injection Tautology Attack'"
SecRule ARGS|REQUEST_HEADERS|REQUEST_BODY "@rx (?i)\bunion\s+(?:all\s+)?select\b" "id:942110,phase:2,deny,status:403,tag:'attack-sqli',msg:'SQL Injection UNION SELECT Attack'"
SecRule ARGS|REQUEST_HEADERS|REQUEST_BODY "@rx (?i);\s*(?:drop\s+(?:table|database|schema)|alter\s+table|truncate\s+table|delete\s+from|exec(?:ute)?\s*\(|shutdown\b)" "id:942120,phase:2,deny,status:403,tag:'attack-sqli',msg:'SQL Injection Stacked Query Attack'"
SecRule ARGS|REQUEST_HEADERS|REQUEST_BODY "@rx (?i)(?:\b(?:sleep|benchmark|pg_sleep|waitfor\s+delay)\s*\(|\binformation_schema\b|\bpg_catalog\.pg_)" "id:942130,phase:2,deny,status:403,tag:'attack-sqli',msg:'SQL Injection Time-Based / Introspection Attack'"
`

// Config configures the Coraza OWASP CRS WAF middleware.
type Config struct {
	Directives      string
	MaxBodyBytes    int64
	SSEBypassPaths  []string
	Logger          *slog.Logger
	InspectHeaders  []string
	DetectionOnly   bool
	DisabledRuleIDs []string
}

// DefaultConfig returns production-ready Coraza CRS WAF defaults for OCI BFFs.
func DefaultConfig() Config {
	return Config{
		Directives:   DefaultOWASPCRSDirectives,
		MaxBodyBytes: DefaultMaxBodyBytes,
		SSEBypassPaths: []string{
			"/api/v1/customer/ticker",
			"/api/v1/employee/incidents/stream",
			"/employee/incidents/stream",
		},
		InspectHeaders: []string{
			"User-Agent",
			"Referer",
			"X-Forwarded-For",
			"X-Original-URL",
		},
	}
}

// Violation describes an OWASP CRS rule match.
type Violation struct {
	RuleID   string
	Category string
	Target   string
	Message  string
	Status   int
}

type secRule struct {
	id       string
	category string
	msg      string
	status   int
	targets  []string
	pattern  *regexp.Regexp
	deny     bool
}

// Engine evaluates incoming HTTP requests against compiled Coraza SecRule directives.
type Engine struct {
	enabled       bool
	detectionOnly bool
	maxBodyBytes  int64
	ssePaths      []string
	inspectHdrs   []string
	rules         []secRule
	logger        *slog.Logger
}

var (
	reSecRule = regexp.MustCompile(`^SecRule\s+(\S+)\s+"@rx\s+(.+)"\s+"([^"]+)"\s*$`)
	reTagID   = regexp.MustCompile(`(?:^|,)id:([0-9]+)(?:,|$)`)
	reTagCat  = regexp.MustCompile(`tag:'([^']+)'`)
	reTagMsg  = regexp.MustCompile(`msg:'([^']+)'`)
	reTagStat = regexp.MustCompile(`(?:^|,)status:([0-9]+)(?:,|$)`)
)

// NewEngine compiles Coraza SecRule directives into a reusable WAF Engine.
func NewEngine(cfg Config) (*Engine, error) {
	directives := cfg.Directives
	if strings.TrimSpace(directives) == "" {
		directives = DefaultOWASPCRSDirectives
	}
	maxBody := cfg.MaxBodyBytes
	if maxBody <= 0 {
		maxBody = DefaultMaxBodyBytes
	}
	if len(cfg.InspectHeaders) == 0 {
		cfg.InspectHeaders = DefaultConfig().InspectHeaders
	}
	if len(cfg.SSEBypassPaths) == 0 {
		cfg.SSEBypassPaths = DefaultConfig().SSEBypassPaths
	}

	eng := &Engine{
		enabled:       true,
		detectionOnly: cfg.DetectionOnly,
		maxBodyBytes:  maxBody,
		ssePaths:      slices.Clone(cfg.SSEBypassPaths),
		inspectHdrs:   slices.Clone(cfg.InspectHeaders),
		logger:        cfg.Logger,
	}

	scanner := bufio.NewScanner(strings.NewReader(directives))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "SecRuleEngine ") {
			mode := strings.TrimSpace(strings.TrimPrefix(line, "SecRuleEngine "))
			switch strings.ToLower(mode) {
			case "off":
				eng.enabled = false
			case "detectiononly":
				eng.detectionOnly = true
			case "on":
				eng.enabled = true
			}
			continue
		}
		if !strings.HasPrefix(line, "SecRule ") {
			continue
		}
		m := reSecRule.FindStringSubmatch(line)
		if m == nil {
			return nil, errors.New("waf: invalid SecRule syntax: " + line)
		}
		targets := strings.Split(strings.ToUpper(m[1]), "|")
		rx, err := regexp.Compile(m[2])
		if err != nil {
			return nil, err
		}
		actions := m[3]
		id := ""
		if idm := reTagID.FindStringSubmatch(actions); idm != nil {
			id = idm[1]
		}
		if slices.Contains(cfg.DisabledRuleIDs, id) {
			continue
		}
		cat := "owasp-crs"
		if cm := reTagCat.FindStringSubmatch(actions); cm != nil {
			cat = strings.TrimPrefix(cm[1], "attack-")
		}
		msg := "OWASP CRS rule violation"
		if mm := reTagMsg.FindStringSubmatch(actions); mm != nil {
			msg = mm[1]
		}
		status := http.StatusForbidden
		if sm := reTagStat.FindStringSubmatch(actions); sm != nil {
			if parsed, convErr := strconv.Atoi(sm[1]); convErr == nil && parsed >= 400 && parsed <= 599 {
				status = parsed
			}
		}

		eng.rules = append(eng.rules, secRule{
			id:       id,
			category: cat,
			msg:      msg,
			status:   status,
			targets:  targets,
			pattern:  rx,
			deny:     strings.Contains(actions, "deny"),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return eng, nil
}

// Middleware returns a standard Chi HTTP middleware enforcing the configured Coraza CRS rules.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	eng, err := NewEngine(cfg)
	if err != nil {
		// Fallback to verified embedded directives if custom directives fail to parse.
		eng, _ = NewEngine(DefaultConfig())
	}
	return eng.Handler
}

// Handler wraps an http.Handler with WAF inspection on URI, query parameters, headers, and bodies.
func (e *Engine) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !e.enabled {
			next.ServeHTTP(w, r)
			return
		}

		violation, err := e.InspectRequest(r)
		if err != nil {
			http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
			return
		}
		if violation != nil {
			telemetry.RecordWAFBlockedRequest(r.Context(), violation.RuleID, violation.Category, violation.Target)
			if e.logger != nil {
				// CWE-532: Log metadata only; never log raw exploit strings or sensitive request bodies.
				e.logger.Warn("WAF blocked malicious request",
					"rule_id", violation.RuleID,
					"category", violation.Category,
					"target", violation.Target,
					"method", r.Method,
					"path", r.URL.Path,
				)
			}
			if !e.detectionOnly {
				if violation.RuleID != "" {
					w.Header().Set("X-OCI-WAF-Rule", violation.RuleID)
				}
				w.Header().Set("Content-Type", "application/json")
				status := violation.Status
				if status == 0 {
					status = http.StatusForbidden
				}
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":    "request blocked by web application firewall",
					"code":     "WAF_RULE_VIOLATION",
					"rule_id":  violation.RuleID,
					"category": violation.Category,
				})
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// InspectRequest evaluates an *http.Request against active rules and restores r.Body for downstream handlers.
func (e *Engine) InspectRequest(r *http.Request) (*Violation, error) {
	if r == nil || r.URL == nil {
		return nil, nil
	}

	uriCandidates := collectURICandidates(r)
	argCandidates := collectQueryCandidates(r)
	hdrCandidates := e.collectHeaderCandidates(r)

	var bodyCandidates []string
	if !e.isSSEStream(r) && r.Body != nil && r.ContentLength != 0 {
		limited := io.LimitReader(r.Body, e.maxBodyBytes+1)
		bodyBytes, err := io.ReadAll(limited)
		_ = r.Body.Close()
		if err != nil {
			return nil, err
		}
		if int64(len(bodyBytes)) > e.maxBodyBytes {
			return &Violation{
				RuleID:   "920420",
				Category: "protocol",
				Target:   "body",
				Message:  "Request body exceeds WAF inspection limit",
				Status:   http.StatusRequestEntityTooLarge,
			}, nil
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		bodyCandidates = extractBodyCandidates(bodyBytes, r.Header.Get("Content-Type"))
	}

	for _, rule := range e.rules {
		if !rule.deny {
			continue
		}
		for _, target := range rule.targets {
			switch target {
			case "REQUEST_URI":
				if matchAny(rule.pattern, uriCandidates) {
					return &Violation{RuleID: rule.id, Category: rule.category, Target: "uri", Message: rule.msg, Status: rule.status}, nil
				}
			case "ARGS", "QUERY_STRING":
				if matchAny(rule.pattern, argCandidates) {
					return &Violation{RuleID: rule.id, Category: rule.category, Target: "query", Message: rule.msg, Status: rule.status}, nil
				}
			case "REQUEST_HEADERS":
				if matchAny(rule.pattern, hdrCandidates) {
					return &Violation{RuleID: rule.id, Category: rule.category, Target: "header", Message: rule.msg, Status: rule.status}, nil
				}
			case "REQUEST_BODY":
				if matchAny(rule.pattern, bodyCandidates) {
					return &Violation{RuleID: rule.id, Category: rule.category, Target: "body", Message: rule.msg, Status: rule.status}, nil
				}
			}
		}
	}

	return nil, nil
}

func (e *Engine) isSSEStream(r *http.Request) bool {
	if strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/event-stream") {
		return true
	}
	return slices.Contains(e.ssePaths, r.URL.Path)
}

func collectURICandidates(r *http.Request) []string {
	out := []string{r.URL.Path}
	if r.URL.RawPath != "" && r.URL.RawPath != r.URL.Path {
		out = append(out, r.URL.RawPath)
	}
	if r.RequestURI != "" {
		out = append(out, r.RequestURI)
	}
	if dec, err := url.PathUnescape(r.URL.Path); err == nil && dec != r.URL.Path {
		out = append(out, dec)
	}
	return out
}

func collectQueryCandidates(r *http.Request) []string {
	var out []string
	if r.URL.RawQuery != "" {
		out = append(out, r.URL.RawQuery)
		if dec, err := url.QueryUnescape(r.URL.RawQuery); err == nil && dec != r.URL.RawQuery {
			out = append(out, dec)
		}
	}
	for k, vals := range r.URL.Query() {
		out = append(out, k)
		out = append(out, vals...)
	}
	return out
}

func (e *Engine) collectHeaderCandidates(r *http.Request) []string {
	var out []string
	for _, h := range e.inspectHdrs {
		for _, v := range r.Header.Values(h) {
			if v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

func extractBodyCandidates(bodyBytes []byte, contentType string) []string {
	trimmed := bytes.TrimSpace(bodyBytes)
	if len(trimmed) == 0 {
		return nil
	}
	lowerCT := strings.ToLower(contentType)
	if strings.Contains(lowerCT, "application/json") || (len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')) {
		var parsed any
		if err := json.Unmarshal(trimmed, &parsed); err == nil {
			var vals []string
			collectJSONStrings(parsed, &vals)
			return vals
		}
	}
	if strings.Contains(lowerCT, "application/x-www-form-urlencoded") {
		if formVals, err := url.ParseQuery(string(trimmed)); err == nil {
			var out []string
			for k, vs := range formVals {
				out = append(out, k)
				out = append(out, vs...)
			}
			return out
		}
	}
	return []string{string(trimmed)}
}

func collectJSONStrings(v any, out *[]string) {
	switch x := v.(type) {
	case string:
		if x != "" {
			*out = append(*out, x)
		}
	case []any:
		for _, item := range x {
			collectJSONStrings(item, out)
		}
	case map[string]any:
		for k, val := range x {
			*out = append(*out, k)
			collectJSONStrings(val, out)
		}
	}
}

func matchAny(rx *regexp.Regexp, candidates []string) bool {
	for _, c := range candidates {
		if rx.MatchString(c) {
			return true
		}
	}
	return false
}
