package section

import (
	"encoding/json"
	"strings"

	"github.com/sai-toolboxs/noti-cli/internal/client"
	nerr "github.com/sai-toolboxs/noti-cli/internal/errors"
)

// HeadingLevel represents the level of a heading block.
type HeadingLevel int

const (
	HeadingLevel1 HeadingLevel = 1
	HeadingLevel2 HeadingLevel = 2
	HeadingLevel3 HeadingLevel = 3
)

// headingLevelFromType converts a block type to a heading level.
func headingLevelFromType(blockType string) (HeadingLevel, bool) {
	switch blockType {
	case "heading_1":
		return HeadingLevel1, true
	case "heading_2":
		return HeadingLevel2, true
	case "heading_3":
		return HeadingLevel3, true
	default:
		return 0, false
	}
}

// headingTypeFromLevel converts a heading level to a block type.
func headingTypeFromLevel(level HeadingLevel) string {
	switch level {
	case HeadingLevel1:
		return "heading_1"
	case HeadingLevel2:
		return "heading_2"
	case HeadingLevel3:
		return "heading_3"
	default:
		return ""
	}
}

// isHeading reports whether the block type is a heading.
func isHeading(blockType string) bool {
	_, ok := headingLevelFromType(blockType)
	return ok
}

// terminatesSection reports whether a new heading terminates the current section.
// A heading terminates a section if it is at the same level or higher.
func terminatesSection(currentLevel HeadingLevel, newBlockType string) bool {
	newLevel, ok := headingLevelFromType(newBlockType)
	if !ok {
		return false
	}
	return newLevel <= currentLevel
}

// Section represents a resolved section in a Notion page.
type Section struct {
	Heading    *client.NotionBlock
	Blocks     []client.NotionBlock
	Level      HeadingLevel
	Path       string
	PageID     string
	FirstBlock string
	LastBlock  string
}

