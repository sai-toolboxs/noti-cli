package section

import (
	"testing"

	"noti-cli/internal/client"
)

func TestHeadingLevelFromType(t *testing.T) {
	tests := []struct {
		blockType string
		level     HeadingLevel
		ok        bool
	}{
		{"heading_1", HeadingLevel1, true},
		{"heading_2", HeadingLevel2, true},
		{"heading_3", HeadingLevel3, true},
		{"paragraph", 0, false},
		{"", 0, false},
	}

	for _, tt := range tests {
		level, ok := headingLevelFromType(tt.blockType)
		if ok != tt.ok {
			t.Errorf("headingLevelFromType(%q) ok = %v, want %v", tt.blockType, ok, tt.ok)
		}
		if ok && level != tt.level {
			t.Errorf("headingLevelFromType(%q) = %v, want %v", tt.blockType, level, tt.level)
		}
	}
}

func TestTerminatesSection(t *testing.T) {
	tests := []struct {
		currentLevel HeadingLevel
		newBlockType string
		expected     bool
	}{
		{HeadingLevel2, "heading_2", true},
		{HeadingLevel2, "heading_1", true},
		{HeadingLevel2, "heading_3", false},
		{HeadingLevel2, "paragraph", false},
		{HeadingLevel1, "heading_1", true},
		{HeadingLevel1, "heading_2", false},
		{HeadingLevel3, "heading_3", true},
		{HeadingLevel3, "heading_2", true},
		{HeadingLevel3, "heading_1", true},
	}

	for _, tt := range tests {
		result := terminatesSection(tt.currentLevel, tt.newBlockType)
		if result != tt.expected {
			t.Errorf("terminatesSection(%v, %q) = %v, want %v", tt.currentLevel, tt.newBlockType, result, tt.expected)
		}
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"Report / Findings / Evidence", []string{"Report", "Findings", "Evidence"}},
		{"Findings", []string{"Findings"}},
		{"", nil},
		{"  Report  /  Findings  ", []string{"Report", "Findings"}},
		{"Report / / Findings", []string{"Report", "Findings"}},
	}

	for _, tt := range tests {
		result := ParsePath(tt.path)
		if len(result) != len(tt.expected) {
			t.Errorf("ParsePath(%q) = %v, want %v", tt.path, result, tt.expected)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("ParsePath(%q)[%d] = %q, want %q", tt.path, i, result[i], tt.expected[i])
			}
		}
	}
}

