package section

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

func TestFindSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "h1",
					"type":   "heading_2",
					"heading_2": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"plain_text": "Findings"},
						},
					},
				},
				{
					"object": "block",
					"id":     "p1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"plain_text": "Content A"},
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

	code := Find(context.Background(), nc, "page-1", "", "Findings", "", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("found section")) {
		t.Errorf("expected 'found section', got %q", stdout.String())
	}
}

func TestFindJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "h1",
					"type":   "heading_2",
					"heading_2": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"plain_text": "Findings"},
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
	out := output.NewWithWriters(output.FormatJSON, &stdout, &stderr)

	code := Find(context.Background(), nc, "page-1", "", "Findings", "", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	section, ok := result["section"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'section' key in JSON")
	}
	heading, ok := section["heading"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'heading' key in section")
	}
	if heading["text"] != "Findings" {
		t.Errorf("expected heading text 'Findings', got %v", heading["text"])
	}
}

func TestFindEmptyPageID(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Find(context.Background(), nc, "", "", "Findings", "", out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestFindHeadingNotFound(t *testing.T) {
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

	code := Find(context.Background(), nc, "page-1", "", "Nonexistent", "", out)
	if code != 4 {
		t.Errorf("expected exit code 4 (not found), got %d", code)
	}
}

func TestReadSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "h1",
					"type":   "heading_2",
					"heading_2": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"plain_text": "Findings"},
						},
					},
				},
				{
					"object": "block",
					"id":     "p1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"plain_text": "Content A"},
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

	code := Read(context.Background(), nc, "page-1", "", "Findings", "", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("## Findings")) {
		t.Errorf("expected '## Findings', got %q", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Content A")) {
		t.Errorf("expected 'Content A', got %q", stdout.String())
	}
}

func TestAppendSuccess(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		callCount++
		if callCount == 1 {
			// First call: fetch all children.
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "block",
						"id":     "h1",
						"type":   "heading_2",
						"heading_2": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "Findings"},
							},
						},
					},
				},
				"has_more": false,
			})
		} else {
			// Second call: verify position parameter and append blocks.
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			pos, ok := body["position"].(map[string]interface{})
			if !ok {
				t.Error("expected position in request body")
			} else {
				if pos["type"] != "after_block" {
					t.Errorf("expected position type 'after_block', got %v", pos["type"])
				}
				after, ok := pos["after_block"].(map[string]interface{})
				if !ok {
					t.Error("expected after_block in position")
				} else if after["id"] != "h1" {
					t.Errorf("expected after_block id 'h1' (heading), got %v", after["id"])
				}
			}
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
		}
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"New content"}}]}}]`)
	code := Append(context.Background(), nc, "page-1", "", "Findings", "", input, out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("appended 1 block(s)")) {
		t.Errorf("expected append confirmation, got %q", stdout.String())
	}
}

func TestAppendEmptyInput(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := Append(context.Background(), nc, "page-1", "", "Findings", "", nil, out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestReplaceSuccess(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		callCount++
		if callCount == 1 {
			// First call: fetch all children.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "block",
						"id":     "h1",
						"type":   "heading_2",
						"heading_2": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "Findings"},
							},
						},
					},
					{
						"object": "block",
						"id":     "p1",
						"type":   "paragraph",
						"paragraph": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "Old content"},
							},
						},
					},
				},
				"has_more": false,
			})
		} else if callCount == 2 {
			// Second call: append new blocks after heading — verify position.
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			pos, ok := body["position"].(map[string]interface{})
			if !ok {
				t.Error("expected position in request body")
			} else {
				if pos["type"] != "after_block" {
					t.Errorf("expected position type 'after_block', got %v", pos["type"])
				}
				after, ok := pos["after_block"].(map[string]interface{})
				if !ok {
					t.Error("expected after_block in position")
				} else if after["id"] != "h1" {
					t.Errorf("expected after_block id 'h1' (heading), got %v", after["id"])
				}
			}
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
		} else {
			// Third call: re-fetch for post-validation.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "block",
						"id":     "h1",
						"type":   "heading_2",
						"heading_2": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "Findings"},
							},
						},
					},
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
		}
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	input := []byte(`[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"New content"}}]}}]`)
	code := Replace(context.Background(), nc, "page-1", "", "Findings", "", input, out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("replaced section")) {
		t.Errorf("expected replace confirmation, got %q", stdout.String())
	}
}

func TestInsertAfterSuccess(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		callCount++
		if callCount == 1 {
			// First call: fetch all children.
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "block",
						"id":     "h1",
						"type":   "heading_2",
						"heading_2": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "Findings"},
							},
						},
					},
					{
						"object": "block",
						"id":     "p1",
						"type":   "paragraph",
						"paragraph": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "Content A"},
							},
						},
					},
				},
				"has_more": false,
			})
		} else {
			// Second call: verify position and append blocks.
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			pos, ok := body["position"].(map[string]interface{})
			if !ok {
				t.Error("expected position in request body")
			} else {
				if pos["type"] != "after_block" {
					t.Errorf("expected position type 'after_block', got %v", pos["type"])
				}
				after, ok := pos["after_block"].(map[string]interface{})
				if !ok {
					t.Error("expected after_block in position")
				} else if after["id"] != "p1" {
					t.Errorf("expected after_block id 'p1' (last block), got %v", after["id"])
				}
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "block",
						"id":     "new-block-1",
						"type":   "paragraph",
						"paragraph": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "Inserted content"},
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

	input := []byte(`[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Inserted content"}}]}}]`)
	code := InsertAfter(context.Background(), nc, "page-1", "", "Findings", "", input, out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("inserted 1 block(s)")) {
		t.Errorf("expected insert confirmation, got %q", stdout.String())
	}
}

func TestTextFindSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "p1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"type": "text", "text": map[string]interface{}{"content": "This is test content"}, "plain_text": "This is test content"},
						},
					},
				},
				{
					"object": "block",
					"id":     "p2",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"type": "text", "text": map[string]interface{}{"content": "Another paragraph"}, "plain_text": "Another paragraph"},
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

	code := TextFind(context.Background(), nc, "page-1", "", "", "", "test content", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("found 1 match")) {
		t.Errorf("expected match confirmation, got %q", stdout.String())
	}
}

func TestTextFindNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "p1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"type": "text", "text": map[string]interface{}{"content": "Hello world"}, "plain_text": "Hello world"},
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

	code := TextFind(context.Background(), nc, "page-1", "", "", "", "nonexistent", out)
	if code != 4 {
		t.Errorf("expected exit code 4 (not found), got %d", code)
	}
}

func TestTextFindAmbiguous(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "p1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"type": "text", "text": map[string]interface{}{"content": "target text here"}, "plain_text": "target text here"},
						},
					},
				},
				{
					"object": "block",
					"id":     "p2",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"type": "text", "text": map[string]interface{}{"content": "also target text"}, "plain_text": "also target text"},
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

	code := TextFind(context.Background(), nc, "page-1", "", "", "", "target text", out)
	if code != 0 {
		t.Errorf("expected exit code 0 (find returns matches), got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("found 2 match")) {
		t.Errorf("expected 2 matches, got %q", stdout.String())
	}
}

func TestTextFindJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "p1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"type": "text", "text": map[string]interface{}{"content": "Hello world"}, "plain_text": "Hello world"},
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
	out := output.NewWithWriters(output.FormatJSON, &stdout, &stderr)

	code := TextFind(context.Background(), nc, "page-1", "", "", "", "Hello", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result["total_matches"] != float64(1) {
		t.Errorf("expected total_matches 1, got %v", result["total_matches"])
	}
}

func TestTextFindEmptyPageID(t *testing.T) {
	nc := client.New(&http.Client{}, "http://unused", "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := TextFind(context.Background(), nc, "", "", "", "", "test", out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestTextFindEmptySearchText(t *testing.T) {
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

	code := TextFind(context.Background(), nc, "page-1", "", "", "", "", out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestTextReplaceSuccess(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		callCount++
		if callCount == 1 {
			// First call: fetch all children.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "block",
						"id":     "p1",
						"type":   "paragraph",
						"paragraph": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"type": "text", "text": map[string]interface{}{"content": "Hello world"}, "plain_text": "Hello world"},
							},
						},
					},
				},
				"has_more": false,
			})
		} else if callCount == 2 {
			// Second call: GetBlock for the target.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "block",
				"id":     "p1",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]interface{}{"content": "Hello world"}, "plain_text": "Hello world"},
					},
				},
			})
		} else if callCount == 3 {
			// Third call: UpdateBlock.
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if r.Method != http.MethodPatch {
				t.Errorf("expected PATCH method, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "block",
				"id":     "p1",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]interface{}{"content": "Hello Go"}, "plain_text": "Hello Go"},
					},
				},
			})
		} else {
			// Fourth call: verification re-read.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "block",
				"id":     "p1",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]interface{}{"content": "Hello Go"}, "plain_text": "Hello Go"},
					},
				},
			})
		}
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := TextReplace(context.Background(), nc, "page-1", "", "", "", "world", "Go", "", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("replaced")) {
		t.Errorf("expected replace confirmation, got %q", stdout.String())
	}
}

func TestTextReplaceNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "p1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"type": "text", "text": map[string]interface{}{"content": "Hello world"}, "plain_text": "Hello world"},
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

	code := TextReplace(context.Background(), nc, "page-1", "", "", "", "nonexistent", "replacement", "", out)
	if code != 4 {
		t.Errorf("expected exit code 4 (not found), got %d", code)
	}
}

func TestTextReplaceAmbiguous(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "p1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"type": "text", "text": map[string]interface{}{"content": "target text here"}, "plain_text": "target text here"},
						},
					},
				},
				{
					"object": "block",
					"id":     "p2",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"type": "text", "text": map[string]interface{}{"content": "also target text"}, "plain_text": "also target text"},
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

	code := TextReplace(context.Background(), nc, "page-1", "", "", "", "target text", "replacement", "", out)
	if code != 5 {
		t.Errorf("expected exit code 5 (ambiguous), got %d", code)
	}
}

func TestTextReplaceExpectedMismatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "block",
					"id":     "p1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{"type": "text", "text": map[string]interface{}{"content": "Hello world"}, "plain_text": "Hello world"},
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

	code := TextReplace(context.Background(), nc, "page-1", "", "", "", "world", "Go", "wrong expected", out)
	if code != 6 {
		t.Errorf("expected exit code 6 (validation), got %d", code)
	}
}

func TestTextReplaceOnlyTargetBlockChanged(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		callCount++
		if callCount == 1 {
			// First call: fetch all children.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "block",
						"id":     "p1",
						"type":   "paragraph",
						"paragraph": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"type": "text", "text": map[string]interface{}{"content": "Hello world"}, "plain_text": "Hello world"},
							},
						},
					},
					{
						"object": "block",
						"id":     "p2",
						"type":   "paragraph",
						"paragraph": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"type": "text", "text": map[string]interface{}{"content": "Goodbye world"}, "plain_text": "Goodbye world"},
							},
						},
					},
				},
				"has_more": false,
			})
		} else if callCount == 2 {
			// Second call: GetBlock for p1.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "block",
				"id":     "p1",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]interface{}{"content": "Hello world"}, "plain_text": "Hello world"},
					},
				},
			})
		} else if callCount == 3 {
			// Third call: UpdateBlock for p1 only.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "block",
				"id":     "p1",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]interface{}{"content": "Hello Go"}, "plain_text": "Hello Go"},
					},
				},
			})
		} else {
			// Fourth call: verification re-read.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "block",
				"id":     "p1",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]interface{}{"content": "Hello Go"}, "plain_text": "Hello Go"},
					},
				},
			})
		}
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatText, &stdout, &stderr)

	code := TextReplace(context.Background(), nc, "page-1", "p1", "", "", "world", "Go", "", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("p1")) {
		t.Errorf("expected output to reference block p1, got %q", stdout.String())
	}
}

func TestTextReplaceJSONSuccess(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "block",
						"id":     "p1",
						"type":   "paragraph",
						"paragraph": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"type": "text", "text": map[string]interface{}{"content": "Hello world"}, "plain_text": "Hello world"},
							},
						},
					},
				},
				"has_more": false,
			})
		} else if callCount == 2 {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "block",
				"id":     "p1",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]interface{}{"content": "Hello world"}, "plain_text": "Hello world"},
					},
				},
			})
		} else if callCount == 3 {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "block",
				"id":     "p1",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]interface{}{"content": "Hello Go"}, "plain_text": "Hello Go"},
					},
				},
			})
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "block",
				"id":     "p1",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]interface{}{"content": "Hello Go"}, "plain_text": "Hello Go"},
					},
				},
			})
		}
	}))
	defer ts.Close()

	nc := client.New(&http.Client{}, ts.URL, "2022-06-28")
	var stdout, stderr bytes.Buffer
	out := output.NewWithWriters(output.FormatJSON, &stdout, &stderr)

	code := TextReplace(context.Background(), nc, "page-1", "", "", "", "world", "Go", "", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	replaced, ok := result["replaced"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'replaced' key in JSON")
	}
	if replaced["block_id"] != "p1" {
		t.Errorf("expected block_id p1, got %v", replaced["block_id"])
	}
}
