package page

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"noti-cli/internal/client"
	nerr "noti-cli/internal/errors"
	"noti-cli/internal/output"
)

// Info retrieves page metadata and outputs it.
func Info(ctx context.Context, nc *client.NotionClient, pageID string, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	page, status, err := nc.GetPage(ctx, pageID)
	if err != nil {
		return handleAPIError(err, status, out)
	}

	if out.IsJSON() {
		var raw map[string]interface{}
		json.Unmarshal(page.Raw, &raw)
		out.PrintJSON(raw)
	} else {
		out.Print("Page Information")
		out.Print("=================")
		out.Printf("ID:             %s", page.ID)
		out.Printf("Title:          %s", page.Title)
		out.Printf("Created:        %s", page.CreatedTime)
		out.Printf("Last edited:    %s", page.LastEditedTime)
		out.Printf("Archived:       %v", page.Archived)
		out.Print("")
		out.Print("Properties:")
		printProperties(out, page.Properties)
	}
	return nerr.ExitSuccess
}

// Children retrieves the direct children of a page and outputs them.
func Children(ctx context.Context, nc *client.NotionClient, pageID string, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
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
				id := b.ID
				if len(id) > 8 {
					id = id[:8]
				}
				text := extractText(&b)
				if text != "" {
					out.Printf("  [%s] %s  %s", id, b.Type, text)
				} else {
					out.Printf("  [%s] %s", id, b.Type)
				}
			}
		}
	}
	return nerr.ExitSuccess
}

// Read retrieves the full content of a page (all blocks recursively)
// and outputs a rendered representation.
func Read(ctx context.Context, nc *client.NotionClient, pageID string, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	// Get page metadata.
	page, status, err := nc.GetPage(ctx, pageID)
	if err != nil {
		return handleAPIError(err, status, out)
	}

	// Get all blocks.
	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
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
			"page": map[string]interface{}{
				"id":           page.ID,
				"title":        page.Title,
				"created_time": page.CreatedTime,
				"last_edited":  page.LastEditedTime,
				"archived":     page.Archived,
			},
			"blocks": items,
			"count":  len(blocks),
		})
	} else {
		out.Printf("# %s", page.Title)
		out.Print("")
		for _, b := range blocks {
			printBlock(out, &b, 0)
		}
	}
	return nerr.ExitSuccess
}

