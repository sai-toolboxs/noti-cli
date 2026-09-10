package section

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sai-toolboxs/noti-cli/internal/client"
	nerr "github.com/sai-toolboxs/noti-cli/internal/errors"
	"github.com/sai-toolboxs/noti-cli/internal/input"
	"github.com/sai-toolboxs/noti-cli/internal/output"
	"github.com/sai-toolboxs/noti-cli/internal/text"
)

// Find resolves a section target and outputs metadata.
func Find(ctx context.Context, nc *client.NotionClient, pageID, blockID, heading, path string, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	section, err := ResolveTarget(blocks, pageID, blockID, heading, path)
	if err != nil {
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitCode(err.(*nerr.Error).Code)
	}

	if out.IsJSON() {
		out.PrintJSON(map[string]interface{}{
			"section": map[string]interface{}{
				"heading": map[string]interface{}{
					"id":    section.Heading.ID,
					"type":  section.Heading.Type,
					"text":  ExtractHeadingText(section.Heading),
					"level": int(section.Level),
				},
				"path":        section.Path,
				"block_count": len(section.Blocks),
				"first_block": section.FirstBlock,
				"last_block":  section.LastBlock,
			},
		})
	} else {
		text := ExtractHeadingText(section.Heading)
		level := int(section.Level)
		headingID := section.Heading.ID
		if len(headingID) > 8 {
			headingID = headingID[:8]
		}
		out.Printf("found section %q (%d blocks)", text, len(section.Blocks))
		out.Printf("  heading: [%s] %s %s", headingID, section.Heading.Type, text)
		out.Printf("  level:   %d", level)
		if section.Path != "" {
			out.Printf("  path:    %s", section.Path)
		}
		out.Printf("  range:   %s to %s", section.FirstBlock, section.LastBlock)
	}
	return nerr.ExitSuccess
}

// Read resolves a section target and outputs its content.
func Read(ctx context.Context, nc *client.NotionClient, pageID, blockID, heading, path string, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	section, err := ResolveTarget(blocks, pageID, blockID, heading, path)
	if err != nil {
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitCode(err.(*nerr.Error).Code)
	}

	if out.IsJSON() {
		items := make([]map[string]interface{}, len(section.Blocks))
		for i, b := range section.Blocks {
			var raw map[string]interface{}
			json.Unmarshal(b.Raw, &raw)
			items[i] = raw
		}
		out.PrintJSON(map[string]interface{}{
			"section": map[string]interface{}{
				"heading": map[string]interface{}{
					"id":    section.Heading.ID,
					"type":  section.Heading.Type,
					"text":  ExtractHeadingText(section.Heading),
					"level": int(section.Level),
				},
				"path":        section.Path,
				"block_count": len(section.Blocks),
			},
			"blocks": items,
		})
	} else {
		for _, b := range section.Blocks {
			printBlock(out, &b, 0)
		}
	}
	return nerr.ExitSuccess
}

// Append resolves a section target and appends new blocks to it.
// Uses position: after_block to insert blocks directly after the section's last block.
func Append(ctx context.Context, nc *client.NotionClient, pageID, blockID, heading, path string, rawInput []byte, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	if len(rawInput) == 0 {
		err := nerr.New(nerr.CodeUsage, "block content is required (provide via stdin or --input)")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	newBlocks, err := input.Parse(rawInput)
	if err != nil {
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitValidation
	}

	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	section, err := ResolveTarget(blocks, pageID, blockID, heading, path)
	if err != nil {
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitCode(err.(*nerr.Error).Code)
	}

	// Use position: after_block to insert after the section's last block.
	afterID := section.LastBlock
	if afterID == "" {
		afterID = section.Heading.ID
	}
	position := &client.BlockPosition{
		Type:         "after_block",
		AfterBlockID: afterID,
	}

	resp, status, err := nc.AppendBlockChildren(ctx, pageID, newBlocks, position)
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
			"section": map[string]interface{}{
				"heading": map[string]interface{}{
					"id":    section.Heading.ID,
					"type":  section.Heading.Type,
					"text":  ExtractHeadingText(section.Heading),
					"level": int(section.Level),
				},
				"path": section.Path,
			},
			"appended": len(items),
			"blocks":   items,
		})
	} else {
		out.Printf("appended %d block(s) to section %q", len(resp.Results), ExtractHeadingText(section.Heading))
	}
	return nerr.ExitSuccess
}