// ParsePath parses a heading path string like "Report / Findings / Evidence".
// Returns the normalized path components.
func ParsePath(path string) []string {
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// NormalizePath normalizes a path for comparison (lowercase, trimmed).
func NormalizePath(path string) string {
	components := ParsePath(path)
	for i, c := range components {
		components[i] = strings.ToLower(strings.TrimSpace(c))
	}
	return strings.Join(components, " / ")
}

// ExtractHeadingText extracts plain text from a heading block's rich_text.
func ExtractHeadingText(b *client.NotionBlock) string {
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

// FindHeadingByBlockID finds a heading block by its exact block ID.
func FindHeadingByBlockID(blocks []client.NotionBlock, blockID string) (*client.NotionBlock, error) {
	for _, b := range blocks {
		if b.ID == blockID {
			if !isHeading(b.Type) {
				return nil, nerr.New(nerr.CodeValidation, "block is not a heading")
			}
			return &b, nil
		}
	}
	return nil, nerr.New(nerr.CodeNotFound, "block not found")
}

// FindHeadingsByText finds all heading blocks with matching text.
func FindHeadingsByText(blocks []client.NotionBlock, text string) []client.NotionBlock {
	var matches []client.NotionBlock
	for _, b := range blocks {
		if !isHeading(b.Type) {
			continue
		}
		headingText := ExtractHeadingText(&b)
		if headingText == text {
			matches = append(matches, b)
		}
	}
	return matches
}

// FindSectionByPath finds a section by its heading path.
func FindSectionByPath(blocks []client.NotionBlock, pathComponents []string) (*client.NotionBlock, error) {
	if len(pathComponents) == 0 {
		return nil, nerr.New(nerr.CodeUsage, "path components required")
	}

	// Find heading_1 blocks matching the first component.
	var candidates []client.NotionBlock
	for _, b := range blocks {
		if b.Type != "heading_1" {
			continue
		}
		text := ExtractHeadingText(&b)
		if strings.EqualFold(text, pathComponents[0]) {
			candidates = append(candidates, b)
		}
	}

	if len(candidates) == 0 {
		return nil, nerr.New(nerr.CodeNotFound, "heading not found for path component: "+pathComponents[0])
	}

	// For each candidate, try to resolve remaining path components.
	var resolved []*client.NotionBlock
	for _, candidate := range candidates {
		result, err := resolvePathUnder(blocks, candidate, pathComponents[1:])
		if err == nil && result != nil {
			resolved = append(resolved, result)
		}
	}

	if len(resolved) == 0 {
		return nil, nerr.New(nerr.CodeNotFound, "path not found")
	}
	if len(resolved) > 1 {
		return nil, nerr.New(nerr.CodeAmbiguousTarget, "ambiguous path: multiple sections match")
	}

	return resolved[0], nil
}

// resolvePathUnder resolves the remaining path components under a parent heading.
func resolvePathUnder(blocks []client.NotionBlock, parent client.NotionBlock, remaining []string) (*client.NotionBlock, error) {
	if len(remaining) == 0 {
		return &parent, nil
	}

	parentLevel, _ := headingLevelFromType(parent.Type)
	parentIdx := -1
	for i, b := range blocks {
		if b.ID == parent.ID {
			parentIdx = i
			break
		}
	}
	if parentIdx < 0 {
		return nil, nerr.New(nerr.CodeInternal, "parent block not found in list")
	}

	// Find the next heading that terminates this parent's section.
	terminatorIdx := len(blocks)
	for i := parentIdx + 1; i < len(blocks); i++ {
		if terminatesSection(parentLevel, blocks[i].Type) {
			terminatorIdx = i
			break
		}
	}

	// Find the next heading matching the first remaining component.
	nextLevel := parentLevel + 1
	if nextLevel > HeadingLevel3 {
		nextLevel = HeadingLevel3
	}
	expectedType := headingTypeFromLevel(nextLevel)

	for i := parentIdx + 1; i < terminatorIdx; i++ {
		if blocks[i].Type != expectedType {
			continue
		}
		text := ExtractHeadingText(&blocks[i])
		if strings.EqualFold(text, remaining[0]) {
			return resolvePathUnder(blocks, blocks[i], remaining[1:])
		}
	}

	return nil, nerr.New(nerr.CodeNotFound, "heading not found for path component: "+remaining[0])
}

// FindSection finds a section starting at the given heading block.
// It returns all blocks in the section (heading + descendants).
func FindSection(blocks []client.NotionBlock, headingBlock client.NotionBlock) Section {
	level, _ := headingLevelFromType(headingBlock.Type)
	var sectionBlocks []client.NotionBlock
	sectionBlocks = append(sectionBlocks, headingBlock)

	headingIdx := -1
	for i, b := range blocks {
		if b.ID == headingBlock.ID {
			headingIdx = i
			break
		}
	}

	if headingIdx < 0 {
		return Section{
			Heading: &headingBlock,
			Blocks:  sectionBlocks,
			Level:   level,
		}
	}

	// Collect descendants until termination.
	for i := headingIdx + 1; i < len(blocks); i++ {
		if terminatesSection(level, blocks[i].Type) {
			break
		}
		sectionBlocks = append(sectionBlocks, blocks[i])
	}

	var firstBlock, lastBlock string
	if len(sectionBlocks) > 0 {
		firstBlock = sectionBlocks[0].ID
		lastBlock = sectionBlocks[len(sectionBlocks)-1].ID
	}

	return Section{
		Heading:    &headingBlock,
		Blocks:     sectionBlocks,
		Level:      level,
		FirstBlock: firstBlock,
		LastBlock:  lastBlock,
	}
}

// FindSectionAfter finds the section that comes after the given section.
// It returns the next heading block (if any) that terminates the current section.
func FindSectionAfter(blocks []client.NotionBlock, section Section) *client.NotionBlock {
	if section.Heading == nil {
		return nil
	}

	headingIdx := -1
	for i, b := range blocks {
		if b.ID == section.Heading.ID {
			headingIdx = i
			break
		}
	}

	if headingIdx < 0 {
		return nil
	}

	// Find the next heading that terminates this section.
	for i := headingIdx + len(section.Blocks); i < len(blocks); i++ {
		if terminatesSection(section.Level, blocks[i].Type) {
			return &blocks[i]
		}
	}

	return nil
}

// ResolveTarget resolves a section target from the given selectors.
// It returns the resolved section and any error.
func ResolveTarget(blocks []client.NotionBlock, pageID, blockID, heading, path string) (*Section, error) {
	// Validate selectors: exactly one is required.
	selectorCount := 0
	if blockID != "" {
		selectorCount++
	}
	if heading != "" {
		selectorCount++
	}
	if path != "" {
		selectorCount++
	}

	if selectorCount == 0 {
		return nil, nerr.New(nerr.CodeUsage, "exactly one selector is required: --block, --heading, or --path")
	}
	if selectorCount > 1 {
		return nil, nerr.New(nerr.CodeUsage, "only one selector is allowed: --block, --heading, or --path")
	}

	// Resolve based on selector type.
	if blockID != "" {
		headingBlock, err := FindHeadingByBlockID(blocks, blockID)
		if err != nil {
			return nil, err
		}
		section := FindSection(blocks, *headingBlock)
		section.PageID = pageID
		return &section, nil
	}

	if heading != "" {
		matches := FindHeadingsByText(blocks, heading)
		if len(matches) == 0 {
			return nil, nerr.New(nerr.CodeNotFound, "heading not found: "+heading)
		}
		if len(matches) > 1 {
			return nil, nerr.New(nerr.CodeAmbiguousTarget, "ambiguous heading: "+heading+" (use --path to disambiguate)")
		}
		section := FindSection(blocks, matches[0])
		section.PageID = pageID
		return &section, nil
	}

	if path != "" {
		pathComponents := ParsePath(path)
		headingBlock, err := FindSectionByPath(blocks, pathComponents)
		if err != nil {
			return nil, err
		}
		section := FindSection(blocks, *headingBlock)
		section.PageID = pageID
		section.Path = path
		return &section, nil
	}

	return nil, nerr.New(nerr.CodeUsage, "exactly one selector is required: --block, --heading, or --path")
}
