package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sai-toolboxs/noti-cli/internal/auth"
	"github.com/sai-toolboxs/noti-cli/internal/block"
	"github.com/sai-toolboxs/noti-cli/internal/client"
	"github.com/sai-toolboxs/noti-cli/internal/config"
	nerr "github.com/sai-toolboxs/noti-cli/internal/errors"
	"github.com/sai-toolboxs/noti-cli/internal/output"
	"github.com/sai-toolboxs/noti-cli/internal/page"
	"github.com/sai-toolboxs/noti-cli/internal/report"
	"github.com/sai-toolboxs/noti-cli/internal/section"
	"github.com/sai-toolboxs/noti-cli/internal/version"
)

func main() {
	code := run()
	os.Exit(code)
}

func run() int {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		return nerr.ExitUsage
	}

	// Detect --json global flag before subcommand parsing.
	jsonOutput := false
	var filteredArgs []string
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	args = filteredArgs

	if len(args) == 0 {
		printUsage()
		return nerr.ExitUsage
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "version":
		return runVersion(jsonOutput)
	case "auth":
		return runAuth(ctx, cmdArgs, jsonOutput)
	case "block":
		return runBlock(ctx, cmdArgs, jsonOutput)
	case "page":
		return runPage(ctx, cmdArgs, jsonOutput)
	case "section":
		return runSection(ctx, cmdArgs, jsonOutput)
	case "text":
		return runText(ctx, cmdArgs, jsonOutput)
	case "report":
		return runReport(ctx, cmdArgs, jsonOutput)
	case "--help", "-h", "help":
		printUsage()
		return nerr.ExitSuccess
	case "--version", "-v":
		return runVersion(jsonOutput)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		return nerr.ExitUsage
	}
}

// newClient creates a configured NotionClient from environment.
// Returns nil and prints error if config is invalid.
func newClient(jsonOutput bool) *client.NotionClient {
	cfg, err := config.Load()
	if err != nil {
		nerr.PrintError(err, jsonOutput)
		return nil
	}
	if err := cfg.Validate(); err != nil {
		nerr.PrintError(err, jsonOutput)
		return nil
	}
	httpClient := auth.Client(cfg.Token, cfg.APIVersion, cfg.Timeout)
	return client.New(httpClient, cfg.APIBaseURL, cfg.APIVersion)
}

func runVersion(jsonOutput bool) int {
	out := output.New(output.FormatText)
	if jsonOutput {
		out = output.New(output.FormatJSON)
	}

	if out.IsJSON() {
		out.PrintJSON(map[string]string{
			"version":    version.Short(),
			"commit":     version.Commit,
			"build_date": version.BuildDate,
		})
	} else {
		out.Print(version.String())
	}
	return nerr.ExitSuccess
}

func runAuth(ctx context.Context, args []string, jsonOutput bool) int {
	out := output.New(output.FormatText)
	if jsonOutput {
		out = output.New(output.FormatJSON)
	}

	if len(args) == 0 || args[0] != "check" {
		fmt.Fprintf(os.Stderr, "usage: noti-cli auth check\n")
		return nerr.ExitUsage
	}

	token := os.Getenv("NOTION_API_TOKEN")
	if token == "" {
		token = os.Getenv("NOTION_TOKEN")
	}
	if token == "" {
		err := nerr.New(nerr.CodeAuthentication, "NOTION_API_TOKEN not set")
		nerr.PrintError(err, jsonOutput)
		return nerr.ExitAuthentication
	}

	if len(token) < 10 {
		err := nerr.New(nerr.CodeAuthentication, "NOTION_API_TOKEN appears too short to be valid")
		nerr.PrintError(err, jsonOutput)
		return nerr.ExitAuthentication
	}

	if out.IsJSON() {
		out.PrintJSON(map[string]interface{}{
			"status":  "ok",
			"message": "token configured",
		})
	} else {
		out.Print("auth: token configured")
	}
	return nerr.ExitSuccess
}

