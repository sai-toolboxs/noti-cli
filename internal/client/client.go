package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	nerr "noti-cli/internal/errors"
)

// NotionClient handles HTTP communication with the Notion API.
type NotionClient struct {
	httpClient *http.Client
	baseURL    string
	apiVersion string
	maxRetries int
	retryDelay time.Duration
}

// Option configures the NotionClient.
type Option func(*NotionClient)

// WithMaxRetries sets the maximum number of retries for idempotent requests.
func WithMaxRetries(n int) Option {
	return func(c *NotionClient) { c.maxRetries = n }
}

// WithRetryDelay sets the base delay between retries.
func WithRetryDelay(d time.Duration) Option {
	return func(c *NotionClient) { c.retryDelay = d }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *NotionClient) { c.httpClient = hc }
}

// New creates a new NotionClient.
func New(httpClient *http.Client, baseURL, apiVersion string, opts ...Option) *NotionClient {
	c := &NotionClient{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiVersion: apiVersion,
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// APIError represents an error response from the Notion API.
type APIError struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("notion API %d: %s — %s", e.Status, e.Code, e.Message)
}

// NotionBlock represents a block object from the Notion API.
type NotionBlock struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Object      string          `json:"object"`
	HasChildren bool            `json:"has_children"`
	Parent      json.RawMessage `json:"parent,omitempty"`
	RichText    json.RawMessage `json:"-"`
	Raw         json.RawMessage `json:"-"`
}

// UnmarshalJSON implements custom unmarshaling to capture the type-specific field.
func (b *NotionBlock) UnmarshalJSON(data []byte) error {
	type Alias NotionBlock
	a := &struct {
		*Alias
	}{Alias: (*Alias)(b)}
	if err := json.Unmarshal(data, a); err != nil {
		return err
	}
	b.Raw = data

	// Extract rich_text from the type-specific field.
	var typeData map[string]json.RawMessage
	if err := json.Unmarshal(data, &typeData); err != nil {
		return nil
	}
	if rt, ok := typeData[b.Type]; ok {
		var content struct {
			RichText json.RawMessage `json:"rich_text"`
		}
		if err := json.Unmarshal(rt, &content); err == nil {
			b.RichText = content.RichText
		}
	}
	return nil
}

// BlockListResponse represents a paginated list of blocks.
type BlockListResponse struct {
	Object     string        `json:"object"`
	Results    []NotionBlock `json:"results"`
	HasMore    bool          `json:"has_more"`
	NextCursor string        `json:"next_cursor"`
}

// DoRequest performs an HTTP request and returns the response body.
// It handles retries for idempotent methods on transient errors.
func (c *NotionClient) DoRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	return c.doRequestWithRetry(ctx, method, path, body, true)
}

// DoRequestNoRetry performs an HTTP request without retries.
func (c *NotionClient) DoRequestNoRetry(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	return c.doRequestWithRetry(ctx, method, path, body, false)
}

func (c *NotionClient) doRequestWithRetry(ctx context.Context, method, path string, body interface{}, allowRetry bool) ([]byte, int, error) {
	var lastErr error
	maxAttempts := 1
	if allowRetry && (method == http.MethodGet || method == http.MethodDelete) {
		maxAttempts = c.maxRetries + 1
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := c.retryDelay * time.Duration(1<<uint(attempt-1))
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			select {
			case <-ctx.Done():
				return nil, 0, nerr.Wrap(nerr.CodeTimeout, "context cancelled during retry backoff", ctx.Err())
			case <-time.After(delay):
			}
		}

		data, status, err := c.doRequest(ctx, method, path, body)
		if err != nil {
			lastErr = err
			continue
		}

		// Retry on 429 and 529.
		if status == 429 || status == 529 {
			lastErr = &APIError{Status: status, Code: "rate_limited", Message: "rate limited by Notion API"}
			continue
		}

		// Retry on 5xx for idempotent methods.
		if status >= 500 && (method == http.MethodGet || method == http.MethodDelete) {
			lastErr = &APIError{Status: status, Code: "server_error", Message: fmt.Sprintf("server error %d", status)}
			continue
		}

		if status >= 400 {
			var apiErr APIError
			if jsonErr := json.Unmarshal(data, &apiErr); jsonErr == nil && apiErr.Code != "" {
				return data, status, &apiErr
			}
			return data, status, &APIError{
				Status:  status,
				Code:    "unknown_error",
				Message: fmt.Sprintf("HTTP %d: %s", status, string(data)),
			}
		}

		return data, status, nil
	}

	return nil, 0, lastErr
}

