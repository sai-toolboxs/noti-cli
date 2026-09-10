package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sai-toolboxs/noti-cli/internal/client"
)

// ExportMarkdown renders a list of blocks as GitHub Flavored Markdown.
func ExportMarkdown(blocks []client.NotionBlock) string {
	var sb strings.Builder

	for _, block := range blocks {
		renderBlock(&sb, &block, 0)
	}

	return sb.String()
}

// renderBlock renders a single block as markdown.
func renderBlock(sb *strings.Builder, b *client.NotionBlock, indent int) {
	if b == nil {
		return
	}

	prefix := strings.Repeat("  ", indent)
	text := extractText(b)

	switch b.Type {
	case "heading_1":
		fmt.Fprintf(sb, "# %s\n\n", text)
	case "heading_2":
		fmt.Fprintf(sb, "## %s\n\n", text)
	case "heading_3":
		fmt.Fprintf(sb, "### %s\n\n", text)
	case "paragraph":
		if text == "" {
			sb.WriteString("\n")
		} else {
			fmt.Fprintf(sb, "%s\n\n", text)
		}
	case "bulleted_list_item":
		fmt.Fprintf(sb, "%s- %s\n", prefix, text)
	case "numbered_list_item":
		fmt.Fprintf(sb, "%s1. %s\n", prefix, text)
	case "to_do":
		checked := " "
		if extractToDoChecked(b) {
			checked = "x"
		}
		fmt.Fprintf(sb, "%s- [%s] %s\n", prefix, checked, text)
	case "toggle":
		fmt.Fprintf(sb, "%s<details>\n%s<summary>%s</summary>\n\n", prefix, prefix, text)
	case "code":
		lang := extractCodeLanguage(b)
		fmt.Fprintf(sb, "%s```%s\n%s\n%s```\n\n", prefix, lang, text, prefix)
	case "quote":
		fmt.Fprintf(sb, "%s> %s\n\n", prefix, text)
	case "divider":
		fmt.Fprintf(sb, "%s---\n\n", prefix)
	case "callout":
		fmt.Fprintf(sb, "%s> %s\n\n", prefix, text)
	case "child_page":
		fmt.Fprintf(sb, "%s📄 **%s**\n\n", prefix, text)
	case "child_database":
		fmt.Fprintf(sb, "%s🗄 **%s**\n\n", prefix, text)
	default:
		if text != "" {
			fmt.Fprintf(sb, "%s%s\n\n", prefix, text)
		}
	}
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
