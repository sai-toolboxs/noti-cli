package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// headerTransport adds auth headers for test HTTP clients.
type headerTransport struct {
	token string
	inner http.RoundTripper
}

func (ht *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+ht.token)
	req.Header.Set("Notion-Version", "2022-06-28")
	return ht.inner.RoundTrip(req)
}

func testHTTPClient(token string) *http.Client {
	return &http.Client{
		Transport: &headerTransport{
			token: token,
			inner: http.DefaultTransport,
		},
	}
}

func TestDoRequestSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing Authorization header")
		}
		if r.Header.Get("Notion-Version") != "2022-06-28" {
			t.Error("missing Notion-Version header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "test-block"})
	}))
	defer ts.Close()

	client := New(testHTTPClient("test-token"), ts.URL, "2022-06-28")

	data, status, err := client.DoRequest(context.Background(), http.MethodGet, "/v1/blocks/test-id", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if len(data) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestDoRequest404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "error",
			"status":  404,
			"code":    "object_not_found",
			"message": "block not found",
		})
	}))
	defer ts.Close()

	client := New(testHTTPClient("test-token"), ts.URL, "2022-06-28")

	_, status, err := client.DoRequestNoRetry(context.Background(), http.MethodGet, "/v1/blocks/missing", nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", status)
	}
}

func TestDoRequest429(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "error",
			"status":  429,
			"code":    "rate_limited",
			"message": "rate limited",
		})
	}))
	defer ts.Close()

	client := New(testHTTPClient("test-token"), ts.URL, "2022-06-28",
		WithMaxRetries(1), WithRetryDelay(0))

	_, _, err := client.DoRequest(context.Background(), http.MethodGet, "/v1/test", nil)
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls (1 initial + 1 retry), got %d", callCount)
	}
}

func TestGetBlock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":       "block",
			"id":           "block-123",
			"type":         "paragraph",
			"has_children": false,
			"paragraph": map[string]interface{}{
				"rich_text": []interface{}{},
				"color":     "default",
			},
		})
	}))
	defer ts.Close()

	client := New(testHTTPClient("test-token"), ts.URL, "2022-06-28")

	block, status, err := client.GetBlock(context.Background(), "block-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if block.ID != "block-123" {
		t.Errorf("expected block ID 'block-123', got %q", block.ID)
	}
	if block.Type != "paragraph" {
		t.Errorf("expected type 'paragraph', got %q", block.Type)
	}
}

func TestListBlockChildren(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []interface{}{
				map[string]interface{}{
					"object": "block",
					"id":     "child-1",
					"type":   "paragraph",
				},
				map[string]interface{}{
					"object": "block",
					"id":     "child-2",
					"type":   "heading_2",
				},
			},
			"has_more":    false,
			"next_cursor": nil,
		})
	}))
	defer ts.Close()

	client := New(testHTTPClient("test-token"), ts.URL, "2022-06-28")

	resp, status, err := client.ListBlockChildren(context.Background(), "parent-block", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if len(resp.Results) != 2 {
		t.Errorf("expected 2 children, got %d", len(resp.Results))
	}
	if resp.HasMore {
		t.Error("expected has_more=false")
	}
}

func TestListAllBlockChildren(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if callCount == 1 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []interface{}{
					map[string]interface{}{
						"object": "block",
						"id":     "child-1",
						"type":   "paragraph",
					},
				},
				"has_more":    true,
				"next_cursor": "cursor-page-2",
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []interface{}{
					map[string]interface{}{
						"object": "block",
						"id":     "child-2",
						"type":   "heading_2",
					},
				},
				"has_more":    false,
				"next_cursor": nil,
			})
		}
	}))
	defer ts.Close()

	client := New(testHTTPClient("test-token"), ts.URL, "2022-06-28")

	blocks, err := client.ListAllBlockChildren(context.Background(), "parent-block")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 2 {
		t.Errorf("expected 2 blocks across pages, got %d", len(blocks))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestFormatBlockSize(t *testing.T) {
	tests := []struct {
		block *NotionBlock
		want  string
	}{
		{nil, "<nil>"},
		{&NotionBlock{ID: "abc", Type: "paragraph"}, "[abc] paragraph"},
		{&NotionBlock{ID: "12345678-1234-1234-1234-123456789abc", Type: "heading_2"}, "[12345678] heading_2"},
	}
	for _, tt := range tests {
		got := FormatBlockSize(tt.block)
		if got != tt.want {
			t.Errorf("FormatBlockSize(%v) = %q, want %q", tt.block, got, tt.want)
		}
	}
}

func TestBlockUnmarshalJSON(t *testing.T) {
	raw := `{
		"object": "block",
		"id": "test-id",
		"type": "paragraph",
		"has_children": false,
		"paragraph": {
			"rich_text": [
				{
					"type": "text",
					"text": {"content": "Hello world"}
				}
			],
			"color": "default"
		}
	}`

	var block NotionBlock
	if err := json.Unmarshal([]byte(raw), &block); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if block.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", block.ID)
	}
	if block.Type != "paragraph" {
		t.Errorf("expected type 'paragraph', got %q", block.Type)
	}
	if block.RichText == nil {
		t.Error("expected RichText to be extracted")
	}
}

