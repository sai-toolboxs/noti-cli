package report

import (
	"context"
	"strings"

	"github.com/sai-toolboxs/noti-cli/internal/client"
	nerr "github.com/sai-toolboxs/noti-cli/internal/errors"
	"github.com/sai-toolboxs/noti-cli/internal/output"
)

// Tree renders the full block tree of a page.
func Tree(ctx context.Context, nc *client.NotionClient, pageID string, maxDepth int, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	var tree []*TreeNode
	if maxDepth > 0 {
		tree = BuildTreeWithDepth(blocks, maxDepth)
	} else {
		tree = BuildTree(blocks)
	}

	if out.IsJSON() {
		out.PrintJSON(map[string]interface{}{
			"page_id": pageID,
			"tree":    tree,
			"count":   CountNodes(tree),
		})
	} else {
		printTree(out, tree, 0)
	}
	return nerr.ExitSuccess
}

// printTree renders the tree in human-readable format.
func printTree(out *output.Writer, nodes []*TreeNode, indent int) {
	for _, node := range nodes {
		prefix := strings.Repeat("  ", indent)
		id := node.Block.ID
		if len(id) > 8 {
			id = id[:8]
		}

		switch node.Block.Type {
		case "heading_1":
			out.Printf("%s# %s", prefix, node.Text)
		case "heading_2":
			out.Printf("%s## %s", prefix, node.Text)
		case "heading_3":
			out.Printf("%s### %s", prefix, node.Text)
		case "paragraph":
			if node.Text == "" {
				out.Printf("%s(empty)", prefix)
			} else {
				out.Printf("%s%s", prefix, node.Text)
			}
		case "bulleted_list_item":
			out.Printf("%s• %s", prefix, node.Text)
		case "numbered_list_item":
			out.Printf("%s1. %s", prefix, node.Text)
		case "to_do":
			checked := "[ ]"
			if extractToDoChecked(&node.Block) {
				checked = "[x]"
			}
			out.Printf("%s%s %s", prefix, checked, node.Text)
		case "code":
			lang := extractCodeLanguage(&node.Block)
			out.Printf("%s```%s", prefix, lang)
			out.Printf("%s%s", prefix, node.Text)
			out.Printf("%s```", prefix)
		case "quote":
			out.Printf("%s> %s", prefix, node.Text)
		case "divider":
			out.Printf("%s---", prefix)
		case "child_page":
			out.Printf("%s📄 [child page] %s", prefix, node.Text)
		case "child_database":
			out.Printf("%s🗄 [child database] %s", prefix, node.Text)
		default:
			out.Printf("%s[%s] %s", prefix, node.Block.Type, node.Text)
		}

		if len(node.Children) > 0 {
			printTree(out, node.Children, indent+1)
		}
	}
}

// Outline renders the heading hierarchy of a page.
func Outline(ctx context.Context, nc *client.NotionClient, pageID string, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	entries := BuildOutline(blocks)

	if out.IsJSON() {
		out.PrintJSON(map[string]interface{}{
			"page_id": pageID,
			"outline": entries,
			"count":   len(entries),
		})
	} else {
		if len(entries) == 0 {
			out.Print("(no headings found)")
		} else {
			for _, entry := range entries {
				prefix := strings.Repeat("  ", entry.Level-1)
				out.Printf("%s%s %s", prefix, strings.Repeat("#", entry.Level), entry.Text)
			}
		}
	}
	return nerr.ExitSuccess
}

// StatsCmd renders aggregate statistics for a page.
func StatsCmd(ctx context.Context, nc *client.NotionClient, pageID string, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	// Get page metadata.
	pageInfo, status, err := nc.GetPage(ctx, pageID)
	if err != nil {
		return handleAPIError(err, status, out)
	}

	// Get all blocks.
	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	meta := PageMetadata{
		ID:             pageInfo.ID,
		Title:          pageInfo.Title,
		CreatedTime:    pageInfo.CreatedTime,
		LastEditedTime: pageInfo.LastEditedTime,
		Archived:       pageInfo.Archived,
	}

	stats := ComputeStats(meta, blocks)

	if out.IsJSON() {
		out.PrintJSON(stats)
	} else {
		out.Print("Page Statistics")
		out.Print("================")
		out.Printf("ID:             %s", stats.PageID)
		out.Printf("Title:          %s", stats.Title)
		out.Printf("Created:        %s", stats.CreatedTime)
		out.Printf("Last edited:    %s", stats.LastEditedTime)
		out.Printf("Archived:       %v", stats.Archived)
		out.Print("")
		out.Printf("Total blocks:   %d", stats.TotalBlocks)
		out.Printf("Text length:    %d characters", stats.TextLength)
		out.Printf("Sections:       %d", stats.SectionCount)
		out.Print("")
		out.Print("Block counts by type:")
		for blockType, count := range stats.BlockCounts {
			out.Printf("  %s: %d", blockType, count)
		}
		out.Print("")
		out.Print("Heading counts:")
		for level, count := range stats.HeadingCounts {
			out.Printf("  %s: %d", level, count)
		}
	}
	return nerr.ExitSuccess
}

// Export renders a page as GitHub Flavored Markdown.
func Export(ctx context.Context, nc *client.NotionClient, pageID string, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	// Get page metadata.
	_, status, err := nc.GetPage(ctx, pageID)
	if err != nil {
		return handleAPIError(err, status, out)
	}

	// Get all blocks.
	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	markdown := ExportMarkdown(blocks)

	if out.IsJSON() {
		out.PrintJSON(map[string]interface{}{
			"page_id":  pageID,
			"markdown": markdown,
		})
	} else {
		out.Print(markdown)
	}
	return nerr.ExitSuccess
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