func (c *NotionClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, nerr.Wrap(nerr.CodeInternal, "failed to marshal request body", err)
		}
		bodyReader = strings.NewReader(string(bodyBytes))
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, 0, nerr.Wrap(nerr.CodeInternal, "failed to create request", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, 0, nerr.Wrap(nerr.CodeTimeout, "request timed out", ctx.Err())
		}
		return nil, 0, nerr.Wrap(nerr.CodeNetwork, "request failed", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, nerr.Wrap(nerr.CodeNetwork, "failed to read response body", err)
	}

	return data, resp.StatusCode, nil
}

// GetBlock retrieves a single block by ID.
func (c *NotionClient) GetBlock(ctx context.Context, blockID string) (*NotionBlock, int, error) {
	data, status, err := c.DoRequest(ctx, http.MethodGet, "/v1/blocks/"+url.PathEscape(blockID), nil)
	if err != nil {
		return nil, status, err
	}
	var block NotionBlock
	if err := json.Unmarshal(data, &block); err != nil {
		return nil, status, nerr.Wrap(nerr.CodeInternal, "failed to decode block response", err)
	}
	return &block, status, nil
}

// ListBlockChildren retrieves child blocks with pagination.
func (c *NotionClient) ListBlockChildren(ctx context.Context, blockID string, startCursor string) (*BlockListResponse, int, error) {
	path := "/v1/blocks/" + url.PathEscape(blockID) + "/children"
	if startCursor != "" {
		path += "?start_cursor=" + url.QueryEscape(startCursor)
	}
	data, status, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, status, err
	}
	var resp BlockListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, status, nerr.Wrap(nerr.CodeInternal, "failed to decode block list response", err)
	}
	return &resp, status, nil
}

// ListAllBlockChildren retrieves all child blocks, handling pagination.
func (c *NotionClient) ListAllBlockChildren(ctx context.Context, blockID string) ([]NotionBlock, error) {
	var all []NotionBlock
	var cursor string
	for {
		resp, _, err := c.ListBlockChildren(ctx, blockID, cursor)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Results...)
		if !resp.HasMore {
			break
		}
		cursor = resp.NextCursor
		if cursor == "" {
			break
		}
	}
	return all, nil
}

// ListBlockChildrenCompact retrieves child blocks returning only IDs and types.
func (c *NotionClient) ListBlockChildrenCompact(ctx context.Context, blockID string, startCursor string) (blockIDs []string, blockTypes []string, hasMore bool, nextCursor string, err error) {
	resp, _, err := c.ListBlockChildren(ctx, blockID, startCursor)
	if err != nil {
		return nil, nil, false, "", err
	}
	ids := make([]string, len(resp.Results))
	types := make([]string, len(resp.Results))
	for i, b := range resp.Results {
		ids[i] = b.ID
		types[i] = b.Type
	}
	return ids, types, resp.HasMore, resp.NextCursor, nil
}

// ParseCursor parses a cursor string from API responses.
func ParseCursor(s string) string {
	if s == "" {
		return ""
	}
	// Validate it looks like a UUID.
	if len(s) != 36 {
		return ""
	}
	return s
}

// NotionPage represents a page object from the Notion API.
type NotionPage struct {
	ID             string                 `json:"id"`
	Object         string                 `json:"object"`
	CreatedTime    string                 `json:"created_time"`
	LastEditedTime string                 `json:"last_edited_time"`
	Archived       bool                   `json:"archived"`
	Properties     map[string]interface{} `json:"properties"`
	Title          string                 `json:"-"`
	Raw            json.RawMessage        `json:"-"`
}

