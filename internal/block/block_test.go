package block

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

func TestGetSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":       "block",
			"id":           "test-block-id",
			"type":         "paragraph",
			"has_children": false,
			"paragraph": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{"plain_text": "Hello world"},
				},
				"color": "default",
			},
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Get(context.Background(), nc, "test-block-id", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Hello world")) {
		t.Errorf("expected output to contain 'Hello world', got %q", stdout.String())
	}
}

func TestGetJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "block",
			"id":     "test-block-id",
			"type":   "paragraph",
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatJSON, &stdout, &stderr)

	code := Get(context.Background(), nc, "test-block-id", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result["id"] != "test-block-id" {
		t.Errorf("expected id test-block-id, got %v", result["id"])
	}
}

func TestGetEmptyID(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Get(context.Background(), nc, "", out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestGet404(t *testing.T) {
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

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Get(context.Background(), nc, "missing-id", out)
	if code != 4 {
		t.Errorf("expected exit code 4 (not found), got %d", code)
	}
}

func TestListSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "child-1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"plain_text": "First paragraph"},
						},
					},
				},
				{
					"object": "block",
					"id":     "child-2",
					"type":   "heading_2",
					"heading_2": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"plain_text": "Section Title"},
						},
					},
				},
			},
			"has_more":    false,
			"next_cursor": nil,
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := List(context.Background(), nc, "parent-id", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("2 child block(s)")) {
		t.Errorf("expected count, got %q", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("First paragraph")) {
		t.Errorf("expected first paragraph text, got %q", stdout.String())
	}
}

func TestListEmpty(t *testing.T) {
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

	code := List(context.Background(), nc, "empty-id", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("(no children)")) {
		t.Errorf("expected empty message, got %q", stdout.String())
	}
}

func TestListEmptyID(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := List(context.Background(), nc, "", out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestAppendSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "new-block-1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"plain_text": "New content"},
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

	input := []byte(`[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"New content"}}]}}]`)
	code := Append(context.Background(), nc, "parent-id", input, out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("appended 1 block(s)")) {
		t.Errorf("expected append confirmation, got %q", stdout.String())
	}
}

func TestAppendJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "new-block-1",
					"type":   "paragraph",
				},
			},
			"has_more": false,
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatJSON, &stdout, &stderr)

	input := []byte(`[{"object":"block","type":"paragraph","paragraph":{"rich_text":[]}}]`)
	code := Append(context.Background(), nc, "parent-id", input, out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result["count"] != float64(1) {
		t.Errorf("expected count 1, got %v", result["count"])
	}
}

func TestAppendEmptyID(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`[{"object":"block","type":"paragraph"}]`)
	code := Append(context.Background(), nc, "", input, out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestAppendEmptyInput(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Append(context.Background(), nc, "parent-id", nil, out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestAppendTextInput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "list",
			"results": []interface{}{},
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte("Hello world")
	code := Append(context.Background(), nc, "parent-id", input, out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestAppend404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "error",
			"status":  404,
			"code":    "object_not_found",
			"message": "parent not found",
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`[{"object":"block","type":"paragraph","paragraph":{"rich_text":[]}}]`)
	code := Append(context.Background(), nc, "missing-id", input, out)
	if code != 4 {
		t.Errorf("expected exit code 4 (not found), got %d", code)
	}
}

func TestUpdateSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "block",
			"id":     "block-123",
			"type":   "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{"plain_text": "Updated content"},
				},
			},
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`{"type":"paragraph","content":{"rich_text":[{"type":"text","text":{"content":"Updated content"}}]}}`)
	code := Update(context.Background(), nc, "block-123", input, out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("updated block block-123")) {
		t.Errorf("expected update confirmation, got %q", stdout.String())
	}
}

func TestUpdateJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "block",
			"id":     "block-123",
			"type":   "paragraph",
		})
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatJSON, &stdout, &stderr)

	input := []byte(`{"paragraph":{"rich_text":[{"type":"text","text":{"content":"Updated"}}]}}`)
	code := Update(context.Background(), nc, "block-123", input, out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result["id"] != "block-123" {
		t.Errorf("expected id block-123, got %v", result["id"])
	}
}

func TestUpdateEmptyID(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`{"type":"paragraph","content":{}}`)
	code := Update(context.Background(), nc, "", input, out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestUpdateEmptyInput(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Update(context.Background(), nc, "block-123", nil, out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestUpdateInvalidJSON(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`not json`)
	code := Update(context.Background(), nc, "block-123", input, out)
	if code != 6 {
		t.Errorf("expected exit code 6 (validation), got %d", code)
	}
}

func TestUpdateNoType(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`{"content":{"rich_text":[]}}`)
	code := Update(context.Background(), nc, "block-123", input, out)
	if code != 6 {
		t.Errorf("expected exit code 6 (validation), got %d", code)
	}
}

func TestDeleteSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Delete(context.Background(), nc, "block-123", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("deleted block block-123")) {
		t.Errorf("expected delete confirmation, got %q", stdout.String())
	}
}

func TestDeleteJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatJSON, &stdout, &stderr)

	code := Delete(context.Background(), nc, "block-123", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result["deleted"] != true {
		t.Errorf("expected deleted=true, got %v", result["deleted"])
	}
	if result["id"] != "block-123" {
		t.Errorf("expected id block-123, got %v", result["id"])
	}
}

func TestDeleteEmptyID(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Delete(context.Background(), nc, "", out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestDelete404(t *testing.T) {
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

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Delete(context.Background(), nc, "missing-block", out)
	if code != 4 {
		t.Errorf("expected exit code 4 (not found), got %d", code)
	}
}