func TestAppendBlockChildren(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}
		if r.URL.Path != "/v1/blocks/parent-block/children" {
			t.Errorf("expected path /v1/blocks/parent-block/children, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []interface{}{
				map[string]interface{}{
					"object": "block",
					"id":     "child-1",
					"type":   "paragraph",
				},
			},
			"has_more": false,
		})
	}))
	defer ts.Close()

	client := New(testHTTPClient("test-token"), ts.URL, "2022-06-28")

	children := []map[string]interface{}{
		{
			"object": "block",
			"type":   "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []interface{}{},
			},
		},
	}

	resp, status, err := client.AppendBlockChildren(context.Background(), "parent-block", children, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 child, got %d", len(resp.Results))
	}
}

func TestAppendBlockChildrenWithPosition(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		pos, ok := body["position"].(map[string]interface{})
		if !ok {
			t.Fatal("expected position in request body")
		}
		if pos["type"] != "after_block" {
			t.Errorf("expected position type 'after_block', got %v", pos["type"])
		}
		after, ok := pos["after_block"].(map[string]interface{})
		if !ok {
			t.Fatal("expected after_block in position")
		}
		if after["id"] != "target-block" {
			t.Errorf("expected after_block id 'target-block', got %v", after["id"])
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []interface{}{
				map[string]interface{}{
					"object": "block",
					"id":     "new-child",
					"type":   "paragraph",
				},
			},
			"has_more": false,
		})
	}))
	defer ts.Close()

	nc := New(testHTTPClient("test-token"), ts.URL, "2022-06-28")

	children := []map[string]interface{}{
		{
			"object": "block",
			"type":   "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []interface{}{},
			},
		},
	}

	position := &BlockPosition{
		Type:         "after_block",
		AfterBlockID: "target-block",
	}

	resp, status, err := nc.AppendBlockChildren(context.Background(), "parent-block", children, position)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 child, got %d", len(resp.Results))
	}
}

func TestUpdateBlock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}
		if r.URL.Path != "/v1/blocks/block-123" {
			t.Errorf("expected path /v1/blocks/block-123, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "block",
			"id":     "block-123",
			"type":   "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []interface{}{},
			},
		})
	}))
	defer ts.Close()

	client := New(testHTTPClient("test-token"), ts.URL, "2022-06-28")

	content := map[string]interface{}{
		"rich_text": []interface{}{},
	}

	block, status, err := client.UpdateBlock(context.Background(), "block-123", "paragraph", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if block.ID != "block-123" {
		t.Errorf("expected block ID 'block-123', got %q", block.ID)
	}
}

func TestDeleteBlock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v1/blocks/block-123" {
			t.Errorf("expected path /v1/blocks/block-123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := New(testHTTPClient("test-token"), ts.URL, "2022-06-28")

	status, err := client.DeleteBlock(context.Background(), "block-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", status)
	}
}

func TestCreatePage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v1/pages" {
			t.Errorf("expected path /v1/pages, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":           "page",
			"id":               "new-page-123",
			"created_time":     "2026-09-09T00:00:00Z",
			"last_edited_time": "2026-09-09T00:00:00Z",
			"archived":         false,
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":  "title",
					"title": []interface{}{},
				},
			},
		})
	}))
	defer ts.Close()

	client := New(testHTTPClient("test-token"), ts.URL, "2022-06-28")

	parent := map[string]interface{}{
		"page_id": "parent-page-id",
	}
	properties := map[string]interface{}{
		"title": map[string]interface{}{
			"title": []interface{}{},
		},
	}

	page, status, err := client.CreatePage(context.Background(), parent, properties, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if page.ID != "new-page-123" {
		t.Errorf("expected page ID 'new-page-123', got %q", page.ID)
	}
}

func TestCreatePageWithChildren(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":           "page",
			"id":               "new-page-456",
			"created_time":     "2026-09-09T00:00:00Z",
			"last_edited_time": "2026-09-09T00:00:00Z",
			"archived":         false,
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":  "title",
					"title": []interface{}{},
				},
			},
		})
	}))
	defer ts.Close()

	client := New(testHTTPClient("test-token"), ts.URL, "2022-06-28")

	parent := map[string]interface{}{
		"page_id": "parent-page-id",
	}
	properties := map[string]interface{}{
		"title": map[string]interface{}{
			"title": []interface{}{},
		},
	}
	children := []map[string]interface{}{
		{
			"object": "block",
			"type":   "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []interface{}{},
			},
		},
	}

	page, status, err := client.CreatePage(context.Background(), parent, properties, children)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if page.ID != "new-page-456" {
		t.Errorf("expected page ID 'new-page-456', got %q", page.ID)
	}
}
