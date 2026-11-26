package s3

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRegion          = "garage"
	defaultService         = "s3"
	defaultRequestTimeout  = 15 * time.Second
	defaultPresignTTL      = 15 * time.Minute
	maxDownloadBytes       = 64 * 1024 * 1024
	unsignedPayloadHash    = "UNSIGNED-PAYLOAD"
	sigV4Algorithm         = "AWS4-HMAC-SHA256"
	sigV4RequestTerminator = "aws4_request"
)

// BlobMetadata describes an uploaded object stored in S3-compatible storage.
type BlobMetadata struct {
	Bucket      string
	Key         string
	ContentType string
	SHA256Hex   string
	SizeBytes   int64
	ETag        string
}

// Store defines a vendor-neutral object storage contract for statements, tax forms,
// firmware images, and backup artifacts.
type Store interface {
	EnsureBuckets(ctx context.Context, buckets ...string) error
	PutBlob(ctx context.Context, bucket, key, contentType string, data []byte) (*BlobMetadata, error)
	GetBlob(ctx context.Context, bucket, key string) ([]byte, error)
	PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
}

// Config configures an S3-compatible Garage blobStore client.
type Config struct {
	Endpoint        string
	PublicEndpoint  string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	HTTPClient      *http.Client
	Now             func() time.Time
}

// Client implements Store using path-style S3 HTTP requests signed with AWS SigV4.
type Client struct {
	endpoint        *url.URL
	publicEndpoint  *url.URL
	region          string
	accessKeyID     string
	secretAccessKey string
	httpClient      *http.Client
	now             func() time.Time
}

// NewClient constructs a validated S3-compatible object storage client.
func NewClient(cfg Config) (*Client, error) {
	epRaw := strings.TrimSpace(cfg.Endpoint)
	if epRaw == "" {
		return nil, errors.New("blobstore endpoint is required")
	}
	ep, err := parseEndpointURL(epRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid blobstore endpoint: %w", err)
	}

	pubEp := ep
	if pubRaw := strings.TrimSpace(cfg.PublicEndpoint); pubRaw != "" {
		parsedPub, pubErr := parseEndpointURL(pubRaw)
		if pubErr != nil {
			return nil, fmt.Errorf("invalid blobstore public endpoint: %w", pubErr)
		}
		pubEp = parsedPub
	}

	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = defaultRegion
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultRequestTimeout}
	}

	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}

	return &Client{
		endpoint:        ep,
		publicEndpoint:  pubEp,
		region:          region,
		accessKeyID:     strings.TrimSpace(cfg.AccessKeyID),
		secretAccessKey: strings.TrimSpace(cfg.SecretAccessKey),
		httpClient:      httpClient,
		now:             nowFn,
	}, nil
}

// NewFromEnv constructs a Client from standard environment variables, returning nil
// when BLOBSTORE_ENDPOINT / S3_ENDPOINT is unset.
func NewFromEnv() *Client {
	ep := firstNonEmpty(os.Getenv("BLOBSTORE_ENDPOINT"), os.Getenv("S3_ENDPOINT"))
	if ep == "" {
		return nil
	}
	pubEp := firstNonEmpty(os.Getenv("BLOBSTORE_PUBLIC_ENDPOINT"), os.Getenv("S3_PUBLIC_ENDPOINT"))
	region := firstNonEmpty(os.Getenv("BLOBSTORE_REGION"), os.Getenv("AWS_REGION"), defaultRegion)
	accessKey := firstNonEmpty(os.Getenv("BLOBSTORE_ACCESS_KEY"), os.Getenv("AWS_ACCESS_KEY_ID"))
	secretKey := firstNonEmpty(os.Getenv("BLOBSTORE_SECRET_KEY"), os.Getenv("AWS_SECRET_ACCESS_KEY"))

	client, err := NewClient(Config{
		Endpoint:        ep,
		PublicEndpoint:  pubEp,
		Region:          region,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	})
	if err != nil {
		return nil
	}
	return client
}

