# noti-cli

A Termux-first Go CLI for precise, safe Notion content operations without whole-page replacement.

## Problem

The standard Notion workflow involves reading an entire page, modifying it locally, and writing it back. This approach:

- Wastes tokens and bandwidth on large pages
- Risks losing unrelated content during updates
- Provides no atomic safety for targeted edits

**noti-cli** solves this by providing block-aware operations that modify only the specific content you target.

## Features

- **Block operations**: get, list, append, update, delete
- **Page operations**: info, children, read, create
- **Section operations**: find, read, append, insert-after, replace
- **Text operations**: find exact text, replace exact text in a single block
- **Report operations**: tree, outline, stats, export (markdown)
- **JSON output**: machine-readable format for automation
- **Safe mutations**: deterministic target resolution, owning-block-only updates

## Installation

```sh
go install github.com/your-username/noti-cli/cmd/noti-cli@latest
```

Or build from source:

```sh
git clone https://github.com/your-username/noti-cli.git
cd noti-cli
go build -o noti-cli ./cmd/noti-cli
```

## Authentication

Set your Notion API token:

```sh
export NOTION_API_TOKEN="your-integration-token"
```

Or for compatibility:

```sh
export NOTION_TOKEN="your-integration-token"
```

## Usage

### Text Find

Find exact text within a page:

```sh
noti-cli text find --page <page-id> --find "exact text"
```

Output shows matching blocks with their IDs:

```
found 2 match(es) for "exact text":
  [abc12345] paragraph: This paragraph contains exact text here.
  [def67890] paragraph: Another paragraph with exact text.
```

### Text Replace

Replace exact text in a single block:

```sh
noti-cli text replace --page <page-id> --find "old text" --replace "new text" --block <block-id>
```

With expected text guard (prevents race conditions):

```sh
noti-cli text replace --page <page-id> --find "old text" --replace "new text" \
  --expected "Full paragraph text containing old text" \
  --block <block-id>
```

### Section Operations

Find a section by heading:

```sh
noti-cli section find --page <page-id> --heading "Findings"
```

Read section content:

```sh
noti-cli section read --page <page-id> --path "Report / Findings / Evidence"
```

Replace section content:

```sh
echo "New content" | noti-cli section replace --page <page-id> --heading "Findings"
```

### Block Operations

```sh
noti-cli block get <block-id>
noti-cli block list <block-id>
echo '[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Hello"}}]}}]' | noti-cli block append <page-id>
```

### Page Operations

```sh
noti-cli page info <page-id>
noti-cli page children <page-id>
noti-cli page read <page-id>
```

### Report Operations

```sh
noti-cli report tree <page-id>
noti-cli report outline <page-id>
noti-cli report stats <page-id>
noti-cli report export <page-id>
```

## JSON Output

Add `--json` for machine-readable output:

```sh
noti-cli text find --page <page-id> --find "text" --json
```

```json
{
  "find": "text",
  "matches": [
    {
      "block_id": "abc12345",
      "block_type": "paragraph",
      "plain_text": "This paragraph contains text."
    }
  ],
  "total_matches": 1
}
```

## Safety Behavior

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error |
| 3 | Authentication error |
| 4 | Not found |
| 5 | Ambiguous target |
| 6 | Validation error (--expected mismatch) |
| 7 | Conflict |
| 8 | Rate limited |
| 9 | Network timeout |
| 10 | Notion API error |

### Text Replace Safety

- **Not found**: Exit 4, no mutation
- **Ambiguous**: Exit 5, no mutation (multiple blocks match, use `--block` to disambiguate)
- **Expected mismatch**: Exit 6, no mutation (prevents race conditions)
- **Verified**: After replacement, the tool re-reads the block and confirms the change

### Target Resolution

- `--block`: Target a specific block by ID (any block type)
- `--heading`: Target a section by heading text (must be unique)
- `--path`: Target a section by heading path (e.g., "Report / Findings")

## Limitations

- Cross-item rich_text formatting may not be preserved during text replacement
- Section append/insert-after places blocks after the section's last block
- No fuzzy matching or natural language selection
- No automatic retry for non-idempotent mutations

## Verification Status

- 12 Go packages, all tests pass
- Real Notion API integration verified
- Safe mutation behavior confirmed

## License

See [LICENSE](LICENSE) for details.
