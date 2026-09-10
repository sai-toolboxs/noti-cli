package report

import (
	"strings"

	"github.com/sai-toolboxs/noti-cli/internal/client"
)

// BuildOutline extracts the heading hierarchy from a list of blocks.
// Returns a flat list of outline entries with path information.
func BuildOutline(blocks []client.NotionBlock) []OutlineEntry {
	var entries []OutlineEntry
	var pathStack []string
	var levelStack []int

	for _, block := range blocks {
		if !isHeading(block.Type) {
			continue
		}

		level := headingLevel(block.Type)
		text := extractText(&block)

		// Pop path components until we find a parent with a lower level.
		for len(levelStack) > 0 && levelStack[len(levelStack)-1] >= level {
			pathStack = pathStack[:len(pathStack)-1]
			levelStack = levelStack[:len(levelStack)-1]
		}

		// Build the path.
		path := ""
		if len(pathStack) > 0 {
			path = strings.Join(pathStack, " / ")
		}

		entry := OutlineEntry{
			Level:       level,
			Text:        text,
			BlockID:     block.ID,
			Path:        path,
			HasChildren: block.HasChildren,
		}
		entries = append(entries, entry)

		// Push this heading onto the path stack.
		pathStack = append(pathStack, text)
		levelStack = append(levelStack, level)
	}

	return entries
}

// OutlineByLevel groups outline entries by heading level.
func OutlineByLevel(entries []OutlineEntry) map[int][]OutlineEntry {
	result := make(map[int][]OutlineEntry)
	for _, entry := range entries {
		result[entry.Level] = append(result[entry.Level], entry)
	}
	return result
}

// CountByLevel counts entries at each heading level.
func CountByLevel(entries []OutlineEntry) map[string]int {
	counts := map[string]int{
		"heading_1": 0,
		"heading_2": 0,
		"heading_3": 0,
	}
	for _, entry := range entries {
		switch entry.Level {
		case 1:
			counts["heading_1"]++
		case 2:
			counts["heading_2"]++
		case 3:
			counts["heading_3"]++
		}
	}
	return counts
}
