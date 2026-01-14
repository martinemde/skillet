package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/martinemde/skillet/internal/executor"
	"github.com/martinemde/skillet/internal/formatter"
	"github.com/martinemde/skillet/internal/parser"
	"github.com/martinemde/skillet/internal/resolver"
)

const version = "0.1.0"

func main() {
	if err := run(os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// separateFlags separates flag arguments from positional arguments.
// This allows flags to appear anywhere in the argument list, not just before positional args.
// Returns (flagArgs, positionalArgs).
func separateFlags(args []string) ([]string, []string) {
	var flagArgs []string
	var posArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if this is a flag (starts with -)
		if len(arg) > 0 && arg[0] == '-' {
			flagArgs = append(flagArgs, arg)

			// Check if this flag takes a value
			// Flags with = are handled by flag package (e.g., --prompt=value)
			// Flags without = may have their value as the next argument
			hasEquals := false
			for _, c := range arg {
				if c == '=' {
					hasEquals = true
					break
				}
			}

			// If the flag doesn't contain =, and there's a next arg that doesn't start with -,
			// it might be the flag's value. We include it with the flags.
			if !hasEquals && i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				// Check if this is a boolean flag (these don't take values)
				isBoolFlag := arg == "-version" || arg == "--version" ||
					arg == "-help" || arg == "--help" ||
					arg == "-verbose" || arg == "--verbose" ||
					arg == "-usage" || arg == "--usage" ||
					arg == "-dry-run" || arg == "--dry-run"

				if !isBoolFlag {
					// This is likely a flag that takes a value, so include the next arg
					i++
					flagArgs = append(flagArgs, args[i])
				}
			}
		} else {
			// This is a positional argument
			posArgs = append(posArgs, arg)
		}
	}

	return flagArgs, posArgs
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	flags.SetOutput(stderr)

	var (
		showVersion    = flags.Bool("version", false, "Show version information")
		showHelp       = flags.Bool("help", false, "Show help information")
		verbose        = flags.Bool("verbose", false, "Show verbose output including raw JSON")
		showUsage      = flags.Bool("usage", false, "Show token usage statistics")
		dryRun         = flags.Bool("dry-run", false, "Show the command that would be executed without running it")
		prompt         = flags.String("prompt", "", "Optional prompt to pass to Claude (if not provided, uses skill description)")
		model          = flags.String("model", "", "Override model to use (overrides SKILL.md setting)")
		allowedTools   = flags.String("allowed-tools", "", "Override allowed tools (overrides SKILL.md setting)")
		permissionMode = flags.String("permission-mode", "", "Override permission mode (default: acceptEdits)")
		outputFormat   = flags.String("output-format", "", "Override output format (default: stream-json)")
	)

	// Separate flags from positional arguments to support flags in any position
	flagArgs, posArgs := separateFlags(args[1:])

	if err := flags.Parse(flagArgs); err != nil {
		return err
	}

	if *showVersion {
		_, _ = fmt.Fprintf(stdout, "skillet version %s\n", version)
		return nil
	}

	if *showHelp || len(posArgs) == 0 {
		printHelp(stdout)
		return nil
	}

	skillPath := posArgs[0]

	// Resolve the skill path (handles files, directories, .claude/skills shortcuts, and URLs)
	result, err := resolver.Resolve(skillPath)
	if err != nil {
		return fmt.Errorf("failed to resolve skill: %w", err)
	}

	// Clean up temporary file if it was downloaded from a URL
	if result.IsURL {
		defer func() {
			_ = os.Remove(result.Path)
		}()
	}

	// Parse the SKILL.md file
	var skill *parser.Skill
	if result.BaseURL != "" {
		// For URL-based skills, use the base URL as the base directory
		skill, err = parser.ParseWithBaseDir(result.Path, result.BaseURL)
	} else {
		// For local files, use the default base directory (file's directory)
		skill, err = parser.Parse(result.Path)
	}
	if err != nil {
		return fmt.Errorf("failed to parse skill file: %w", err)
	}

	// Create executor with CLI overrides
	exec := executor.New(skill, *prompt)
	exec.SetOverrides(*model, *allowedTools, *permissionMode, *outputFormat)

	// If dry-run, just print the command and exit
	if *dryRun {
		_, _ = fmt.Fprintf(stdout, "Would execute:\n%s\n", exec.GetCommand())
		return nil
	}

	// Create a pipe to capture Claude's output
	pr, pw := io.Pipe()

	// Set executor output
	exec.SetOutput(pw, stderr)

	// Create formatter
	// If user explicitly set --output-format, we're in passthrough mode
	passthroughMode := *outputFormat != ""
	form := formatter.New(stdout, *verbose, *showUsage, passthroughMode)

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Run executor and formatter concurrently
	errChan := make(chan error, 2)

	// Start formatter
	go func() {
		errChan <- form.Format(pr)
	}()

	// Run executor
	go func() {
		err := exec.Execute(ctx)
		_ = pw.Close() // Close the writer when execution is done
		errChan <- err
	}()

	// Wait for both to complete
	var execErr, formatErr error
	for i := 0; i < 2; i++ {
		err := <-errChan
		if err != nil {
			if execErr == nil {
				execErr = err
			} else if formatErr == nil {
				formatErr = err
			}
		}
	}

	if execErr != nil {
		return fmt.Errorf("execution failed: %w", execErr)
	}
	if formatErr != nil {
		return fmt.Errorf("formatting failed: %w", formatErr)
	}

	return nil
}

func printHelp(w io.Writer) {
	_, _ = fmt.Fprintf(w, `skillet - Run SKILL.md files with Claude CLI

Usage:
  skillet [options] <skill-path>

Description:
  Skillet parses SKILL.md files and executes them using the Claude CLI.
  It reads the frontmatter configuration, interpolates variables, and
  invokes Claude with the appropriate arguments in headless mode.

  The skill path can be:
  - An exact file path (e.g., path/to/SKILL.md)
  - A directory containing SKILL.md (e.g., path/to/skill)
  - A skill name in .claude/skills/ (e.g., write-skill)
  - A URL to a skill file (e.g., https://example.com/skill.md)

Options:
  --help              Show this help message
  --version           Show version information
  --verbose           Show verbose output including raw JSON stream
  --usage             Show token usage statistics after execution
  --dry-run           Show the command that would be executed without running it
  --prompt            Optional prompt to pass to Claude (default: uses skill description)
  --model             Override model to use (overrides SKILL.md setting)
  --allowed-tools     Override allowed tools (overrides SKILL.md setting)
  --permission-mode   Override permission mode (default: acceptEdits)
  --output-format     Override output format (default: stream-json)

Examples:
  # Run a skill by exact path
  skillet path/to/SKILL.md

  # Run a skill by directory (looks for SKILL.md inside)
  skillet .claude/skills/write-skill

  # Run a skill by name (looks in .claude/skills/<name>/SKILL.md)
  skillet write-skill

  # Run a skill from a URL
  skillet https://raw.githubusercontent.com/user/repo/main/skill.md

  # Run with a custom prompt
  skillet --prompt "Analyze this code" write-skill

  # Show what command would be executed
  skillet --dry-run write-skill

  # Show verbose output and usage statistics
  skillet --verbose --usage write-skill

SKILL.md Format:
  A SKILL.md file must contain YAML frontmatter followed by markdown content:

  ---
  name: skill-name
  description: What this skill does and when to use it
  allowed-tools: Read,Write,Bash
  model: claude-opus-4-5-20251101
  ---

  # Skill Instructions

  Your skill instructions go here...

For more information, see: https://agentskills.io
`)
}
