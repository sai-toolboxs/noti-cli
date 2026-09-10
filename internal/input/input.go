// Package input parses human-friendly text or raw JSON into Notion block arrays.
//
// Auto-detection: input starting with '[' or '{' (after trimming) is parsed as JSON.
// Otherwise, input is parsed as line-prefix text:
//
//	# heading_1
//	## heading_2
//	### heading_3
//	- bullet / * bullet
//	1. numbered
//	> quote
//	--- divider
//	```lang ... ``` code block
//	other text → paragraph
package input

import (
	"encoding/json"
	"strings"

	nerr "github.com/sai-toolboxs/noti-cli/internal/errors"
)

// Parse reads raw input bytes and returns a Notion block array.
// It auto-detects JSON vs text mode.
func Parse(data []byte) ([]map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, nerr.New(nerr.CodeValidation, "empty input")
	}

	trimmed := string(data)

	// Auto-detect JSON: first non-whitespace char is '[' or '{'.
	first := firstNonWhitespace(trimmed)
	if first == '[' || first == '{' {
		return parseJSON(trimmed)
	}

	return parseText(trimmed)
}

// parseJSON parses input as a JSON block array or single JSON object.
func parseJSON(s string) ([]map[string]interface{}, error) {
	// Try array first.
	var arr []map[string]interface{}
	if err := json.Unmarshal([]byte(s), &arr); err == nil {
		if len(arr) == 0 {
			return nil, nerr.New(nerr.CodeValidation, "empty JSON array")
		}
		return arr, nil
	}

	// Try single object.
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return nil, nerr.Wrap(nerr.CodeValidation, "input starts with JSON syntax but is invalid JSON", err)
	}

	return []map[string]interface{}{obj}, nil
}

// parseText parses line-prefix text into Notion block arrays.
func parseText(s string) ([]map[string]interface{}, error) {
	lines := strings.Split(s, "\n")
	var blocks []map[string]interface{}
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Blank line: skip.
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		// Fenced code block.
		if strings.HasPrefix(strings.TrimLeft(line, " "), "```") {
			lang, body, endIdx := parseCodeBlock(lines, i)
			blocks = append(blocks, makeCodeBlock(lang, body))
			i = endIdx
			continue
		}

		// Divider: --- (exactly three dashes, optional trailing whitespace).
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			blocks = append(blocks, makeDivider())
			i++
			continue
		}

		// Heading: # / ## / ### followed by space.
		if strings.HasPrefix(line, "### ") {
			blocks = append(blocks, makeHeading(3, strings.TrimPrefix(line, "### ")))
			i++
			continue
		}
		if strings.HasPrefix(line, "## ") {
			blocks = append(blocks, makeHeading(2, strings.TrimPrefix(line, "## ")))
			i++
			continue
		}
		if strings.HasPrefix(line, "# ") {
			blocks = append(blocks, makeHeading(1, strings.TrimPrefix(line, "# ")))
			i++
			continue
		}

		// Bullet: - or * followed by space.
		if strings.HasPrefix(line, "- ") {
			blocks = append(blocks, makeBullet(strings.TrimPrefix(line, "- ")))
			i++
			continue
		}
		if strings.HasPrefix(line, "* ") {
			blocks = append(blocks, makeBullet(strings.TrimPrefix(line, "* ")))
			i++
			continue
		}

		// Numbered list: digits + ". " (e.g. "1. ").
		if idx := strings.Index(line, ". "); idx > 0 && idx <= 9 {
			prefix := line[:idx+2]
			allDigits := true
			for _, c := range prefix[:len(prefix)-2] {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				blocks = append(blocks, makeNumbered(strings.TrimPrefix(line, prefix)))
				i++
				continue
			}
		}

		// Quote: > followed by space.
		if strings.HasPrefix(line, "> ") {
			blocks = append(blocks, makeQuote(strings.TrimPrefix(line, "> ")))
			i++
			continue
		}

		// Default: paragraph.
		blocks = append(blocks, makeParagraph(line))
		i++
	}

	if len(blocks) == 0 {
		return nil, nerr.New(nerr.CodeValidation, "input produced no blocks")
	}

	return blocks, nil
}

// parseCodeBlock parses a fenced code block starting at lines[idx].
// Returns language, body text, and the index after the closing fence.
func parseCodeBlock(lines []string, idx int) (lang string, body string, nextIdx int) {
	opening := strings.TrimLeft(lines[idx], " ")
	// Extract language after ```.
	lang = strings.TrimPrefix(opening, "```")
	lang = strings.TrimSpace(lang)

	var bodyLines []string
	i := idx + 1
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "```" {
			return lang, strings.Join(bodyLines, "\n"), i + 1
		}
		bodyLines = append(bodyLines, lines[i])
		i++
	}

	// Unclosed fence: treat rest as body.
	return lang, strings.Join(bodyLines, "\n"), len(lines)
}

func makeHeading(level int, text string) map[string]interface{} {
	key := "heading_" + string(rune('0'+level))
	return map[string]interface{}{
		"object": "block",
		"type":   key,
		key: map[string]interface{}{
			"rich_text": richText(text),
		},
	}
}

func makeParagraph(text string) map[string]interface{} {
	return map[string]interface{}{
		"object": "block",
		"type":   "paragraph",
		"paragraph": map[string]interface{}{
			"rich_text": richText(text),
		},
	}
}

func makeBullet(text string) map[string]interface{} {
	return map[string]interface{}{
		"object": "block",
		"type":   "bulleted_list_item",
		"bulleted_list_item": map[string]interface{}{
			"rich_text": richText(text),
		},
	}
}

func makeNumbered(text string) map[string]interface{} {
	return map[string]interface{}{
		"object": "block",
		"type":   "numbered_list_item",
		"numbered_list_item": map[string]interface{}{
			"rich_text": richText(text),
		},
	}
}

func makeQuote(text string) map[string]interface{} {
	return map[string]interface{}{
		"object": "block",
		"type":   "quote",
		"quote": map[string]interface{}{
			"rich_text": richText(text),
		},
	}
}

func makeDivider() map[string]interface{} {
	return map[string]interface{}{
		"object":  "block",
		"type":    "divider",
		"divider": map[string]interface{}{},
	}
}

func makeCodeBlock(lang, body string) map[string]interface{} {
	if lang == "" {
		lang = "plain text"
	}
	return map[string]interface{}{
		"object": "block",
		"type":   "code",
		"code": map[string]interface{}{
			"rich_text": richText(body),
			"language":  lang,
		},
	}
}

func richText(text string) []map[string]interface{} {
	if text == "" {
		return []map[string]interface{}{}
	}
	return []map[string]interface{}{
		{
			"type": "text",
			"text": map[string]interface{}{
				"content": text,
			},
		},
	}
}

func firstNonWhitespace(s string) byte {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' {
			return s[i]
		}
	}
	return 0
}