func TestExtractHeadingText(t *testing.T) {
	tests := []struct {
		name     string
		block    *client.NotionBlock
		expected string
	}{
		{
			name:     "nil block",
			block:    nil,
			expected: "",
		},
		{
			name:     "nil rich text",
			block:    &client.NotionBlock{},
			expected: "",
		},
		{
			name: "single text item",
			block: &client.NotionBlock{
				RichText: []byte(`[{"plain_text": "Findings"}]`),
			},
			expected: "Findings",
		},
		{
			name: "multiple text items",
			block: &client.NotionBlock{
				RichText: []byte(`[{"plain_text": "Hello "}, {"plain_text": "World"}]`),
			},
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractHeadingText(tt.block)
			if result != tt.expected {
				t.Errorf("ExtractHeadingText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFindHeadingByBlockID(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "block-1", Type: "heading_1"},
		{ID: "block-2", Type: "paragraph"},
		{ID: "block-3", Type: "heading_2"},
	}

	tests := []struct {
		blockID   string
		expectErr bool
		errCode   string
	}{
		{"block-1", false, ""},
		{"block-3", false, ""},
		{"block-2", true, "validation"},
		{"block-4", true, "not_found"},
	}

	for _, tt := range tests {
		result, err := FindHeadingByBlockID(blocks, tt.blockID)
		if tt.expectErr {
			if err == nil {
				t.Errorf("FindHeadingByBlockID(%q) expected error, got nil", tt.blockID)
			}
		} else {
			if err != nil {
				t.Errorf("FindHeadingByBlockID(%q) unexpected error: %v", tt.blockID, err)
			}
			if result == nil {
				t.Errorf("FindHeadingByBlockID(%q) returned nil block", tt.blockID)
			}
		}
	}
}

func TestFindHeadingsByText(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "block-1", Type: "heading_1", RichText: []byte(`[{"plain_text": "Findings"}]`)},
		{ID: "block-2", Type: "paragraph", RichText: []byte(`[{"plain_text": "Some text"}]`)},
		{ID: "block-3", Type: "heading_2", RichText: []byte(`[{"plain_text": "Findings"}]`)},
	}

	matches := FindHeadingsByText(blocks, "Findings")
	if len(matches) != 2 {
		t.Errorf("FindHeadingsByText('Findings') returned %d matches, want 2", len(matches))
	}

	matches = FindHeadingsByText(blocks, "Nonexistent")
	if len(matches) != 0 {
		t.Errorf("FindHeadingsByText('Nonexistent') returned %d matches, want 0", len(matches))
	}
}

func TestFindSection(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "h1", Type: "heading_1", RichText: []byte(`[{"plain_text": "Report"}]`)},
		{ID: "p1", Type: "paragraph", RichText: []byte(`[{"plain_text": "Content A"}]`)},
		{ID: "h2", Type: "heading_2", RichText: []byte(`[{"plain_text": "Findings"}]`)},
		{ID: "p2", Type: "paragraph", RichText: []byte(`[{"plain_text": "Finding A"}]`)},
		{ID: "h3", Type: "heading_2", RichText: []byte(`[{"plain_text": "Decision"}]`)},
		{ID: "p3", Type: "paragraph", RichText: []byte(`[{"plain_text": "Decision text"}]`)},
	}

	// Test finding "Findings" section.
	heading := client.NotionBlock{ID: "h2", Type: "heading_2", RichText: []byte(`[{"plain_text": "Findings"}]`)}
	section := FindSection(blocks, heading)

	if section.Heading == nil {
		t.Fatal("FindSection returned nil heading")
	}
	if section.Heading.ID != "h2" {
		t.Errorf("section heading ID = %q, want %q", section.Heading.ID, "h2")
	}
	if len(section.Blocks) != 2 {
		t.Errorf("section has %d blocks, want 2", len(section.Blocks))
	}
	if section.FirstBlock != "h2" {
		t.Errorf("section FirstBlock = %q, want %q", section.FirstBlock, "h2")
	}
	if section.LastBlock != "p2" {
		t.Errorf("section LastBlock = %q, want %q", section.LastBlock, "p2")
	}
}

func TestFindSectionEmpty(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "h1", Type: "heading_2", RichText: []byte(`[{"plain_text": "Empty"}]`)},
		{ID: "h2", Type: "heading_2", RichText: []byte(`[{"plain_text": "Next"}]`)},
	}

	heading := client.NotionBlock{ID: "h1", Type: "heading_2", RichText: []byte(`[{"plain_text": "Empty"}]`)}
	section := FindSection(blocks, heading)

	if len(section.Blocks) != 1 {
		t.Errorf("empty section has %d blocks, want 1", len(section.Blocks))
	}
}

func TestResolveTargetBlockID(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "h1", Type: "heading_1", RichText: []byte(`[{"plain_text": "Report"}]`)},
		{ID: "p1", Type: "paragraph", RichText: []byte(`[{"plain_text": "Content"}]`)},
	}

	section, err := ResolveTarget(blocks, "page-1", "h1", "", "")
	if err != nil {
		t.Fatalf("ResolveTarget() error: %v", err)
	}
	if section.Heading.ID != "h1" {
		t.Errorf("section heading ID = %q, want %q", section.Heading.ID, "h1")
	}
}

func TestResolveTargetHeading(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "h1", Type: "heading_1", RichText: []byte(`[{"plain_text": "Report"}]`)},
		{ID: "h2", Type: "heading_2", RichText: []byte(`[{"plain_text": "Findings"}]`)},
	}

	section, err := ResolveTarget(blocks, "page-1", "", "Findings", "")
	if err != nil {
		t.Fatalf("ResolveTarget() error: %v", err)
	}
	if section.Heading.ID != "h2" {
		t.Errorf("section heading ID = %q, want %q", section.Heading.ID, "h2")
	}
}

func TestResolveTargetHeadingNotFound(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "h1", Type: "heading_1", RichText: []byte(`[{"plain_text": "Report"}]`)},
	}

	_, err := ResolveTarget(blocks, "page-1", "", "Nonexistent", "")
	if err == nil {
		t.Fatal("ResolveTarget() expected error for nonexistent heading")
	}
}

