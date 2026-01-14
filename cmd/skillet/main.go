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
)

const version = "0.1.0"

func main() {
	if err := run(os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	flags.SetOutput(stderr)

	var (
		showVersion = flags.Bool("version", false, "Show version information")
		showHelp    = flags.Bool("help", false, "Show help information")
		verbose     = flags.Bool("verbose", false, "Show verbose output including raw JSON")
		showUsage   = flags.Bool("usage", false, "Show token usage statistics")
		dryRun      = flags.Bool("dry-run", false, "Show the command that would be executed without running it")
		prompt      = flags.String("prompt", "", "Optional prompt to pass to Claude (if not provided, uses skill description)")
	)

	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	if *showVersion {
		fmt.Fprintf(stdout, "skillet version %s\n", version)
		return nil
	}

	if *showHelp || flags.NArg() == 0 {
		printHelp(stdout)
		return nil
	}

	skillPath := flags.Arg(0)

	// Parse the SKILL.md file
	skill, err := parser.Parse(skillPath)
	if err != nil {
		return fmt.Errorf("failed to parse skill file: %w", err)
	}

	// Create executor
	exec := executor.New(skill, *prompt)

	// If dry-run, just print the command and exit
	if *dryRun {
		fmt.Fprintf(stdout, "Would execute:\n%s\n", exec.GetCommand())
		return nil
	}

	// Create a pipe to capture Claude's output
	pr, pw := io.Pipe()

	// Set executor output
	exec.SetOutput(pw, stderr)

	// Create formatter
	form := formatter.New(stdout, *verbose, *showUsage)

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
		pw.Close() // Close the writer when execution is done
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
	fmt.Fprintf(w, `skillet - Run SKILL.md files with Claude CLI

Usage:
  skillet [options] <path-to-SKILL.md>

Description:
  Skillet parses SKILL.md files and executes them using the Claude CLI.
  It reads the frontmatter configuration, interpolates variables, and
  invokes Claude with the appropriate arguments in headless mode.

Options:
  --help          Show this help message
  --version       Show version information
  --verbose       Show verbose output including raw JSON stream
  --usage         Show token usage statistics after execution
  --dry-run       Show the command that would be executed without running it
  --prompt        Optional prompt to pass to Claude (default: uses skill description)

Examples:
  # Run a skill file
  skillet path/to/SKILL.md

  # Run with a custom prompt
  skillet --prompt "Analyze this code" path/to/SKILL.md

  # Show what command would be executed
  skillet --dry-run path/to/SKILL.md

  # Show verbose output and usage statistics
  skillet --verbose --usage path/to/SKILL.md

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
