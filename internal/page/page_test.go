package page

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"noti-cli/internal/client"
	"noti-cli/internal/output"
)

func TestInfoSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":           "page",
			"id":               "page-123",
			"created_time":     "2026-09-09T00:00:00Z",
			"last_edited_time": "2026-09-09T01:00:00Z",
			"archived":         false,
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type": "title",
					"title": []map[string]interface{}{
						{"plain_text": "Test Page"},
					},
				},
				"Status": map[string]interface{}{
					"type": "select",
					"select": map[string]interface{}{
						"name": "Active",
					},
				},
			},
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Info(context.Background(), nc, "page-123", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Test Page")) {
		t.Errorf("expected title, got %q", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Status: Active")) {
		t.Errorf("expected Status property, got %q", stdout.String())
	}
}

func TestInfoJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "page",
			"id":     "page-123",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type": "title",
					"title": []map[string]interface{}{
						{"plain_text": "Test Page"},
					},
				},
			},
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatJSON, &stdout, &stderr)

	code := Info(context.Background(), nc, "page-123", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result["id"] != "page-123" {
		t.Errorf("expected id page-123, got %v", result["id"])
	}
}

func TestInfoEmptyID(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Info(context.Background(), nc, "", out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestInfo404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "error",
			"status":  404,
			"code":    "object_not_found",
			"message": "page not found",
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Info(context.Background(), nc, "missing-page", out)
	if code != 4 {
		t.Errorf("expected exit code 4 (not found), got %d", code)
	}
}

func TestChildrenSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "block-a",
					"type":   "heading_1",
					"heading_1": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"plain_text": "My Page"},
						},
					},
				},
				{
					"object": "block",
					"id":     "block-b",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"plain_text": "Content here"},
						},
					},
				},
			},
			"has_more": false,
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Children(context.Background(), nc, "page-abc", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("2 child block(s)")) {
		t.Errorf("expected count, got %q", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("My Page")) {
		t.Errorf("expected heading text, got %q", stdout.String())
	}
}

func TestReadSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		if path == "/v1/pages/page-xyz" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object":           "page",
				"id":               "page-xyz",
				"created_time":     "2026-09-09T00:00:00Z",
				"last_edited_time": "2026-09-09T01:00:00Z",
				"archived":         false,
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type": "title",
						"title": []map[string]interface{}{
							{"plain_text": "Full Page"},
						},
					},
				},
			})
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "block",
						"id":     "block-1",
						"type":   "heading_2",
						"heading_2": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "Section"},
							},
						},
					},
					{
						"object": "block",
						"id":     "block-2",
						"type":   "paragraph",
						"paragraph": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "Body text"},
							},
						},
					},
				},
				"has_more": false,
			})
		}
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Read(context.Background(), nc, "page-xyz", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("# Full Page")) {
		t.Errorf("expected title, got %q", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("## Section")) {
		t.Errorf("expected heading, got %q", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Body text")) {
		t.Errorf("expected body text, got %q", stdout.String())
	}
}

func TestReadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		if path == "/v1/pages/page-json" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object":           "page",
				"id":               "page-json",
				"created_time":     "2026-09-09T00:00:00Z",
				"last_edited_time": "2026-09-09T01:00:00Z",
				"archived":         false,
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type": "title",
						"title": []map[string]interface{}{
							{"plain_text": "JSON Page"},
						},
					},
				},
			})
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object":   "list",
				"results":  []interface{}{},
				"has_more": false,
			})
		}
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatJSON, &stdout, &stderr)

	code := Read(context.Background(), nc, "page-json", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	page, ok := result["page"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'page' key in JSON")
	}
	if page["id"] != "page-json" {
		t.Errorf("expected page id page-json, got %v", page["id"])
	}
}

func TestReadEmptyID(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Read(context.Background(), nc, "", out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestChildrenEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "list",
			"results":  []interface{}{},
			"has_more": false,
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Children(context.Background(), nc, "empty-page", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("(no children)")) {
		t.Errorf("expected empty message, got %q", stdout.String())
	}
}

func TestCreateSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
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
					"type": "title",
					"title": []map[string]interface{}{
						{"plain_text": "New Page"},
					},
				},
			},
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`{"parent":{"page_id":"parent-page-id"},"properties":{"title":{"title":[{"type":"text","text":{"content":"New Page"}}]}}}`)
	code := Create(context.Background(), nc, input, out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("created page new-page-123")) {
		t.Errorf("expected create confirmation, got %q", stdout.String())
	}
}

func TestCreateJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "page",
			"id":     "new-page-123",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type": "title",
					"title": []map[string]interface{}{
						{"plain_text": "New Page"},
					},
				},
			},
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatJSON, &stdout, &stderr)

	input := []byte(`{"parent":{"page_id":"parent-page-id"},"properties":{"title":{"title":[{"type":"text","text":{"content":"New Page"}}]}}}`)
	code := Create(context.Background(), nc, input, out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result["id"] != "new-page-123" {
		t.Errorf("expected id new-page-123, got %v", result["id"])
	}
}

func TestCreateEmptyInput(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Create(context.Background(), nc, nil, out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestCreateInvalidJSON(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`not json`)
	code := Create(context.Background(), nc, input, out)
	if code != 6 {
		t.Errorf("expected exit code 6 (validation), got %d", code)
	}
}

func TestCreateNoParent(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`{"properties":{"title":{"title":[{"type":"text","text":{"content":"No Parent"}}]}}}`)
	code := Create(context.Background(), nc, input, out)
	if code != 6 {
		t.Errorf("expected exit code 6 (validation), got %d", code)
	}
}

func TestCreateNoProperties(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`{"parent":{"page_id":"parent-page-id"}}`)
	code := Create(context.Background(), nc, input, out)
	if code != 6 {
		t.Errorf("expected exit code 6 (validation), got %d", code)
	}
}

func TestCreate400(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "error",
			"status":  400,
			"code":    "validation_error",
			"message": "invalid page properties",
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`{"parent":{"page_id":"parent-page-id"},"properties":{"invalid":true}}`)
	code := Create(context.Background(), nc, input, out)
	if code != 6 {
		t.Errorf("expected exit code 6 (validation), got %d", code)
	}
}
