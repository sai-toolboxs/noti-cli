package report

import (
	"encoding/json"
	"strings"

	"github.com/sai-toolboxs/noti-cli/internal/client"
)

// OutlineEntry represents a heading in the page outline.
type OutlineEntry struct {
	Level       int    `json:"level"`
	Text        string `json:"text"`
	BlockID     string `json:"block_id"`
	Path        string `json:"path,omitempty"`
	HasChildren bool   `json:"has_children"`
}

// Stats represents aggregate page statistics.
type Stats struct {
	PageID         string         `json:"page_id"`
	Title          string         `json:"title"`
	CreatedTime    string         `json:"created_time"`
	LastEditedTime string         `json:"last_edited_time"`
	Archived       bool           `json:"archived"`
	BlockCounts    map[string]int `json:"block_counts"`
	TotalBlocks    int            `json:"total_blocks"`
	TextLength     int            `json:"text_length"`
	HeadingCounts  map[string]int `json:"heading_counts"`
	SectionCount   int            `json:"section_count"`
}

// extractText extracts plain text from a block's rich_text.
func extractText(b *client.NotionBlock) string {
	if b == nil || b.RichText == nil {
		return ""
	}
	var items []struct {
		PlainText string `json:"plain_text"`
	}
	if err := json.Unmarshal(b.RichText, &items); err != nil {
		return ""
	}
	var sb strings.Builder
	for _, item := range items {
		sb.WriteString(item.PlainText)
	}
	return sb.String()
}

// isHeading reports whether the block type is a heading.
func isHeading(blockType string) bool {
	switch blockType {
	case "heading_1", "heading_2", "heading_3":
		return true
	}
	return false
}

// headingLevel returns the level of a heading block type.
func headingLevel(blockType string) int {
	switch blockType {
	case "heading_1":
		return 1
	case "heading_2":
		return 2
	case "heading_3":
		return 3
	}
	return 0
}
