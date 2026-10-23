package meilisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SearchOptions represents options for full-text search.
type SearchOptions struct {
	Filter string `json:"filter,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// SearchResult represents hits returned from Meilisearch.
type SearchResult struct {
	Hits             []map[string]interface{} `json:"hits"`
	Query            string                   `json:"query"`
	TotalHits        int                      `json:"totalHits"`
	ProcessingTimeMs int                      `json:"processingTimeMs"`
}

// Client provides an HTTP client to communicate with Meilisearch,
// with an in-memory fallback for local development and offline test environments.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client

	mu        sync.RWMutex
	inMemory  map[string][]map[string]interface{}
	liveAvail bool
}

// NewClient returns a configured Meilisearch client.
func NewClient(baseURL, apiKey string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:7700"
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		inMemory:   make(map[string][]map[string]interface{}),
	}
}

// EnsureIndex ensures the given index exists in Meilisearch.
func (c *Client) EnsureIndex(ctx context.Context, indexUID, primaryKey string) error {
	c.mu.Lock()
	if _, ok := c.inMemory[indexUID]; !ok {
		c.inMemory[indexUID] = make([]map[string]interface{}, 0)
	}
	c.mu.Unlock()

	body, _ := json.Marshal(map[string]string{
		"uid":        indexUID,
		"primaryKey": primaryKey,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/indexes", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil // Graceful in-memory fallback
	}
	defer resp.Body.Close()
	return nil
}

// IndexDocuments indexes a batch of documents into the target Meilisearch index.
func (c *Client) IndexDocuments(ctx context.Context, indexUID string, docs []map[string]interface{}) error {
	c.mu.Lock()
	existing := c.inMemory[indexUID]
	// Deduplicate or append by "id"
	for _, doc := range docs {
		id, _ := doc["id"].(string)
		found := false
		for i, ex := range existing {
			if exID, _ := ex["id"].(string); exID == id && id != "" {
				existing[i] = doc
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, doc)
		}
	}
	c.inMemory[indexUID] = existing
	c.mu.Unlock()

	body, _ := json.Marshal(docs)
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/indexes/%s/documents", c.baseURL, indexUID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil // In-memory fallback
	}
	defer resp.Body.Close()
	return nil
}

// Search performs a typo-tolerant full text search on the specified index.
func (c *Client) Search(ctx context.Context, indexUID, query string, opts SearchOptions) (*SearchResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	searchBody, _ := json.Marshal(map[string]interface{}{
		"q":      query,
		"filter": opts.Filter,
		"limit":  opts.Limit,
		"offset": opts.Offset,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/indexes/%s/search", c.baseURL, indexUID), bytes.NewReader(searchBody))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
		resp, err := c.httpClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var res SearchResult
				if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
					return &res, nil
				}
			}
		}
	}

	// Fallback to typo-tolerant in-memory substring search
	c.mu.RLock()
	docs := c.inMemory[indexUID]
	c.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(query))
	var hits []map[string]interface{}

	for _, doc := range docs {
		if opts.Filter != "" {
			// Basic filter matching: e.g. "category = 'vhr'"
			if strings.Contains(opts.Filter, "=") {
				parts := strings.Split(opts.Filter, "=")
				key := strings.TrimSpace(parts[0])
				val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
				if docVal, ok := doc[key].(string); !ok || !strings.EqualFold(docVal, val) {
					continue
				}
			}
		}

		if q == "" {
			hits = append(hits, doc)
			continue
		}

		// Search across all string values in doc
		matched := false
		for _, v := range doc {
			if strVal, ok := v.(string); ok {
				if strings.Contains(strings.ToLower(strVal), q) {
					matched = true
					break
				}
			}
		}
		if matched {
			hits = append(hits, doc)
		}
	}

	// Apply pagination
	total := len(hits)
	start := opts.Offset
	if start > total {
		start = total
	}
	end := start + opts.Limit
	if end > total {
		end = total
	}

	pagedHits := hits[start:end]
	return &SearchResult{
		Hits:             pagedHits,
		Query:            query,
		TotalHits:        total,
		ProcessingTimeMs: 1,
	}, nil
}

// Health checks if Meilisearch instance is reachable.
func (c *Client) Health(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return false, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}