// InsertAfter resolves a section target and inserts new blocks after it.
// Uses position: after_block to insert blocks directly after the section's last block.
func InsertAfter(ctx context.Context, nc *client.NotionClient, pageID, blockID, heading, path string, rawInput []byte, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	if len(rawInput) == 0 {
		err := nerr.New(nerr.CodeUsage, "block content is required (provide via stdin or --input)")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	newBlocks, err := input.Parse(rawInput)
	if err != nil {
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitValidation
	}

	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	section, err := ResolveTarget(blocks, pageID, blockID, heading, path)
	if err != nil {
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitCode(err.(*nerr.Error).Code)
	}

	// Use position: after_block to insert after the section's last block.
	afterID := section.LastBlock
	if afterID == "" {
		afterID = section.Heading.ID
	}
	position := &client.BlockPosition{
		Type:         "after_block",
		AfterBlockID: afterID,
	}

	resp, status, err := nc.AppendBlockChildren(ctx, pageID, newBlocks, position)
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
			"section": map[string]interface{}{
				"heading": map[string]interface{}{
					"id":    section.Heading.ID,
					"type":  section.Heading.Type,
					"text":  ExtractHeadingText(section.Heading),
					"level": int(section.Level),
				},
				"path": section.Path,
			},
			"inserted": len(items),
			"blocks":   items,
		})
	} else {
		out.Printf("inserted %d block(s) after section %q", len(resp.Results), ExtractHeadingText(section.Heading))
	}
	return nerr.ExitSuccess
}

