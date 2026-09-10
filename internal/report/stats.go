package report

import (
	"github.com/sai-toolboxs/noti-cli/internal/client"
)

// PageMetadata contains page metadata needed for stats.
type PageMetadata struct {
	ID             string
	Title          string
	CreatedTime    string
	LastEditedTime string
	Archived       bool
}

// ComputeStats calculates aggregate statistics for a page.
func ComputeStats(meta PageMetadata, blocks []client.NotionBlock) Stats {
	stats := Stats{
		PageID:         meta.ID,
		Title:          meta.Title,
		CreatedTime:    meta.CreatedTime,
		LastEditedTime: meta.LastEditedTime,
		Archived:       meta.Archived,
		BlockCounts:    make(map[string]int),
		HeadingCounts:  make(map[string]int),
		TotalBlocks:    len(blocks),
	}

	for _, block := range blocks {
		// Count by block type.
		stats.BlockCounts[block.Type]++

		// Count text length.
		text := extractText(&block)
		stats.TextLength += len(text)

		// Count headings and sections.
		if isHeading(block.Type) {
			level := headingLevel(block.Type)
			key := "heading_" + string(rune('0'+level))
			stats.HeadingCounts[key]++
			if level == 1 {
				stats.SectionCount++
			}
		}
	}

	return stats
}
