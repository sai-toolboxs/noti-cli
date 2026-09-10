package report

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"noti-cli/internal/client"
	"noti-cli/internal/output"
)

func setupTestClient(handler http.Handler) (*client.NotionClient, *httptest.Server) {
	server := httptest.NewServer(handler)
	httpClient := server.Client()
	nc := client.New(httpClient, server.URL, "2022-06-28")
	return nc, server
}

func TestTreeSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "list",
			"results": []interface{}{},
			"count":   0,
		})
	})
	nc, server := setupTestClient(handler)
	defer server.Close()

	out := output.New(output.FormatJSON)
	code := Tree(context.Background(), nc, "test-page-id", 0, out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestOutlineSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "list",
			"results": []interface{}{},
			"count":   0,
		})
	})
	nc, server := setupTestClient(handler)
	defer server.Close()

	out := output.New(output.FormatJSON)
	code := Outline(context.Background(), nc, "test-page-id", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestStatsSuccess(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// Page metadata.
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":               "test-page-id",
				"object":           "page",
				"created_time":     "2026-01-01",
				"last_edited_time": "2026-01-02",
				"archived":         false,
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type": "title",
						"title": []interface{}{
							map[string]interface{}{
								"plain_text": "Test Page",
							},
						},
					},
				},
			})
		} else {
			// Block list.
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "list",
				"results": []interface{}{},
				"count":   0,
			})
		}
	})
	nc, server := setupTestClient(handler)
	defer server.Close()

	out := output.New(output.FormatJSON)
	code := StatsCmd(context.Background(), nc, "test-page-id", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestExportSuccess(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// Page metadata.
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":               "test-page-id",
				"object":           "page",
				"created_time":     "2026-01-01",
				"last_edited_time": "2026-01-02",
				"archived":         false,
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type": "title",
						"title": []interface{}{
							map[string]interface{}{
								"plain_text": "Test Page",
							},
						},
					},
				},
			})
		} else {
			// Block list.
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "list",
				"results": []interface{}{},
				"count":   0,
			})
		}
	})
	nc, server := setupTestClient(handler)
	defer server.Close()

	out := output.New(output.FormatJSON)
	code := Export(context.Background(), nc, "test-page-id", out)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestTreeEmptyPageID(t *testing.T) {
	nc, server := setupTestClient(nil)
	defer server.Close()

	out := output.New(output.FormatJSON)
	code := Tree(context.Background(), nc, "", 0, out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestOutlineEmptyPageID(t *testing.T) {
	nc, server := setupTestClient(nil)
	defer server.Close()

	out := output.New(output.FormatJSON)
	code := Outline(context.Background(), nc, "", out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestStatsEmptyPageID(t *testing.T) {
	nc, server := setupTestClient(nil)
	defer server.Close()

	out := output.New(output.FormatJSON)
	code := StatsCmd(context.Background(), nc, "", out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}

func TestExportEmptyPageID(t *testing.T) {
	nc, server := setupTestClient(nil)
	defer server.Close()

	out := output.New(output.FormatJSON)
	code := Export(context.Background(), nc, "", out)
	if code != 2 {
		t.Errorf("expected exit code 2 (usage), got %d", code)
	}
}
