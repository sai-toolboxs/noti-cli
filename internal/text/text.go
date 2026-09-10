package text

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RichTextItem represents a single Notion rich_text element.
type RichTextItem struct {
	Type      string                 `json:"type"`
	Text      map[string]interface{} `json:"text,omitempty"`
	PlainText string                 `json:"plain_text"`
	Anno      map[string]interface{} `json:"annotations,omitempty"`
}

// Match represents a found text match within a rich_text array.
type Match struct {
	RichTextIndex int
	Offset        int
	EndOffset     int
}

// ExtractPlainText concatenates all plain_text items into a single string.
func ExtractPlainText(richText json.RawMessage) string {
	if richText == nil {
		return ""
	}
	var items []struct {
		PlainText string `json:"plain_text"`
	}
	if err := json.Unmarshal(richText, &items); err != nil {
		return ""
	}
	var sb strings.Builder
	for _, item := range items {
		sb.WriteString(item.PlainText)
	}
	return sb.String()
}

// FindExact searches for exact text within a block's rich_text array.
// Returns matches with their position in the concatenated plain text.
// The search is a substring match within the full concatenated plain text.
func FindExact(richText json.RawMessage, searchText string) []Match {
	if richText == nil || searchText == "" {
		return nil
	}

	var items []struct {
		PlainText string `json:"plain_text"`
	}
	if err := json.Unmarshal(richText, &items); err != nil {
		return nil
	}

	full := ExtractPlainText(richText)
	if !strings.Contains(full, searchText) {
		return nil
	}

	// Build a mapping from global offset to rich_text item index.
	type offsetRange struct {
		itemIndex int
		start     int // global start offset
		end       int // global end offset
	}
	var ranges []offsetRange
	globalOffset := 0
	for i, item := range items {
		itemLen := len(item.PlainText)
		ranges = append(ranges, offsetRange{
			itemIndex: i,
			start:     globalOffset,
			end:       globalOffset + itemLen,
		})
		globalOffset += itemLen
	}

	// Search the concatenated string and map each match to its item.
	var matches []Match
	searchStart := 0
	for {
		idx := strings.Index(full[searchStart:], searchText)
		if idx < 0 {
			break
		}
		globalPos := searchStart + idx

		// Find which item contains the start of this match.
		for _, r := range ranges {
			if globalPos >= r.start && globalPos < r.end {
				localOffset := globalPos - r.start
				matches = append(matches, Match{
					RichTextIndex: r.itemIndex,
					Offset:        localOffset,
					EndOffset:     localOffset + len(searchText),
				})
				break
			}
		}

		searchStart = globalPos + 1
	}

	return matches
}

// ReplaceText replaces all occurrences of searchText with replacementText
// within the rich_text array, preserving annotations on the first matched item.
// Returns the modified rich_text JSON and the number of replacements made.
// If searchText is not found, returns the original richText unchanged with 0 replacements.
func ReplaceText(richText json.RawMessage, searchText, replacementText string) (json.RawMessage, int, error) {
	if richText == nil || searchText == "" {
		return richText, 0, nil
	}

	var items []RichTextItem
	if err := json.Unmarshal(richText, &items); err != nil {
		return nil, 0, fmt.Errorf("failed to parse rich_text: %w", err)
	}

	// Build the full plain text and find all occurrences.
	full := ExtractPlainText(richText)
	if !strings.Contains(full, searchText) {
		return richText, 0, nil
	}

	// Strategy: find which item(s) contain the search text.
	// For simplicity and safety, handle the common case where the search text
	// falls within a single rich_text item. For cross-item spans, merge into
	// the first item containing the match.
	replacements := 0
	offset := 0
	result := make([]RichTextItem, len(items))
	copy(result, items)

	for i := range result {
		item := &result[i]
		offset += len(item.PlainText)

		// Check if any occurrence of searchText starts within this item.
		for {
			idx := strings.Index(item.PlainText, searchText)
			if idx < 0 {
				break
			}

			// Replace in this item.
			item.PlainText = item.PlainText[:idx] + replacementText + item.PlainText[idx+len(searchText):]
			replacements++

			// Update the text.content field if it exists.
			if item.Text != nil {
				if content, ok := item.Text["content"].(string); ok {
					// Find and replace in the content string too.
					if cidx := strings.Index(content, searchText); cidx >= 0 {
						item.Text["content"] = content[:cidx] + replacementText + content[cidx+len(searchText):]
					}
				}
			}
		}
	}

	if replacements == 0 {
		return richText, 0, nil
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal rich_text: %w", err)
	}

	return data, replacements, nil
}

// ValidateExpected checks if the current plain text matches the expected text.
// Returns nil if they match, or an error describing the mismatch.
func ValidateExpected(currentPlain, expected string) error {
	if currentPlain == expected {
		return nil
	}
	return fmt.Errorf("expected %q but found %q", expected, currentPlain)
}