// Replace resolves a section target and replaces its content.
func Replace(ctx context.Context, nc *client.NotionClient, pageID, blockID, heading, path string, rawInput []byte, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	if len(rawInput) == 0 {
		err := nerr.New(nerr.CodeUsage, "block content is required (provide via stdin or --input)")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	newBlocks, err := input.Parse(rawInput)
	if err != nil {
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitValidation
	}

	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	section, err := ResolveTarget(blocks, pageID, blockID, heading, path)
	if err != nil {
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitCode(err.(*nerr.Error).Code)
	}

	// Record original block count for post-validation.
	originalCount := len(section.Blocks)

	// Delete existing section blocks (in reverse order, skipping the heading).
	for i := len(section.Blocks) - 1; i >= 1; i-- {
		block := section.Blocks[i]
		_, err := nc.DeleteBlock(ctx, block.ID)
		if err != nil {
			nerr.PrintError(nerr.Wrap(nerr.CodeNotionAPI, "failed to delete block during replace", err), out.IsJSON())
			return nerr.ExitNotionAPIFailure
		}
	}

	// Insert new blocks after the heading using position: after_block.
	position := &client.BlockPosition{
		Type:         "after_block",
		AfterBlockID: section.Heading.ID,
	}
	resp, status, err := nc.AppendBlockChildren(ctx, pageID, newBlocks, position)
	if err != nil {
		return handleAPIError(err, status, out)
	}

	// Post-validation: fetch section again and verify.
	newBlocksList, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		nerr.PrintError(nerr.Wrap(nerr.CodeNotionAPI, "failed to verify replace", err), out.IsJSON())
		return nerr.ExitNotionAPIFailure
	}

	newSection := FindSection(newBlocksList, *section.Heading)
	newCount := len(newSection.Blocks)

	if out.IsJSON() {
		items := make([]map[string]interface{}, len(resp.Results))
		for i, b := range resp.Results {
			var raw map[string]interface{}
			json.Unmarshal(b.Raw, &raw)
			items[i] = raw
		}
		out.PrintJSON(map[string]interface{}{
			"section": map[string]interface{}{
				"heading": map[string]interface{}{
					"id":    section.Heading.ID,
					"type":  section.Heading.Type,
					"text":  ExtractHeadingText(section.Heading),
					"level": int(section.Level),
				},
				"path": section.Path,
			},
			"replaced": map[string]interface{}{
				"removed": originalCount - 1,
				"added":   len(items),
			},
			"blocks": items,
		})
	} else {
		out.Printf("replaced section %q (%d blocks → %d blocks)", ExtractHeadingText(section.Heading), originalCount, newCount)
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

// TextFind resolves a scope, searches for exact text, outputs matches.
func TextFind(ctx context.Context, nc *client.NotionClient, pageID, blockID, heading, path, findText string, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	if findText == "" {
		err := nerr.New(nerr.CodeUsage, "--find text is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	// Determine which blocks to search.
	searchBlocks := blocks
	if blockID != "" {
		// For text find, --block targets a specific block.
		found := false
		for _, b := range blocks {
			if b.ID == blockID {
				searchBlocks = []client.NotionBlock{b}
				found = true
				break
			}
		}
		if !found {
			err := nerr.New(nerr.CodeNotFound, "block not found: "+blockID)
			nerr.PrintError(err, out.IsJSON())
			return nerr.ExitNotFound
		}
	} else if heading != "" || path != "" {
		section, err := ResolveTarget(blocks, pageID, "", heading, path)
		if err != nil {
			nerr.PrintError(err, out.IsJSON())
			return nerr.ExitCode(err.(*nerr.Error).Code)
		}
		searchBlocks = section.Blocks
	}

	// Search each block for the text.
	type matchResult struct {
		BlockID   string `json:"block_id"`
		BlockType string `json:"block_type"`
		PlainText string `json:"plain_text"`
	}
	var matches []matchResult

	for _, b := range searchBlocks {
		if b.RichText == nil {
			continue
		}
		blockMatches := text.FindExact(b.RichText, findText)
		if len(blockMatches) > 0 {
			plainText := text.ExtractPlainText(b.RichText)
			matches = append(matches, matchResult{
				BlockID:   b.ID,
				BlockType: b.Type,
				PlainText: plainText,
			})
		}
	}

	if out.IsJSON() {
		out.PrintJSON(map[string]interface{}{
			"matches":       matches,
			"total_matches": len(matches),
			"find":          findText,
		})
	} else {
		if len(matches) == 0 {
			out.Printf("text not found: %q", findText)
		} else {
			out.Printf("found %d match(es) for %q:", len(matches), findText)
			for _, m := range matches {
				id := m.BlockID
				if len(id) > 8 {
					id = id[:8]
				}
				out.Printf("  [%s] %s: %s", id, m.BlockType, m.PlainText)
			}
		}
	}

	if len(matches) == 0 {
		return nerr.ExitNotFound
	}
	return nerr.ExitSuccess
}

// TextReplace resolves a scope, finds exact text, validates --expected, replaces in owning block.
func TextReplace(ctx context.Context, nc *client.NotionClient, pageID, blockID, heading, path, findText, replaceText, expected string, out *output.Writer) int {
	if pageID == "" {
		err := nerr.New(nerr.CodeUsage, "page ID is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	if findText == "" {
		err := nerr.New(nerr.CodeUsage, "--find text is required")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitUsage
	}

	if replaceText == "" && findText != "" {
		// Allow empty replacement (deletion).
	}

	blocks, err := nc.ListAllBlockChildren(ctx, pageID)
	if err != nil {
		return handleAPIError(err, 0, out)
	}

	// Determine search scope.
	searchBlocks := blocks
	if blockID != "" {
		// For text replace, --block targets a specific block (not necessarily a heading).
		// Find the block in the full list.
		found := false
		for _, b := range blocks {
			if b.ID == blockID {
				searchBlocks = []client.NotionBlock{b}
				found = true
				break
			}
		}
		if !found {
			err := nerr.New(nerr.CodeNotFound, "block not found: "+blockID)
			nerr.PrintError(err, out.IsJSON())
			return nerr.ExitNotFound
		}
	} else if heading != "" || path != "" {
		section, err := ResolveTarget(blocks, pageID, "", heading, path)
		if err != nil {
			nerr.PrintError(err, out.IsJSON())
			return nerr.ExitCode(err.(*nerr.Error).Code)
		}
		searchBlocks = section.Blocks
	}

	// Find blocks containing the text.
	type candidateBlock struct {
		block     client.NotionBlock
		plainText string
	}
	var candidates []candidateBlock

	for _, b := range searchBlocks {
		if b.RichText == nil {
			continue
		}
		blockMatches := text.FindExact(b.RichText, findText)
		if len(blockMatches) > 0 {
			plainText := text.ExtractPlainText(b.RichText)
			candidates = append(candidates, candidateBlock{block: b, plainText: plainText})
		}
	}

	// Validate: not found.
	if len(candidates) == 0 {
		err := nerr.New(nerr.CodeNotFound, "text not found: "+findText)
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitNotFound
	}

	// Validate: ambiguous (multiple blocks contain the text, no explicit --block).
	if len(candidates) > 1 && blockID == "" {
		err := nerr.New(nerr.CodeAmbiguousTarget, "text found in multiple blocks; use --block to disambiguate")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitAmbiguousTarget
	}

	// Use the first candidate (or the explicit block).
	target := candidates[0]
	if blockID != "" {
		// Find the specific candidate matching the block ID.
		found := false
		for _, c := range candidates {
			if c.block.ID == blockID {
				target = c
				found = true
				break
			}
		}
		if !found {
			// The specified block doesn't contain the text.
			err := nerr.New(nerr.CodeNotFound, "text not found in block "+blockID)
			nerr.PrintError(err, out.IsJSON())
			return nerr.ExitNotFound
		}
	}

	// Validate --expected if supplied.
	if expected != "" {
		if err := text.ValidateExpected(target.plainText, expected); err != nil {
			validationErr := nerr.New(nerr.CodeValidation, "--expected mismatch: "+err.Error())
			nerr.PrintError(validationErr, out.IsJSON())
			return nerr.ExitValidation
		}
	}

	// Read the current block to get the latest rich_text.
	currentBlock, status, err := nc.GetBlock(ctx, target.block.ID)
	if err != nil {
		return handleAPIError(err, status, out)
	}

	// Perform the replacement.
	newRichText, count, err := text.ReplaceText(currentBlock.RichText, findText, replaceText)
	if err != nil {
		nerr.PrintError(nerr.Wrap(nerr.CodeInternal, "failed to replace text", err), out.IsJSON())
		return nerr.ExitGeneral
	}

	if count == 0 {
		err := nerr.New(nerr.CodeNotFound, "text not found during replacement")
		nerr.PrintError(err, out.IsJSON())
		return nerr.ExitNotFound
	}

	// Construct the minimal block update payload.
	var content map[string]interface{}
	if err := json.Unmarshal(newRichText, &content); err != nil {
		// If it's an array (rich_text), wrap it.
		var arr []map[string]interface{}
		if err2 := json.Unmarshal(newRichText, &arr); err2 == nil {
			content = map[string]interface{}{
				"rich_text": arr,
			}
		} else {
			nerr.PrintError(nerr.Wrap(nerr.CodeInternal, "failed to construct update payload", err), out.IsJSON())
			return nerr.ExitGeneral
		}
	}

	// Update the owning block only.
	_, updateStatus, err := nc.UpdateBlock(ctx, target.block.ID, currentBlock.Type, content)
	if err != nil {
		return handleAPIError(err, updateStatus, out)
	}

	// Verify: re-read the block.
	verifiedBlock, _, verifyErr := nc.GetBlock(ctx, target.block.ID)
	verified := false
	if verifyErr == nil {
		verifiedPlain := text.ExtractPlainText(verifiedBlock.RichText)
		verified = strings.Contains(verifiedPlain, replaceText)
	}

	if out.IsJSON() {
		out.PrintJSON(map[string]interface{}{
			"replaced": map[string]interface{}{
				"block_id":    target.block.ID,
				"block_type":  currentBlock.Type,
				"find":        findText,
				"replace":     replaceText,
				"count":       count,
				"expected_ok": expected != "",
				"verified":    verified,
			},
		})
	} else {
		out.Printf("replaced %q with %q in block %s (verified: %v)", findText, replaceText, target.block.ID, verified)
	}

	return nerr.ExitSuccess
}