// EnsureBuckets verifies that each named bucket exists, creating missing ones idempotently.
func (c *Client) EnsureBuckets(ctx context.Context, buckets ...string) error {
	for _, b := range buckets {
		bucket := strings.TrimSpace(b)
		if bucket == "" {
			continue
		}
		target := c.objectURL(c.endpoint, bucket, "")
		req, err := c.newSignedRequest(ctx, http.MethodHead, target, "", nil)
		if err != nil {
			return err
		}
		resp, err := c.httpClient.Do(req) // #nosec G107 -- target URL derived from validated Config.Endpoint
		if err != nil {
			return fmt.Errorf("head bucket %s: %w", bucket, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
			continue
		}
		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("head bucket %s returned HTTP %d", bucket, resp.StatusCode)
		}

		createReq, err := c.newSignedRequest(ctx, http.MethodPut, target, "", nil)
		if err != nil {
			return err
		}
		createResp, err := c.httpClient.Do(createReq) // #nosec G107 -- target URL derived from validated Config.Endpoint
		if err != nil {
			return fmt.Errorf("create bucket %s: %w", bucket, err)
		}
		body, _ := io.ReadAll(io.LimitReader(createResp.Body, 4096))
		_ = createResp.Body.Close()
		if createResp.StatusCode != http.StatusOK && createResp.StatusCode != http.StatusCreated && createResp.StatusCode != http.StatusConflict {
			return fmt.Errorf("create bucket %s returned HTTP %d: %s", bucket, createResp.StatusCode, strings.TrimSpace(string(body)))
		}
	}
	return nil
}

