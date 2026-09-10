package block

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sai-toolboxs/noti-cli/internal/client"
	nerr "github.com/sai-toolboxs/noti-cli/internal/errors"
	"github.com/sai-toolboxs/noti-cli/internal/input"
	"github.com/sai-toolboxs/noti-cli/internal/output"
)

// Get retrieves a single block by ID and outputs it.
func Get(ctx context.Context, nc *client.NotionClient, blockID string, out *output.Writer) int {
	if blockID == "" {
		err := nerr.New(nerr.CodeUsage, "block ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	block, status, err := nc.GetBlock(ctx, blockID)
	if err != nil {
		return handleAPIError(err, status, out)
	}

	if out.IsJSON() {
		// Output the raw JSON from the API.
		var raw map[string]interface{}
		json.Unmarshal(block.Raw, &raw)
		out.PrintJSON(raw)
	} else {
		printBlock(out, block, 0)
	}
	return nerr.ExitSuccess
}

// List retrieves child blocks of a parent and outputs them.
func List(ctx context.Context, nc *client.NotionClient, blockID string, out *output.Writer) int {
	if blockID == "" {
		err := nerr.New(nerr.CodeUsage, "block ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	blocks, err := nc.ListAllBlockChildren(ctx, blockID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	if out.IsJSON() {
		items := make([]map[string]interface{}, len(blocks))
		for i, b := range blocks {
			var raw map[string]interface{}
			json.Unmarshal(b.Raw, &raw)
			items[i] = raw
		}
		out.PrintJSON(map[string]interface{}{
			"object":  "list",
			"results": items,
			"count":   len(blocks),
		})
	} else {
		if len(blocks) == 0 {
			out.Print("(no children)")
		} else {
			out.Printf("%d child block(s):", len(blocks))
			for _, b := range blocks {
				printBlock(out, &b, 1)
			}
		}
	}
	return nerr.ExitSuccess
}

// printBlock renders a block in human-readable text format.
func printBlock(out *output.Writer, b *client.NotionBlock, indent int) {
	if b == nil {
		return
	}
	prefix := strings.Repeat("  ", indent)
	id := b.ID
	if len(id) > 8 {
		id = id[:8]
	}

	// Extract text content from rich_text.
	text := extractText(b)

	switch b.Type {
	case "heading_1", "heading_2", "heading_3":
		level := strings.TrimPrefix(b.Type, "heading_")
		n := int(level[0] - '0')
		if n > 3 {
			n = 3
		}
		dashes := strings.Repeat("#", n)
		out.Printf("%s%s %s", prefix, dashes, text)
	case "paragraph":
		if text == "" {
			out.Printf("%s(empty paragraph)", prefix)
		} else {
			out.Printf("%s%s", prefix, text)
		}
	case "bulleted_list_item":
		out.Printf("%s• %s", prefix, text)
	case "numbered_list_item":
		out.Printf("%s1. %s", prefix, text)
	case "to_do":
		checked := "[ ]"
		if extractToDoChecked(b) {
			checked = "[x]"
		}
		out.Printf("%s%s %s", prefix, checked, text)
	case "toggle":
		out.Printf("%s▸ %s", prefix, text)
	case "code":
		lang := extractCodeLanguage(b)
		out.Printf("%s```%s", prefix, lang)
		out.Printf("%s%s", prefix, text)
		out.Printf("%s```", prefix)
	case "quote":
		out.Printf("%s> %s", prefix, text)
	case "divider":
		out.Printf("%s---", prefix)
	case "callout":
		out.Printf("%s💬 %s", prefix, text)
	case "child_page":
		out.Printf("%s📄 [child page] %s", prefix, text)
	case "child_database":
		out.Printf("%s🗄 [child database] %s", prefix, text)
	default:
		if text != "" {
			out.Printf("%s[%s] %s", prefix, b.Type, text)
		} else {
			out.Printf("%s[%s] (id: %s)", prefix, b.Type, id)
		}
	}

	if b.HasChildren && b.Type != "child_page" && b.Type != "child_database" {
		out.Printf("%s  (has children)", prefix)
	}
}

// extractText extracts plain text from a block's rich_text.
func extractText(b *client.NotionBlock) string {
	if b.RichText == nil {
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

// extractToDoChecked extracts the checked state from a to_do block.
func extractToDoChecked(b *client.NotionBlock) bool {
	var raw map[string]interface{}
	if err := json.Unmarshal(b.Raw, &raw); err != nil {
		return false
	}
	if blockContent, ok := raw[b.Type].(map[string]interface{}); ok {
		if checked, ok := blockContent["checked"].(bool); ok {
			return checked
		}
	}
	return false
}

// extractCodeLanguage extracts the language from a code block.
func extractCodeLanguage(b *client.NotionBlock) string {
	var raw map[string]interface{}
	if err := json.Unmarshal(b.Raw, &raw); err != nil {
		return ""
	}
	if blockContent, ok := raw[b.Type].(map[string]interface{}); ok {
		if lang, ok := blockContent["language"].(string); ok {
			return lang
		}
	}
	return ""
}

// handleAPIError maps API errors to appropriate error output and exit codes.
func handleAPIError(err error, status int, out *output.Writer) int {
	var apiErr *client.APIError
	if nerr.As(err, &apiErr) {
		switch apiErr.Status {
		case 400:
			nerr.PrintError(nerr.Wrap(nerr.CodeValidation, apiErr.Message, err), out.IsJSON())
			return nerr.ExitValidation
		case 404:
			nerr.PrintError(nerr.Wrap(nerr.CodeNotFound, apiErr.Message, err), out.IsJSON())
			return nerr.ExitNotFound
		case 401, 403:
			nerr.PrintError(nerr.Wrap(nerr.CodeAuthentication, apiErr.Message, err), out.IsJSON())
			return nerr.ExitAuthentication
		case 409:
			nerr.PrintError(nerr.Wrap(nerr.CodeConflict, apiErr.Message, err), out.IsJSON())
			return nerr.ExitConflict
		case 429:
			nerr.PrintError(nerr.Wrap(nerr.CodeRateLimit, apiErr.Message, err), out.IsJSON())
			return nerr.ExitRateLimited
		default:
			nerr.PrintError(nerr.Wrap(nerr.CodeNotionAPI, apiErr.Message, err), out.IsJSON())
			return nerr.ExitNotionAPIFailure
		}
	}
	nerr.PrintError(err, out.IsJSON())
	if status == 0 {
		return nerr.ExitNetworkTimeout
	}
	return nerr.ExitNotionAPIFailure
}

// Append appends child blocks to a parent block or page.
// Input: JSON array of block objects or human-readable text via stdin or --input flag.
func Append(ctx context.Context, nc *client.NotionClient, parentID string, rawInput []byte, out *output.Writer) int {
	if parentID == "" {
		err := nerr.New(nerr.CodeUsage, "parent block ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	if len(rawInput) == 0 {
		err := nerr.New(nerr.CodeUsage, "block content is required (provide via stdin or --input)")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	blocks, err := input.Parse(rawInput)
	if err != nil {
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitValidation
	}

	resp, status, err := nc.AppendBlockChildren(ctx, parentID, blocks, nil)
	if err != nil {
		return handleAPIError(err, status, out)
	}

	if out.IsJSON() {
		items := make([]map[string]interface{}, len(resp.Results))
		for i, b := range resp.Results {
			var raw map[string]interface{}
			json.Unmarshal(b.Raw, &raw)
			items[i] = raw
		}
		out.PrintJSON(map[string]interface{}{
			"object":  "list",
			"results": items,
			"count":   len(items),
		})
	} else {
		out.Printf("appended %d block(s)", len(resp.Results))
	}
	return nerr.ExitSuccess
}

// Update updates a single block's content.
// Input: JSON object with the block type and content via stdin or --input flag.
func Update(ctx context.Context, nc *client.NotionClient, blockID string, input []byte, out *output.Writer) int {
	if blockID == "" {
		err := nerr.New(nerr.CodeUsage, "block ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	if len(input) == 0 {
		err := nerr.New(nerr.CodeUsage, "update payload is required (provide via stdin or --input)")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(input, &payload); err != nil {
		nerr.PrintError(nerr.Wrap(nerr.CodeValidation, "invalid JSON payload", err), out.IsJSON())
		return nerr.ExitValidation
	}

	// Extract block type and content from payload.
	// Expected format: {"type": "paragraph", "content": {"rich_text": [...]}}
	// Or simplified: {"paragraph": {"rich_text": [...]}}
	var blockType string
	var content map[string]interface{}

	if t, ok := payload["type"].(string); ok {
		blockType = t
		if c, ok := payload["content"].(map[string]interface{}); ok {
			content = c
		}
	} else {
		// Try to detect type from the payload keys.
		for _, t := range []string{"paragraph", "heading_1", "heading_2", "heading_3",
			"bulleted_list_item", "numbered_list_item", "to_do", "toggle",
			"code", "quote", "divider", "callout"} {
			if c, ok := payload[t].(map[string]interface{}); ok {
				blockType = t
				content = c
				break
			}
		}
	}

	if blockType == "" {
		nerr.PrintError(nerr.New(nerr.CodeValidation, "block type is required"), out.IsJSON())
		return nerr.ExitValidation
	}

	if content == nil {
		content = make(map[string]interface{})
	}

	block, status, err := nc.UpdateBlock(ctx, blockID, blockType, content)
	if err != nil {
		return handleAPIError(err, status, out)
	}

	if out.IsJSON() {
		var raw map[string]interface{}
		json.Unmarshal(block.Raw, &raw)
		out.PrintJSON(raw)
	} else {
		out.Printf("updated block %s", block.ID)
	}
	return nerr.ExitSuccess
}

// Delete deletes (archives) a single block.
func Delete(ctx context.Context, nc *client.NotionClient, blockID string, out *output.Writer) int {
	if blockID == "" {
		err := nerr.New(nerr.CodeUsage, "block ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	status, err := nc.DeleteBlock(ctx, blockID)
	if err != nil {
		return handleAPIError(err, status, out)
	}

	if out.IsJSON() {
		out.PrintJSON(map[string]interface{}{
			"deleted": true,
			"id":      blockID,
		})
	} else {
		out.Printf("deleted block %s", blockID)
	}
	return nerr.ExitSuccess
}
