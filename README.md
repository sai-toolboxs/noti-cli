# noti-cli

A cross-platform Go CLI for precise, safe Notion content operations without whole-page replacement.

## Why noti-cli?

A common Notion automation workflow is:

1. Read an entire page
2. Modify the content locally
3. Write the page back

For large pages, this can waste tokens and bandwidth and can put unrelated content at risk.

noti-cli works at the block and section level instead. It resolves a specific target and mutates only the content that was requested.

This makes it useful for automation, scripting, and AI-assisted workflows where predictable and limited changes matter.

## Features

- **Block operations** — get, list, append, update, delete
- **Page operations** — info, children, read, create
- **Section operations** — find, read, append, insert-after, replace
- **Text operations** — find exact text and replace exact text within a block
- **Report operations** — tree, outline, stats, export
- **JSON output** — machine-readable output for automation
- **Safe mutations** — deterministic target resolution and owning-block-only updates
- **Expected-value guard** — verify the current content before applying a replacement

## Installation

Install with Go

Requires Go:

```sh
go install github.com/sai-toolboxs/noti-cli/cmd/noti-cli@latest
```

Build from source

```sh
git clone https://github.com/sai-toolboxs/noti-cli.git
cd noti-cli
go build -o noti-cli ./cmd/noti-cli
```

## Authentication

Create a Notion integration and make sure the integration has access to the pages you want to operate on.

Set your Notion API token:

```sh
export NOTION_API_TOKEN="your-integration-token"
```

For compatibility, `NOTION_TOKEN` is also supported:

```sh
export NOTION_TOKEN="your-integration-token"
```

## Usage

### Text Find

Find exact text within a page:

```sh
noti-cli text find \
  --page <page-id> \
  --find "exact text"
```

The command reports matching blocks and their IDs.

Example:

```
found 2 match(es) for "exact text":
  [abc12345] paragraph: This paragraph contains exact text here.
  [def67890] paragraph: Another paragraph with exact text.
```

### Text Replace

Replace exact text in a specific block:

```sh
noti-cli text replace \
  --page <page-id> \
  --find "old text" \
  --replace "new text" \
  --block <block-id>
```

You can provide `--expected` to verify that the block still contains the expected content before the mutation:

```sh
noti-cli text replace \
  --page <page-id> \
  --find "old text" \
  --replace "new text" \
  --expected "Full paragraph text containing old text" \
  --block <block-id>
```

After a successful replacement, `noti-cli` re-reads the target block and verifies the resulting content.

### Section Operations

Find a section by heading:

```sh
noti-cli section find \
  --page <page-id> \
  --heading "Findings"
```

Read a section:

```sh
noti-cli section read \
  --page <page-id> \
  --path "Report / Findings / Evidence"
```

Replace a section:

```sh
echo "New content" | \
  noti-cli section replace \
  --page <page-id> \
  --heading "Findings"
```

Section targeting supports heading-based and path-based resolution.

### Block Operations

Get a block:

```sh
noti-cli block get <block-id>
```

List child blocks:

```sh
noti-cli block list <block-id>
```

Append blocks from JSON:

```sh
echo '[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Hello"}}]}}]' |
  noti-cli block append <page-id>
```

### Page Operations

Get page information:

```sh
noti-cli page info <page-id>
```

List page children:

```sh
noti-cli page children <page-id>
```

Read a page:

```sh
noti-cli page read <page-id>
```

### Report Operations

Generate a block tree:

```sh
noti-cli report tree <page-id>
```

Generate an outline:

```sh
noti-cli report outline <page-id>
```

Show report statistics:

```sh
noti-cli report stats <page-id>
```

Export a report as Markdown:

```sh
noti-cli report export <page-id>
```

## JSON Output

Use `--json` when the output will be consumed by another program or script:

```sh
noti-cli text find \
  --page <page-id> \
  --find "text" \
  --json
```

Example:

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

`noti-cli` is designed to fail safely when a requested mutation cannot be resolved deterministically.

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error |
| 3 | Authentication error |
| 4 | Not found |
| 5 | Ambiguous target |
| 6 | Validation error (`--expected` mismatch) |
| 7 | Conflict |
| 8 | Rate limited |
| 9 | Network timeout |
| 10 | Notion API error |

### Text Replacement Safety

**Not found**

If the requested text cannot be found, the command exits with code `4` and does not mutate the page.

**Ambiguous target**

If multiple blocks match and the target cannot be resolved uniquely, the command exits with code `5` without mutating the page. Use `--block` to explicitly identify the target.

**Expected-value mismatch**

If `--expected` does not match the current block content, the command exits with code `6` without mutating the page.

This provides a guard against applying a replacement to content that has changed since it was inspected.

**Post-mutation verification**

After a successful text replacement, the target block is read again and the resulting content is verified.

### Target Resolution

The mutation commands support explicit targeting:

- `--block` — target a specific block by ID
- `--heading` — target a section by heading
- `--path` — target a section by heading path, such as "Report / Findings"

The goal is deterministic targeting rather than fuzzy or natural-language selection.

## Limitations

- Text replacement is exact; there is no fuzzy or natural-language matching.
- Cross-item `rich_text` formatting may not be preserved during text replacement.
- Section append and insert-after operations place blocks relative to the resolved section boundary.
- Non-idempotent mutations do not have automatic retry behavior.
- `noti-cli` does not replace an entire page as part of a targeted text mutation.

## Verification

The project has been validated against the real Notion API and its test suite.

Current verification includes:

- Go formatting and vet checks
- Full Go test suite
- Build verification
- Real Notion API text search
- Exact text replacement
- Not-found safe failure
- Ambiguous-target safe failure
- `--expected` validation
- Post-replacement verification
- Preservation of unrelated page content

## Project Status

`noti-cli` has completed its core MVP goal: providing precise, block-aware Notion content operations suitable for automation and AI-assisted workflows.

The project is intentionally focused on safe, deterministic content operations rather than becoming a general-purpose Notion client.

## Project Origin

`noti-cli` was originally created to solve a personal workflow problem: making precise and safe edits to Notion content from automation and AI-assisted development workflows.

It is intentionally focused rather than a full-featured Notion client. The goal is to make targeted content operations predictable, safe, and easy to automate.

If you have a similar workflow, you may find `noti-cli` useful. Contributions and improvements are welcome, but the project does not aim to cover every Notion API feature.

## License

MIT License. See [LICENSE](https://github.com/sai-toolboxs/noti-cli/blob/main/LICENSE) for details.