// PutBlob uploads data to bucket/key with SHA-256 integrity metadata.
func (c *Client) PutBlob(ctx context.Context, bucket, key, contentType string, data []byte) (*BlobMetadata, error) {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimLeft(strings.TrimSpace(key), "/")
	if bucket == "" || key == "" {
		return nil, errors.New("bucket and key are required")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	sum := sha256.Sum256(data)
	sumHex := hex.EncodeToString(sum[:])

	target := c.objectURL(c.endpoint, bucket, key)
	req, err := c.newSignedRequest(ctx, http.MethodPut, target, contentType, data, map[string]string{
		"x-amz-meta-sha256": sumHex,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req) // #nosec G107 -- target URL derived from validated Config.Endpoint
	if err != nil {
		return nil, fmt.Errorf("put blob %s/%s: %w", bucket, key, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("put blob %s/%s returned HTTP %d: %s", bucket, key, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return &BlobMetadata{
		Bucket:      bucket,
		Key:         key,
		ContentType: contentType,
		SHA256Hex:   sumHex,
		SizeBytes:   int64(len(data)),
		ETag:        strings.Trim(resp.Header.Get("ETag"), `"`),
	}, nil
}

// GetBlob retrieves an object from bucket/key and verifies x-amz-meta-sha256 when present.
func (c *Client) GetBlob(ctx context.Context, bucket, key string) ([]byte, error) {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimLeft(strings.TrimSpace(key), "/")
	if bucket == "" || key == "" {
		return nil, errors.New("bucket and key are required")
	}

	target := c.objectURL(c.endpoint, bucket, key)
	req, err := c.newSignedRequest(ctx, http.MethodGet, target, "", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req) // #nosec G107 -- target URL derived from validated Config.Endpoint
	if err != nil {
		return nil, fmt.Errorf("get blob %s/%s: %w", bucket, key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("get blob %s/%s returned HTTP %d: %s", bucket, key, resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read blob %s/%s: %w", bucket, key, err)
	}
	if len(body) > maxDownloadBytes {
		return nil, fmt.Errorf("blob %s/%s exceeds %d byte limit", bucket, key, maxDownloadBytes)
	}

	if expectedSHA := strings.TrimSpace(resp.Header.Get("x-amz-meta-sha256")); expectedSHA != "" {
		sum := sha256.Sum256(body)
		actualSHA := hex.EncodeToString(sum[:])
		if !strings.EqualFold(expectedSHA, actualSHA) {
			return nil, fmt.Errorf("blob %s/%s SHA-256 mismatch: expected %s, got %s", bucket, key, expectedSHA, actualSHA)
		}
	}

	return body, nil
}

// PresignGetURL builds a time-limited AWS SigV4 pre-signed GET URL for bucket/key.
func (c *Client) PresignGetURL(_ context.Context, bucket, key string, ttl time.Duration) (string, error) {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimLeft(strings.TrimSpace(key), "/")
	if bucket == "" || key == "" {
		return "", errors.New("bucket and key are required")
	}
	if ttl <= 0 {
		ttl = defaultPresignTTL
	}
	expiresSec := int64(ttl.Seconds())
	if expiresSec < 1 {
		expiresSec = 1
	}
	if expiresSec > 7*24*3600 {
		expiresSec = 7 * 24 * 3600
	}

	now := c.now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	scope := fmt.Sprintf("%s/%s/%s/%s", dateStamp, c.region, defaultService, sigV4RequestTerminator)

	target := c.objectURL(c.publicEndpoint, bucket, key)
	query := target.Query()
	query.Set("X-Amz-Algorithm", sigV4Algorithm)
	query.Set("X-Amz-Credential", c.accessKeyID+"/"+scope)
	query.Set("X-Amz-Date", amzDate)
	query.Set("X-Amz-Expires", strconv.FormatInt(expiresSec, 10))
	query.Set("X-Amz-SignedHeaders", "host")

	canonicalQuery := canonicalQueryString(query)
	canonicalHeaders := fmt.Sprintf("host:%s\n", target.Host)
	canonicalReq := strings.Join([]string{
		http.MethodGet,
		canonicalURI(target),
		canonicalQuery,
		canonicalHeaders,
		"host",
		unsignedPayloadHash,
	}, "\n")

	stringToSign := strings.Join([]string{
		sigV4Algorithm,
		amzDate,
		scope,
		sha256Hex([]byte(canonicalReq)),
	}, "\n")

	sigKey := deriveSigningKey(c.secretAccessKey, dateStamp, c.region, defaultService)
	signature := hex.EncodeToString(hmacSHA256(sigKey, stringToSign))

	target.RawQuery = canonicalQuery + "&X-Amz-Signature=" + signature
	return target.String(), nil
}

func (c *Client) newSignedRequest(ctx context.Context, method string, target *url.URL, contentType string, body []byte, extraHeaders ...map[string]string) (*http.Request, error) {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), bodyReader) // #nosec G107 -- target URL built from validated Config.Endpoint
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if len(body) > 0 {
		req.ContentLength = int64(len(body))
	}

	now := c.now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := sha256Hex(body)

	req.Header.Set("Host", target.Host)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	for _, hdrMap := range extraHeaders {
		for k, v := range hdrMap {
			req.Header.Set(k, v)
		}
	}

	if c.accessKeyID == "" && c.secretAccessKey == "" {
		return req, nil
	}

	signedHeaderMap := map[string]string{
		"host":                 target.Host,
		"x-amz-content-sha256": payloadHash,
		"x-amz-date":           amzDate,
	}
	if contentType != "" {
		signedHeaderMap["content-type"] = contentType
	}
	for _, hdrMap := range extraHeaders {
		for k, v := range hdrMap {
			signedHeaderMap[strings.ToLower(k)] = strings.TrimSpace(v)
		}
	}

	headerKeys := make([]string, 0, len(signedHeaderMap))
	for k := range signedHeaderMap {
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)

	var canonHeaders strings.Builder
	for _, k := range headerKeys {
		canonHeaders.WriteString(k)
		canonHeaders.WriteByte(':')
		canonHeaders.WriteString(signedHeaderMap[k])
		canonHeaders.WriteByte('\n')
	}
	signedHeadersList := strings.Join(headerKeys, ";")

	canonicalReq := strings.Join([]string{
		method,
		canonicalURI(target),
		canonicalQueryString(target.Query()),
		canonHeaders.String(),
		signedHeadersList,
		payloadHash,
	}, "\n")

	scope := fmt.Sprintf("%s/%s/%s/%s", dateStamp, c.region, defaultService, sigV4RequestTerminator)
	stringToSign := strings.Join([]string{
		sigV4Algorithm,
		amzDate,
		scope,
		sha256Hex([]byte(canonicalReq)),
	}, "\n")

	sigKey := deriveSigningKey(c.secretAccessKey, dateStamp, c.region, defaultService)
	signature := hex.EncodeToString(hmacSHA256(sigKey, stringToSign))
	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		sigV4Algorithm, c.accessKeyID, scope, signedHeadersList, signature)
	req.Header.Set("Authorization", authHeader)
	return req, nil
}

func (c *Client) objectURL(base *url.URL, bucket, key string) *url.URL {
	u := *base
	basePath := strings.TrimRight(base.Path, "/")
	if key == "" {
		u.Path = basePath + "/" + bucket
	} else {
		cleanKey := strings.TrimLeft(path.Clean("/"+key), "/")
		u.Path = basePath + "/" + bucket + "/" + cleanKey
	}
	return &u
}

func parseEndpointURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("endpoint %q must use http or https with a non-empty host", raw)
	}
	return u, nil
}

func canonicalURI(u *url.URL) string {
	esc := u.EscapedPath()
	if esc == "" {
		return "/"
	}
	return esc
}

func canonicalQueryString(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		vals := append([]string(nil), q[k]...)
		sort.Strings(vals)
		ek := awsPercentEncode(k)
		for _, v := range vals {
			parts = append(parts, ek+"="+awsPercentEncode(v))
		}
	}
	return strings.Join(parts, "&")
}

func awsPercentEncode(s string) string {
	esc := url.QueryEscape(s)
	esc = strings.ReplaceAll(esc, "+", "%20")
	esc = strings.ReplaceAll(esc, "*", "%2A")
	esc = strings.ReplaceAll(esc, "%7E", "~")
	return esc
}

func deriveSigningKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, sigV4RequestTerminator)
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(data))
	return mac.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
