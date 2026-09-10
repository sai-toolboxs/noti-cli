package input

import (
	"testing"
)

func TestParseEmpty(t *testing.T) {
	_, err := Parse([]byte{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseJSONArray(t *testing.T) {
	input := `[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Hello"}}]}}]`
	blocks, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "paragraph" {
		t.Fatalf("expected paragraph, got %v", blocks[0]["type"])
	}
}

func TestParseJSONObject(t *testing.T) {
	input := `{"object":"block","type":"heading_1","heading_1":{"rich_text":[{"type":"text","text":{"content":"Title"}}]}}`
	blocks, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "heading_1" {
		t.Fatalf("expected heading_1, got %v", blocks[0]["type"])
	}
}

func TestParseJSONInvalid(t *testing.T) {
	input := `{not valid json`
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseJSONEmptyArray(t *testing.T) {
	input := `[]`
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for empty JSON array")
	}
}

func TestParseJSONWithWhitespace(t *testing.T) {
	input := `  [{"type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Hi"}}]}}]`
	blocks, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
}

func TestParseParagraph(t *testing.T) {
	blocks, err := Parse([]byte("Hello world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "paragraph" {
		t.Fatalf("expected paragraph, got %v", blocks[0]["type"])
	}
}

func TestParseHeading1(t *testing.T) {
	blocks, err := Parse([]byte("# Title"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "heading_1" {
		t.Fatalf("expected heading_1, got %v", blocks[0]["type"])
	}
}

func TestParseHeading2(t *testing.T) {
	blocks, err := Parse([]byte("## Subtitle"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocks[0]["type"] != "heading_2" {
		t.Fatalf("expected heading_2, got %v", blocks[0]["type"])
	}
}

func TestParseHeading3(t *testing.T) {
	blocks, err := Parse([]byte("### Section"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocks[0]["type"] != "heading_3" {
		t.Fatalf("expected heading_3, got %v", blocks[0]["type"])
	}
}

func TestParseHeadingNoSpace(t *testing.T) {
	blocks, err := Parse([]byte("#NoSpace"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocks[0]["type"] != "paragraph" {
		t.Fatalf("#NoSpace should be paragraph, got %v", blocks[0]["type"])
	}
}

func TestParseBulletDash(t *testing.T) {
	blocks, err := Parse([]byte("- Item one"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocks[0]["type"] != "bulleted_list_item" {
		t.Fatalf("expected bulleted_list_item, got %v", blocks[0]["type"])
	}
}

func TestParseBulletStar(t *testing.T) {
	blocks, err := Parse([]byte("* Item two"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocks[0]["type"] != "bulleted_list_item" {
		t.Fatalf("expected bulleted_list_item, got %v", blocks[0]["type"])
	}
}

func TestParseBulletNoSpace(t *testing.T) {
	blocks, err := Parse([]byte("-NoSpace"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocks[0]["type"] != "paragraph" {
		t.Fatalf("-NoSpace should be paragraph, got %v", blocks[0]["type"])
	}
}

func TestParseNumbered(t *testing.T) {
	blocks, err := Parse([]byte("1. First item"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocks[0]["type"] != "numbered_list_item" {
		t.Fatalf("expected numbered_list_item, got %v", blocks[0]["type"])
	}
}

func TestParseQuote(t *testing.T) {
	blocks, err := Parse([]byte("> A wise quote"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocks[0]["type"] != "quote" {
		t.Fatalf("expected quote, got %v", blocks[0]["type"])
	}
}

func TestParseDivider(t *testing.T) {
	blocks, err := Parse([]byte("---"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocks[0]["type"] != "divider" {
		t.Fatalf("expected divider, got %v", blocks[0]["type"])
	}
}

func TestParseCodeBlock(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```"
	blocks, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "code" {
		t.Fatalf("expected code, got %v", blocks[0]["type"])
	}
	code := blocks[0]["code"].(map[string]interface{})
	if code["language"] != "go" {
		t.Fatalf("expected language go, got %v", code["language"])
	}
}

func TestParseCodeBlockNoLang(t *testing.T) {
	input := "```\nsome code\n```"
	blocks, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	code := blocks[0]["code"].(map[string]interface{})
	if code["language"] != "plain text" {
		t.Fatalf("expected plain text language, got %v", code["language"])
	}
}

func TestParseCodeBlockUnclosed(t *testing.T) {
	input := "```go\nfmt.Println()"
	blocks, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
}

func TestParseBlankLinesIgnored(t *testing.T) {
	input := "First\n\n\nSecond"
	blocks, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
}

func TestParseMixed(t *testing.T) {
	input := "# Title\nFirst paragraph.\n- Bullet one\n- Bullet two\n> A quote\n---\nDone."
	blocks, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 7 {
		t.Fatalf("expected 7 blocks, got %d", len(blocks))
	}
	expected := []string{"heading_1", "paragraph", "bulleted_list_item", "bulleted_list_item", "quote", "divider", "paragraph"}
	for i, exp := range expected {
		if blocks[i]["type"] != exp {
			t.Errorf("block %d: expected %s, got %v", i, exp, blocks[i]["type"])
		}
	}
}

func TestParseMultipleParagraphs(t *testing.T) {
	input := "First paragraph.\nSecond paragraph.\nThird paragraph."
	blocks, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	for _, b := range blocks {
		if b["type"] != "paragraph" {
			t.Fatalf("expected paragraph, got %v", b["type"])
		}
	}
}

func TestParseConsecutiveBullets(t *testing.T) {
	input := "- One\n- Two\n- Three"
	blocks, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	for _, b := range blocks {
		if b["type"] != "bulleted_list_item" {
			t.Fatalf("expected bulleted_list_item, got %v", b["type"])
		}
	}
}

func TestParseRichTextContent(t *testing.T) {
	blocks, err := Parse([]byte("Hello world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	para := blocks[0]["paragraph"].(map[string]interface{})
	rt := para["rich_text"].([]map[string]interface{})
	text := rt[0]["text"].(map[string]interface{})
	if text["content"] != "Hello world" {
		t.Fatalf("expected 'Hello world', got %v", text["content"])
	}
}

func TestParseCodeBlockBody(t *testing.T) {
	input := "```python\nprint('hi')\n```"
	blocks, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	code := blocks[0]["code"].(map[string]interface{})
	rt := code["rich_text"].([]map[string]interface{})
	text := rt[0]["text"].(map[string]interface{})
	if text["content"] != "print('hi')" {
		t.Fatalf("expected code body, got %v", text["content"])
	}
}
