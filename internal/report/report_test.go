package report

import (
	"encoding/json"
	"testing"

	"github.com/sai-toolboxs/noti-cli/internal/client"
)

func TestBuildTreeEmpty(t *testing.T) {
	tree := BuildTree(nil)
	if tree != nil {
		t.Errorf("expected nil, got %v", tree)
	}
}

func TestBuildTreeSingleBlock(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "1", Type: "paragraph", RichText: json.RawMessage(`[{"plain_text":"Hello"}]`)},
	}
	tree := BuildTree(blocks)
	if len(tree) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tree))
	}
	if tree[0].Text != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", tree[0].Text)
	}
	if tree[0].Depth != 0 {
		t.Errorf("expected depth 0, got %d", tree[0].Depth)
	}
}

func TestBuildTreeHeadingHierarchy(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "1", Type: "heading_1", RichText: json.RawMessage(`[{"plain_text":"Title"}]`)},
		{ID: "2", Type: "paragraph", RichText: json.RawMessage(`[{"plain_text":"Content"}]`)},
		{ID: "3", Type: "heading_2", RichText: json.RawMessage(`[{"plain_text":"Section"}]`)},
		{ID: "4", Type: "paragraph", RichText: json.RawMessage(`[{"plain_text":"More content"}]`)},
	}
	tree := BuildTree(blocks)
	if len(tree) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tree))
	}
	if tree[0].Text != "Title" {
		t.Errorf("expected 'Title', got '%s'", tree[0].Text)
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(tree[0].Children))
	}
	if tree[0].Children[0].Text != "Content" {
		t.Errorf("expected 'Content', got '%s'", tree[0].Children[0].Text)
	}
	if tree[0].Children[1].Text != "Section" {
		t.Errorf("expected 'Section', got '%s'", tree[0].Children[1].Text)
	}
}

func TestBuildTreeWithDepth(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "1", Type: "heading_1", RichText: json.RawMessage(`[{"plain_text":"Title"}]`)},
		{ID: "2", Type: "heading_2", RichText: json.RawMessage(`[{"plain_text":"Section"}]`)},
		{ID: "3", Type: "heading_3", RichText: json.RawMessage(`[{"plain_text":"Subsection"}]`)},
	}
	tree := BuildTreeWithDepth(blocks, 2)
	if len(tree) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tree))
	}
	if len(tree[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(tree[0].Children))
	}
	if len(tree[0].Children[0].Children) != 0 {
		t.Errorf("expected 0 grandchildren due to depth limit")
	}
}

func TestCountNodes(t *testing.T) {
	nodes := []*TreeNode{
		{Text: "a", Children: []*TreeNode{
			{Text: "b"},
			{Text: "c", Children: []*TreeNode{
				{Text: "d"},
			}},
		}},
	}
	count := CountNodes(nodes)
	if count != 4 {
		t.Errorf("expected 4, got %d", count)
	}
}

func TestBuildOutlineEmpty(t *testing.T) {
	entries := BuildOutline(nil)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestBuildOutlineHeadings(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "1", Type: "heading_1", RichText: json.RawMessage(`[{"plain_text":"Title"}]`)},
		{ID: "2", Type: "paragraph", RichText: json.RawMessage(`[{"plain_text":"Content"}]`)},
		{ID: "3", Type: "heading_2", RichText: json.RawMessage(`[{"plain_text":"Section"}]`)},
		{ID: "4", Type: "heading_3", RichText: json.RawMessage(`[{"plain_text":"Subsection"}]`)},
	}
	entries := BuildOutline(blocks)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Text != "Title" || entries[0].Level != 1 {
		t.Errorf("expected 'Title' level 1, got '%s' level %d", entries[0].Text, entries[0].Level)
	}
	if entries[1].Text != "Section" || entries[1].Level != 2 {
		t.Errorf("expected 'Section' level 2, got '%s' level %d", entries[1].Text, entries[1].Level)
	}
	if entries[2].Text != "Subsection" || entries[2].Level != 3 {
		t.Errorf("expected 'Subsection' level 3, got '%s' level %d", entries[2].Text, entries[2].Level)
	}
}