func runBlock(ctx context.Context, args []string, jsonOutput bool) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: noti-cli block <get|list|append|update|delete> <block-id>\n")
		return nerr.ExitUsage
	}

	nc := newClient(jsonOutput)
	if nc == nil {
		return nerr.ExitAuthentication
	}
	out := output.New(output.FormatText)
	if jsonOutput {
		out = output.New(output.FormatJSON)
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "get":
		if len(subArgs) == 0 {
			err := nerr.New(nerr.CodeUsage, "block ID is required")
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return block.Get(ctx, nc, subArgs[0], out)
	case "list":
		if len(subArgs) == 0 {
			err := nerr.New(nerr.CodeUsage, "block ID is required")
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return block.List(ctx, nc, subArgs[0], out)
	case "append":
		if len(subArgs) == 0 {
			err := nerr.New(nerr.CodeUsage, "parent block ID is required")
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		input, err := readInput(subArgs[1:])
		if err != nil {
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return block.Append(ctx, nc, subArgs[0], input, out)
	case "update":
		if len(subArgs) == 0 {
			err := nerr.New(nerr.CodeUsage, "block ID is required")
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		input, err := readInput(subArgs[1:])
		if err != nil {
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return block.Update(ctx, nc, subArgs[0], input, out)
	case "delete":
		if len(subArgs) == 0 {
			err := nerr.New(nerr.CodeUsage, "block ID is required")
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return block.Delete(ctx, nc, subArgs[0], out)
	default:
		fmt.Fprintf(os.Stderr, "unknown block subcommand: %s\n\n", sub)
		fmt.Fprintf(os.Stderr, "usage: noti-cli block <get|list|append|update|delete> <block-id>\n")
		return nerr.ExitUsage
	}
}

func runPage(ctx context.Context, args []string, jsonOutput bool) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: noti-cli page <info|children|read|create> [page-id]\n")
		return nerr.ExitUsage
	}

	nc := newClient(jsonOutput)
	if nc == nil {
		return nerr.ExitAuthentication
	}
	out := output.New(output.FormatText)
	if jsonOutput {
		out = output.New(output.FormatJSON)
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "info":
		if len(subArgs) == 0 {
			err := nerr.New(nerr.CodeUsage, "page ID is required")
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return page.Info(ctx, nc, subArgs[0], out)
	case "children":
		if len(subArgs) == 0 {
			err := nerr.New(nerr.CodeUsage, "page ID is required")
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return page.Children(ctx, nc, subArgs[0], out)
	case "read":
		if len(subArgs) == 0 {
			err := nerr.New(nerr.CodeUsage, "page ID is required")
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return page.Read(ctx, nc, subArgs[0], out)
	case "create":
		input, err := readInput(subArgs)
		if err != nil {
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return page.Create(ctx, nc, input, out)
	default:
		fmt.Fprintf(os.Stderr, "unknown page subcommand: %s\n\n", sub)
		fmt.Fprintf(os.Stderr, "usage: noti-cli page <info|children|read|create> [page-id]\n")
		return nerr.ExitUsage
	}
}

func runSection(ctx context.Context, args []string, jsonOutput bool) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: noti-cli section <find|read|append|insert-after|replace> [flags]\n")
		return nerr.ExitUsage
	}

	nc := newClient(jsonOutput)
	if nc == nil {
		return nerr.ExitAuthentication
	}
	out := output.New(output.FormatText)
	if jsonOutput {
		out = output.New(output.FormatJSON)
	}

	sub := args[0]
	subArgs := args[1:]

	// Parse section flags.
	var pageID, blockID, heading, path string
	var remainingArgs []string

	for i := 0; i < len(subArgs); i++ {
		switch subArgs[i] {
		case "--page":
			if i+1 < len(subArgs) {
				pageID = subArgs[i+1]
				i++
			}
		case "--block":
			if i+1 < len(subArgs) {
				blockID = subArgs[i+1]
				i++
			}
		case "--heading":
			if i+1 < len(subArgs) {
				heading = subArgs[i+1]
				i++
			}
		case "--path":
			if i+1 < len(subArgs) {
				path = subArgs[i+1]
				i++
			}
		default:
			remainingArgs = append(remainingArgs, subArgs[i])
		}
	}

	switch sub {
	case "find":
		return section.Find(ctx, nc, pageID, blockID, heading, path, out)
	case "read":
		return section.Read(ctx, nc, pageID, blockID, heading, path, out)
	case "append":
		input, err := readInput(remainingArgs)
		if err != nil {
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return section.Append(ctx, nc, pageID, blockID, heading, path, input, out)
	case "insert-after":
		input, err := readInput(remainingArgs)
		if err != nil {
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return section.InsertAfter(ctx, nc, pageID, blockID, heading, path, input, out)
	case "replace":
		input, err := readInput(remainingArgs)
		if err != nil {
			nerr.PrintError(err, jsonOutput)
			return nerr.ExitUsage
		}
		return section.Replace(ctx, nc, pageID, blockID, heading, path, input, out)
	default:
		fmt.Fprintf(os.Stderr, "unknown section subcommand: %s\n\n", sub)
		fmt.Fprintf(os.Stderr, "usage: noti-cli section <find|read|append|insert-after|replace> [flags]\n")
		return nerr.ExitUsage
	}
}

func runText(ctx context.Context, args []string, jsonOutput bool) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: noti-cli text <find|replace> [flags]\n")
		return nerr.ExitUsage
	}

	nc := newClient(jsonOutput)
	if nc == nil {
		return nerr.ExitAuthentication
	}
	out := output.New(output.FormatText)
	if jsonOutput {
		out = output.New(output.FormatJSON)
	}

	sub := args[0]
	subArgs := args[1:]

	// Parse text flags.
	var pageID, blockID, heading, path, findText, replaceText, expected string

	for i := 0; i < len(subArgs); i++ {
		switch subArgs[i] {
		case "--page":
			if i+1 < len(subArgs) {
				pageID = subArgs[i+1]
				i++
			}
		case "--block":
			if i+1 < len(subArgs) {
				blockID = subArgs[i+1]
				i++
			}
		case "--heading":
			if i+1 < len(subArgs) {
				heading = subArgs[i+1]
				i++
			}
		case "--path":
			if i+1 < len(subArgs) {
				path = subArgs[i+1]
				i++
			}
		case "--find":
			if i+1 < len(subArgs) {
				findText = subArgs[i+1]
				i++
			}
		case "--replace":
			if i+1 < len(subArgs) {
				replaceText = subArgs[i+1]
				i++
			}
		case "--expected":
			if i+1 < len(subArgs) {
				expected = subArgs[i+1]
				i++
			}
		}
	}

	switch sub {
	case "find":
		return section.TextFind(ctx, nc, pageID, blockID, heading, path, findText, out)
	case "replace":
		return section.TextReplace(ctx, nc, pageID, blockID, heading, path, findText, replaceText, expected, out)
	default:
		fmt.Fprintf(os.Stderr, "unknown text subcommand: %s\n\n", sub)
		fmt.Fprintf(os.Stderr, "usage: noti-cli text <find|replace> [flags]\n")
		return nerr.ExitUsage
	}
}

func runReport(ctx context.Context, args []string, jsonOutput bool) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: noti-cli report <tree|outline|stats|export> [page-id] [flags]\n")
		return nerr.ExitUsage
	}

	nc := newClient(jsonOutput)
	if nc == nil {
		return nerr.ExitAuthentication
	}
	out := output.New(output.FormatText)
	if jsonOutput {
		out = output.New(output.FormatJSON)
	}

	sub := args[0]
	subArgs := args[1:]

	// Parse report flags.
	var pageID string
	var maxDepth int

	for i := 0; i < len(subArgs); i++ {
		switch subArgs[i] {
		case "--depth":
			if i+1 < len(subArgs) {
				fmt.Sscanf(subArgs[i+1], "%d", &maxDepth)
				i++
			}
		default:
			if pageID == "" && subArgs[i][0] != '-' {
				pageID = subArgs[i]
			}
		}
	}

	switch sub {
	case "tree":
		return report.Tree(ctx, nc, pageID, maxDepth, out)
	case "outline":
		return report.Outline(ctx, nc, pageID, out)
	case "stats":
		return report.StatsCmd(ctx, nc, pageID, out)
	case "export":
		return report.Export(ctx, nc, pageID, out)
	default:
		fmt.Fprintf(os.Stderr, "unknown report subcommand: %s\n\n", sub)
		fmt.Fprintf(os.Stderr, "usage: noti-cli report <tree|outline|stats|export> [page-id] [flags]\n")
		return nerr.ExitUsage
	}
}

func printUsage() {
	out := output.New(output.FormatText)
	out.Print("noti-cli — efficient, block-aware Notion operations")
	out.Print("")
	out.Print("Usage:")
	out.Print("  noti-cli <command> [flags]")
	out.Print("")
	out.Print("Commands:")
	out.Print("  auth check                  Verify authentication token")
	out.Print("  version                     Show version information")
	out.Print("")
	out.Print("  block get <id>              Retrieve a single block")
	out.Print("  block list <id>             List child blocks of a block")
	out.Print("  block append <id>           Append child blocks to a page/block")
	out.Print("  block update <id>           Update a block's content")
	out.Print("  block delete <id>           Delete (archive) a block")
	out.Print("")
	out.Print("  page info <id>              Show page metadata")
	out.Print("  page children <id>          List direct children of a page")
	out.Print("  page read <id>              Read full page content")
	out.Print("  page create                 Create a new page")
	out.Print("")
	out.Print("  section find                Find a section by heading, path, or block ID")
	out.Print("  section read                Read section content")
	out.Print("  section append              Append blocks to a section")
	out.Print("  section insert-after        Insert blocks after a section")
	out.Print("  section replace             Replace section content")
	out.Print("")
	out.Print("  text find                   Find exact text within a scope")
	out.Print("  text replace                Replace exact text in a single block")
	out.Print("")
	out.Print("  report tree <id>            Show block tree of a page")
	out.Print("  report outline <id>         Show heading hierarchy")
	out.Print("  report stats <id>           Show page statistics")
	out.Print("  report export <id>          Export page as Markdown")
	out.Print("")
	out.Print("Flags:")
	out.Print("  --json                      Output in JSON format")
	out.Print("  --input <file>              Read input from file (for mutations)")
	out.Print("  -h, --help                  Show this help message")
	out.Print("  -v, --version               Show version")
	out.Print("")
	out.Print("Environment:")
	out.Print("  NOTION_API_TOKEN            Notion integration token (required)")
	out.Print("  NOTION_VERSION              Notion API version (default: 2022-06-28)")
	out.Print("  NOTION_API_BASE_URL         API base URL (default: https://api.notion.com)")
	out.Print("  NOTION_TIMEOUT              HTTP timeout in seconds (default: 30)")
	out.Print("")
	out.Print("Input Format:")
	out.Print("  Mutations (block append, section append, section insert-after, section replace) accept:")
	out.Print("    - JSON: array of Notion API block objects (auto-detected)")
	out.Print("    - Text: human-readable line-prefix syntax (auto-detected)")
	out.Print("")
	out.Print("  Text syntax:")
	out.Print("    # heading     ## heading     ### heading")
	out.Print("    - bullet      * bullet       1. numbered")
	out.Print("    > quote       --- divider    ```lang code```")
	out.Print("    other text    → paragraph")
	out.Print("")
	out.Print("  Input sources:")
	out.Print("    - stdin (pipe):  cat content.md | noti-cli section append --page <id> --heading \"X\"")
	out.Print("    - file:          noti-cli section append --page <id> --heading \"X\" --input content.md")
	out.Print("")
	out.Print("Examples:")
	out.Print("  noti-cli auth check")
	out.Print("  noti-cli block get abc123")
	out.Print("  noti-cli block list abc123 --json")
	out.Print("  noti-cli page info abc123")
	out.Print("  noti-cli page read abc123 --json")
	out.Print("")
	out.Print("  # Append text blocks (auto-detected as text)")
	out.Print("  echo '## Findings\nFirst finding here.' | noti-cli section append --page abc123 --heading \"Report\"")
	out.Print("")
	out.Print("  # Append JSON blocks (auto-detected as JSON)")
	out.Print("  echo '[{\"object\":\"block\",\"type\":\"paragraph\",\"paragraph\":{\"rich_text\":[{\"type\":\"text\",\"text\":{\"content\":\"Hello\"}}]}}]' | noti-cli block append abc123")
	out.Print("")
	out.Print("  noti-cli section find --page abc123 --heading \"Findings\"")
	out.Print("  noti-cli section read --page abc123 --path \"Report / Findings / Evidence\"")
	out.Print("  noti-cli section append --page abc123 --heading \"Findings\" --input content.md")
	out.Print("  noti-cli section replace --page abc123 --path \"Report / Findings\" --input content.md")
}

// ParseFlags extracts global flags from args and returns remaining args.
func ParseFlags(args []string) (jsonOutput bool, verbose bool, remaining []string) {
	for _, arg := range args {
		switch {
		case arg == "--json":
			jsonOutput = true
		case arg == "--verbose":
			verbose = true
		case strings.HasPrefix(arg, "--json="):
			// ignore
		default:
			remaining = append(remaining, arg)
		}
	}
	return
}

// readInput reads input from stdin or from a file path argument.
// Returns the input bytes or an error.
func readInput(args []string) ([]byte, error) {
	// Check for --input flag.
	for i, arg := range args {
		if arg == "--input" && i+1 < len(args) {
			return os.ReadFile(args[i+1])
		}
		if strings.HasPrefix(arg, "--input=") {
			path := strings.TrimPrefix(arg, "--input=")
			return os.ReadFile(path)
		}
	}

	// Check if stdin has data available (not a terminal).
	if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		return io.ReadAll(os.Stdin)
	}

	return nil, nil
}