// UnmarshalJSON extracts the page title from properties.
func (p *NotionPage) UnmarshalJSON(data []byte) error {
	type Alias NotionPage
	a := &struct {
		*Alias
	}{Alias: (*Alias)(p)}
	if err := json.Unmarshal(data, a); err != nil {
		return err
	}
	p.Raw = data

	// Extract title from the "title" property or first title-type property.
	for _, prop := range p.Properties {
		if propMap, ok := prop.(map[string]interface{}); ok {
			if propMap["type"] == "title" {
				if titleArr, ok := propMap["title"].([]interface{}); ok && len(titleArr) > 0 {
					if first, ok := titleArr[0].(map[string]interface{}); ok {
						if plainText, ok := first["plain_text"].(string); ok {
							p.Title = plainText
						}
					}
				}
				break
			}
		}
	}
	return nil
}

// GetPage retrieves a single page by ID.
func (c *NotionClient) GetPage(ctx context.Context, pageID string) (*NotionPage, int, error) {
	data, status, err := c.DoRequest(ctx, http.MethodGet, "/v1/pages/"+url.PathEscape(pageID), nil)
	if err != nil {
		return nil, status, err
	}
	var page NotionPage
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, status, nerr.Wrap(nerr.CodeInternal, "failed to decode page response", err)
	}
	return &page, status, nil
}

// FormatBlockSize returns a compact string summary of a block.
func FormatBlockSize(b *NotionBlock) string {
	if b == nil {
		return "<nil>"
	}
	id := b.ID
	if len(id) > 8 {
		id = id[:8]
	}
	return fmt.Sprintf("[%s] %s", id, b.Type)
}

// IntToString converts an int to a string for pagination.
func IntToString(n int) string {
	return strconv.Itoa(n)
}

// BlockPosition specifies where to insert blocks within a parent.
type BlockPosition struct {
	Type string // "end", "start", or "after_block"
	// AfterBlockID is the block ID to insert after (only used when Type == "after_block").
	AfterBlockID string
}

// AppendBlockChildren appends child blocks to a page or block.
// If position is non-nil, blocks are inserted at the specified position.
// Uses DoRequestNoRetry because append is potentially non-idempotent.
func (c *NotionClient) AppendBlockChildren(ctx context.Context, blockID string, children []map[string]interface{}, position *BlockPosition) (*BlockListResponse, int, error) {
	path := "/v1/blocks/" + url.PathEscape(blockID) + "/children"
	body := map[string]interface{}{
		"children": children,
	}
	if position != nil {
		pos := map[string]interface{}{
			"type": position.Type,
		}
		if position.Type == "after_block" && position.AfterBlockID != "" {
			pos["after_block"] = map[string]interface{}{
				"id": position.AfterBlockID,
			}
		}
		body["position"] = pos
	}
	data, status, err := c.DoRequestNoRetry(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, status, err
	}
	var resp BlockListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, status, nerr.Wrap(nerr.CodeInternal, "failed to decode append response", err)
	}
	return &resp, status, nil
}

// UpdateBlock updates a single block's content.
// Uses DoRequest because update is idempotent (PATCH with complete body).
func (c *NotionClient) UpdateBlock(ctx context.Context, blockID string, blockType string, content map[string]interface{}) (*NotionBlock, int, error) {
	path := "/v1/blocks/" + url.PathEscape(blockID)
	body := map[string]interface{}{
		blockType: content,
	}
	data, status, err := c.DoRequest(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, status, err
	}
	var block NotionBlock
	if err := json.Unmarshal(data, &block); err != nil {
		return nil, status, nerr.Wrap(nerr.CodeInternal, "failed to decode block response", err)
	}
	return &block, status, nil
}

// DeleteBlock deletes (archives) a block.
// Uses DoRequest because delete is idempotent.
func (c *NotionClient) DeleteBlock(ctx context.Context, blockID string) (int, error) {
	path := "/v1/blocks/" + url.PathEscape(blockID)
	_, status, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return status, err
	}
	return status, nil
}

// CreatePage creates a new page.
// Uses DoRequestNoRetry because create is potentially non-idempotent.
func (c *NotionClient) CreatePage(ctx context.Context, parent map[string]interface{}, properties map[string]interface{}, children []map[string]interface{}) (*NotionPage, int, error) {
	path := "/v1/pages"
	body := map[string]interface{}{
		"parent":     parent,
		"properties": properties,
	}
	if len(children) > 0 {
		body["children"] = children
	}
	data, status, err := c.DoRequestNoRetry(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, status, err
	}
	var page NotionPage
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, status, nerr.Wrap(nerr.CodeInternal, "failed to decode page response", err)
	}
	return &page, status, nil
}