func TestOutlineByLevel(t *testing.T) {
	entries := []OutlineEntry{
		{Level: 1, Text: "A"},
		{Level: 2, Text: "B"},
		{Level: 1, Text: "C"},
		{Level: 3, Text: "D"},
	}
	byLevel := OutlineByLevel(entries)
	if len(byLevel[1]) != 2 {
		t.Errorf("expected 2 level-1 entries, got %d", len(byLevel[1]))
	}
	if len(byLevel[2]) != 1 {
		t.Errorf("expected 1 level-2 entry, got %d", len(byLevel[2]))
	}
	if len(byLevel[3]) != 1 {
		t.Errorf("expected 1 level-3 entry, got %d", len(byLevel[3]))
	}
}

func TestCountByLevel(t *testing.T) {
	entries := []OutlineEntry{
		{Level: 1, Text: "A"},
		{Level: 2, Text: "B"},
		{Level: 1, Text: "C"},
		{Level: 3, Text: "D"},
	}
	counts := CountByLevel(entries)
	if counts["heading_1"] != 2 {
		t.Errorf("expected 2 heading_1, got %d", counts["heading_1"])
	}
	if counts["heading_2"] != 1 {
		t.Errorf("expected 1 heading_2, got %d", counts["heading_2"])
	}
	if counts["heading_3"] != 1 {
		t.Errorf("expected 1 heading_3, got %d", counts["heading_3"])
	}
}

func TestComputeStats(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "1", Type: "heading_1", RichText: json.RawMessage(`[{"plain_text":"Title"}]`)},
		{ID: "2", Type: "paragraph", RichText: json.RawMessage(`[{"plain_text":"Hello world"}]`)},
		{ID: "3", Type: "bulleted_list_item", RichText: json.RawMessage(`[{"plain_text":"Item"}]`)},
	}
	meta := PageMetadata{
		ID:             "page-1",
		Title:          "Test Page",
		CreatedTime:    "2026-01-01",
		LastEditedTime: "2026-01-02",
		Archived:       false,
	}
	stats := ComputeStats(meta, blocks)
	if stats.TotalBlocks != 3 {
		t.Errorf("expected 3 blocks, got %d", stats.TotalBlocks)
	}
	if stats.TextLength != 20 {
		t.Errorf("expected 20 characters, got %d", stats.TextLength)
	}
	if stats.SectionCount != 1 {
		t.Errorf("expected 1 section, got %d", stats.SectionCount)
	}
}

func TestExportMarkdown(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "1", Type: "heading_1", RichText: json.RawMessage(`[{"plain_text":"Title"}]`)},
		{ID: "2", Type: "paragraph", RichText: json.RawMessage(`[{"plain_text":"Hello"}]`)},
	}
	markdown := ExportMarkdown(blocks)
	if markdown != "# Title\n\nHello\n\n" {
		t.Errorf("unexpected markdown: %q", markdown)
	}
}

func TestExtractTextNil(t *testing.T) {
	text := extractText(nil)
	if text != "" {
		t.Errorf("expected empty string, got '%s'", text)
	}
}

func TestExtractTextEmpty(t *testing.T) {
	b := &client.NotionBlock{RichText: nil}
	text := extractText(b)
	if text != "" {
		t.Errorf("expected empty string, got '%s'", text)
	}
}

func TestExtractTextPlainText(t *testing.T) {
	b := &client.NotionBlock{RichText: json.RawMessage(`[{"plain_text":"Hello"}]`)}
	text := extractText(b)
	if text != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", text)
	}
}

func TestExtractTextMultipleItems(t *testing.T) {
	b := &client.NotionBlock{RichText: json.RawMessage(`[{"plain_text":"Hello "},{"plain_text":"world"}]`)}
	text := extractText(b)
	if text != "Hello world" {
		t.Errorf("expected 'Hello world', got '%s'", text)
	}
}