func TestResolveTargetHeadingAmbiguous(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "h1", Type: "heading_2", RichText: []byte(`[{"plain_text": "Findings"}]`)},
		{ID: "h2", Type: "heading_2", RichText: []byte(`[{"plain_text": "Findings"}]`)},
	}

	_, err := ResolveTarget(blocks, "page-1", "", "Findings", "")
	if err == nil {
		t.Fatal("ResolveTarget() expected error for ambiguous heading")
	}
}

func TestResolveTargetPath(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "h1", Type: "heading_1", RichText: []byte(`[{"plain_text": "Report"}]`)},
		{ID: "h2", Type: "heading_2", RichText: []byte(`[{"plain_text": "Findings"}]`)},
		{ID: "h3", Type: "heading_3", RichText: []byte(`[{"plain_text": "Evidence"}]`)},
	}

	section, err := ResolveTarget(blocks, "page-1", "", "", "Report / Findings / Evidence")
	if err != nil {
		t.Fatalf("ResolveTarget() error: %v", err)
	}
	if section.Heading.ID != "h3" {
		t.Errorf("section heading ID = %q, want %q", section.Heading.ID, "h3")
	}
}

func TestResolveTargetNoSelector(t *testing.T) {
	blocks := []client.NotionBlock{}

	_, err := ResolveTarget(blocks, "page-1", "", "", "")
	if err == nil {
		t.Fatal("ResolveTarget() expected error for no selector")
	}
}

func TestResolveTargetMultipleSelectors(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "h1", Type: "heading_1", RichText: []byte(`[{"plain_text": "Report"}]`)},
	}

	_, err := ResolveTarget(blocks, "page-1", "h1", "Report", "")
	if err == nil {
		t.Fatal("ResolveTarget() expected error for multiple selectors")
	}
}

func TestSectionBoundary(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "h1", Type: "heading_1", RichText: []byte(`[{"plain_text": "Report"}]`)},
		{ID: "p1", Type: "paragraph", RichText: []byte(`[{"plain_text": "Content A"}]`)},
		{ID: "h2a", Type: "heading_2", RichText: []byte(`[{"plain_text": "Findings"}]`)},
		{ID: "p2", Type: "paragraph", RichText: []byte(`[{"plain_text": "Finding A"}]`)},
		{ID: "h3a", Type: "heading_3", RichText: []byte(`[{"plain_text": "Evidence"}]`)},
		{ID: "p3", Type: "paragraph", RichText: []byte(`[{"plain_text": "Evidence A"}]`)},
		{ID: "h2b", Type: "heading_2", RichText: []byte(`[{"plain_text": "Decision"}]`)},
		{ID: "p4", Type: "paragraph", RichText: []byte(`[{"plain_text": "Decision text"}]`)},
	}

	// Test "Findings" section includes nested "Evidence" heading.
	heading := client.NotionBlock{ID: "h2a", Type: "heading_2", RichText: []byte(`[{"plain_text": "Findings"}]`)}
	section := FindSection(blocks, heading)

	if len(section.Blocks) != 4 {
		t.Errorf("Findings section has %d blocks, want 4 (h2a, p2, h3a, p3)", len(section.Blocks))
	}
	if section.Blocks[2].ID != "h3a" {
		t.Errorf("section block[2] ID = %q, want %q", section.Blocks[2].ID, "h3a")
	}
}

func TestSectionHigherLevelTermination(t *testing.T) {
	blocks := []client.NotionBlock{
		{ID: "h1", Type: "heading_1", RichText: []byte(`[{"plain_text": "Report"}]`)},
		{ID: "p1", Type: "paragraph", RichText: []byte(`[{"plain_text": "Content A"}]`)},
		{ID: "h2", Type: "heading_2", RichText: []byte(`[{"plain_text": "Findings"}]`)},
		{ID: "p2", Type: "paragraph", RichText: []byte(`[{"plain_text": "Finding A"}]`)},
	}

	heading := client.NotionBlock{ID: "h1", Type: "heading_1", RichText: []byte(`[{"plain_text": "Report"}]`)}
	section := FindSection(blocks, heading)

	if len(section.Blocks) != 4 {
		t.Errorf("Report section has %d blocks, want 4 (h1, p1, h2, p2)", len(section.Blocks))
	}
}