// printBlock renders a single block in text format.
func printBlock(out *output.Writer, b *client.NotionBlock, indent int) {
	if b == nil {
		return
	}
	prefix := strings.Repeat("  ", indent)
	text := extractText(b)

	switch b.Type {
	case "heading_1":
		out.Printf("%s# %s", prefix, text)
	case "heading_2":
		out.Printf("%s## %s", prefix, text)
	case "heading_3":
		out.Printf("%s### %s", prefix, text)
	case "paragraph":
		if text == "" {
			out.Printf("%s", prefix)
		} else {
			out.Printf("%s%s", prefix, text)
		}
	case "bulleted_list_item":
		out.Printf("%s• %s", prefix, text)
	case "numbered_list_item":
		out.Printf("%s1. %s", prefix, text)
	case "toggle":
		out.Printf("%s▸ %s", prefix, text)
	case "code":
		lang := ""
		if raw, err := json.Marshal(b.Raw); err == nil {
			var m map[string]interface{}
			json.Unmarshal(raw, &m)
			if content, ok := m[b.Type].(map[string]interface{}); ok {
				if l, ok := content["language"].(string); ok {
					lang = l
				}
			}
		}
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
			out.Printf("%s[%s]", prefix, b.Type)
		}
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

// printProperties prints page properties in a readable format.
func printProperties(out *output.Writer, props map[string]interface{}) {
	for name, prop := range props {
		if name == "title" {
			continue // already shown
		}
		propMap, ok := prop.(map[string]interface{})
		if !ok {
			continue
		}
		propType, _ := propMap["type"].(string)
		value := formatPropertyValue(propMap, propType)
		if value != "" {
			out.Printf("  %s: %s", name, value)
		}
	}
}

// formatPropertyValue formats a property value for display.
func formatPropertyValue(propMap map[string]interface{}, propType string) string {
	switch propType {
	case "rich_text":
		if arr, ok := propMap["rich_text"].([]interface{}); ok {
			return extractRichTextArr(arr)
		}
	case "number":
		if n, ok := propMap["number"]; ok && n != nil {
			return fmt.Sprintf("%v", n)
		}
	case "select":
		if sel, ok := propMap["select"].(map[string]interface{}); ok && sel != nil {
			if name, ok := sel["name"].(string); ok {
				return name
			}
		}
	case "multi_select":
		if arr, ok := propMap["multi_select"].([]interface{}); ok {
			names := make([]string, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if name, ok := m["name"].(string); ok {
						names = append(names, name)
					}
				}
			}
			return strings.Join(names, ", ")
		}
	case "date":
		if d, ok := propMap["date"].(map[string]interface{}); ok && d != nil {
			start, _ := d["start"].(string)
			end, _ := d["end"].(string)
			if end != "" {
				return start + " → " + end
			}
			return start
		}
	case "checkbox":
		if v, ok := propMap["checkbox"].(bool); ok {
			if v {
				return "✓"
			}
			return "✗"
		}
	case "url":
		if v, ok := propMap["url"].(string); ok {
			return v
		}
	case "email":
		if v, ok := propMap["email"].(string); ok {
			return v
		}
	case "status":
		if s, ok := propMap["status"].(map[string]interface{}); ok && s != nil {
			if name, ok := s["name"].(string); ok {
				return name
			}
		}
	case "formula":
		if f, ok := propMap["formula"].(map[string]interface{}); ok && f != nil {
			if v, ok := f["string"].(string); ok {
				return v
			}
			if n, ok := f["number"]; ok && n != nil {
				return fmt.Sprintf("%v", n)
			}
			if b, ok := f["boolean"]; ok {
				return fmt.Sprintf("%v", b)
			}
		}
	case "relation":
		if arr, ok := propMap["relation"].([]interface{}); ok {
			ids := make([]string, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if id, ok := m["id"].(string); ok {
						if len(id) > 8 {
							id = id[:8]
						}
						ids = append(ids, id)
					}
				}
			}
			return strings.Join(ids, ", ")
		}
	}
	return ""
}

// extractRichTextArr extracts text from a rich_text array.
func extractRichTextArr(arr []interface{}) string {
	var sb strings.Builder
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			if plainText, ok := m["plain_text"].(string); ok {
				sb.WriteString(plainText)
			}
		}
	}
	return sb.String()
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

// Create creates a new page under a parent page or database.
// Input: JSON object with parent, properties, and optional children via stdin or --input flag.
func Create(ctx context.Context, nc *client.NotionClient, input []byte, out *output.Writer) int {
	if len(input) == 0 {
		err := nerr.New(nerr.CodeUsage, "page definition is required (provide via stdin or --input)")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(input, &payload); err != nil {
		nerr.PrintError(nerr.Wrap(nerr.CodeValidation, "invalid JSON payload", err), out.IsJSON())
		return nerr.ExitValidation
	}

	// Extract parent.
	parent, ok := payload["parent"].(map[string]interface{})
	if !ok {
		nerr.PrintError(nerr.New(nerr.CodeValidation, "parent is required"), out.IsJSON())
		return nerr.ExitValidation
	}

	// Extract properties.
	properties, ok := payload["properties"].(map[string]interface{})
	if !ok {
		nerr.PrintError(nerr.New(nerr.CodeValidation, "properties is required"), out.IsJSON())
		return nerr.ExitValidation
	}

	// Extract optional children.
	var children []map[string]interface{}
	if c, ok := payload["children"].([]interface{}); ok {
		children = make([]map[string]interface{}, len(c))
		for i, item := range c {
			if m, ok := item.(map[string]interface{}); ok {
				children[i] = m
			}
		}
	}

	page, status, err := nc.CreatePage(ctx, parent, properties, children)
	if err != nil {
		return handleAPIError(err, status, out)
	}

	if out.IsJSON() {
		var raw map[string]interface{}
		json.Unmarshal(page.Raw, &raw)
		out.PrintJSON(raw)
	} else {
		out.Printf("created page %s", page.ID)
		if page.Title != "" {
			out.Printf("  title: %s", page.Title)
		}
	}
	return nerr.ExitSuccess
}
