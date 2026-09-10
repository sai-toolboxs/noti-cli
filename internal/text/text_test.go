package text

import (
	"encoding/json"
	"testing"
)

func TestExtractPlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "nil input",
			input:    "",
			expected: "",
		},
		{
			name:     "single item",
			input:    `[{"plain_text":"Hello world"}]`,
			expected: "Hello world",
		},
		{
			name:     "multiple items",
			input:    `[{"plain_text":"Hello "},{"plain_text":"World"}]`,
			expected: "Hello World",
		},
		{
			name:     "empty items",
			input:    `[{"plain_text":""},{"plain_text":""}]`,
			expected: "",
		},
		{
			name:     "invalid JSON",
			input:    `not json`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw json.RawMessage
			if tt.input != "" {
				raw = json.RawMessage(tt.input)
			}
			result := ExtractPlainText(raw)
			if result != tt.expected {
				t.Errorf("ExtractPlainText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFindExact(t *testing.T) {
	tests := []struct {
		name       string
		richText   string
		searchText string
		wantCount  int
	}{
		{
			name:       "single match in single item",
			richText:   `[{"plain_text":"Hello world"}]`,
			searchText: "world",
			wantCount:  1,
		},
		{
			name:       "no match",
			richText:   `[{"plain_text":"Hello world"}]`,
			searchText: "xyz",
			wantCount:  0,
		},
		{
			name:       "multiple matches across items",
			richText:   `[{"plain_text":"target text "},{"plain_text":"other "},{"plain_text":"target text"}]`,
			searchText: "target text",
			wantCount:  2,
		},
		{
			name:       "empty search text",
			richText:   `[{"plain_text":"Hello"}]`,
			searchText: "",
			wantCount:  0,
		},
		{
			name:       "nil rich text",
			richText:   "",
			searchText: "Hello",
			wantCount:  0,
		},
		{
			name:       "exact full match",
			richText:   `[{"plain_text":"Hello"}]`,
			searchText: "Hello",
			wantCount:  1,
		},
		{
			name:       "partial overlap not matched",
			richText:   `[{"plain_text":"Hello"}]`,
			searchText: "HelloWorld",
			wantCount:  0,
		},
		{
			name:       "search spanning two items",
			richText:   `[{"plain_text":"Hello "},{"plain_text":"World"}]`,
			searchText: "o Wo",
			wantCount:  1,
		},
		{
			name:       "duplicate text in same item",
			richText:   `[{"plain_text":"abc abc"}]`,
			searchText: "abc",
			wantCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw json.RawMessage
			if tt.richText != "" {
				raw = json.RawMessage(tt.richText)
			}
			matches := FindExact(raw, tt.searchText)
			if len(matches) != tt.wantCount {
				t.Errorf("FindExact() returned %d matches, want %d", len(matches), tt.wantCount)
			}
		})
	}
}

func TestReplaceText(t *testing.T) {
	tests := []struct {
		name            string
		richText        string
		searchText      string
		replacementText string
		wantReplCount   int
		wantPlain       string
	}{
		{
			name:            "single replacement in single item",
			richText:        `[{"type":"text","text":{"content":"Hello world"},"plain_text":"Hello world"}]`,
			searchText:      "world",
			replacementText: "Go",
			wantReplCount:   1,
			wantPlain:       "Hello Go",
		},
		{
			name:            "no match",
			richText:        `[{"type":"text","text":{"content":"Hello"},"plain_text":"Hello"}]`,
			searchText:      "xyz",
			replacementText: "Go",
			wantReplCount:   0,
			wantPlain:       "Hello",
		},
		{
			name:            "multiple replacements in same item",
			richText:        `[{"type":"text","text":{"content":"abc abc"},"plain_text":"abc abc"}]`,
			searchText:      "abc",
			replacementText: "xyz",
			wantReplCount:   2,
			wantPlain:       "xyz xyz",
		},
		{
			name:            "empty search",
			richText:        `[{"type":"text","text":{"content":"Hello"},"plain_text":"Hello"}]`,
			searchText:      "",
			replacementText: "Go",
			wantReplCount:   0,
			wantPlain:       "Hello",
		},
		{
			name:            "nil rich text",
			richText:        "",
			searchText:      "Hello",
			replacementText: "Go",
			wantReplCount:   0,
			wantPlain:       "",
		},
		{
			name:            "replacement preserves annotations",
			richText:        `[{"type":"text","text":{"content":"Hello bold"},"plain_text":"Hello bold","annotations":{"bold":true}}]`,
			searchText:      "bold",
			replacementText: "strong",
			wantReplCount:   1,
			wantPlain:       "Hello strong",
		},
		{
			name:            "replacement in second item",
			richText:        `[{"type":"text","text":{"content":"Hello "},"plain_text":"Hello "},{"type":"text","text":{"content":"World"},"plain_text":"World"}]`,
			searchText:      "World",
			replacementText: "Go",
			wantReplCount:   1,
			wantPlain:       "Hello Go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw json.RawMessage
			if tt.richText != "" {
				raw = json.RawMessage(tt.richText)
			}
			result, count, err := ReplaceText(raw, tt.searchText, tt.replacementText)
			if err != nil {
				t.Fatalf("ReplaceText() unexpected error: %v", err)
			}
			if count != tt.wantReplCount {
				t.Errorf("ReplaceText() returned %d replacements, want %d", count, tt.wantReplCount)
			}
			plain := ExtractPlainText(result)
			if plain != tt.wantPlain {
				t.Errorf("ReplaceText() result plain text = %q, want %q", plain, tt.wantPlain)
			}
		})
	}
}

func TestValidateExpected(t *testing.T) {
	tests := []struct {
		name    string
		current string
		want    string
		wantErr bool
	}{
		{
			name:    "match",
			current: "Hello world",
			want:    "Hello world",
			wantErr: false,
		},
		{
			name:    "mismatch",
			current: "Hello world",
			want:    "Hello Go",
			wantErr: true,
		},
		{
			name:    "empty both",
			current: "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "empty current",
			current: "",
			want:    "something",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExpected(tt.current, tt.want)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExpected() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReplaceTextPreservesStructure(t *testing.T) {
	input := `[{"type":"text","text":{"content":"The quick brown fox"},"plain_text":"The quick brown fox","annotations":{"bold":false}}]`
	result, count, err := ReplaceText(json.RawMessage(input), "brown", "red")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 replacement, got %d", count)
	}

	// Verify the result is valid JSON with the expected structure.
	var items []RichTextItem
	if err := json.Unmarshal(result, &items); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].PlainText != "The quick red fox" {
		t.Errorf("plain_text = %q, want %q", items[0].PlainText, "The quick red fox")
	}
	if items[0].Type != "text" {
		t.Errorf("type = %q, want %q", items[0].Type, "text")
	}
	if items[0].Text == nil || items[0].Text["content"] != "The quick red fox" {
		t.Errorf("text.content not updated correctly")
	}
}
