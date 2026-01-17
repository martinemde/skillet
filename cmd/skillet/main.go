package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/martinemde/skillet/internal/color"
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
					arg == "-debug" || arg == "--debug" ||
					arg == "-usage" || arg == "--usage" ||
					arg == "-dry-run" || arg == "--dry-run" ||
					arg == "-q" || arg == "--quiet"

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
		debug          = flags.Bool("debug", false, "Show raw stream JSON as it's received")
		showUsage      = flags.Bool("usage", false, "Show token usage statistics")
		dryRun         = flags.Bool("dry-run", false, "Show the command that would be executed without running it")
		quiet          = flags.Bool("q", false, "Quiet mode - suppress all output except errors")
		prompt         = flags.String("prompt", "", "Prompt to pass to Claude (required if no skill provided)")
		model          = flags.String("model", "", "Override model to use (overrides SKILL.md setting)")
		allowedTools   = flags.String("allowed-tools", "", "Override allowed tools (overrides SKILL.md setting)")
		permissionMode = flags.String("permission-mode", "", "Override permission mode (default: acceptEdits)")
		outputFormat   = flags.String("output-format", "", "Override output format (default: stream-json)")
		color          = flags.String("color", "auto", "Control color output (auto, always, never)")
	)
	// Add alias for --quiet
	flags.BoolVar(quiet, "quiet", false, "Quiet mode - suppress all output except errors")

	// Separate flags from positional arguments to support flags in any position
	flagArgs, posArgs := separateFlags(args[1:])

	if err := flags.Parse(flagArgs); err != nil {
		return err
	}

	if *showVersion {
		_, _ = fmt.Fprintf(stdout, "skillet version %s\n", version)
		return nil
	}

	if *showHelp {
		printHelp(stdout, *color)
		return nil
	}

	// Parse skill if provided
	var skill *parser.Skill
	if len(posArgs) > 0 {
		result, err := resolver.Resolve(posArgs[0])
		if err != nil {
			return fmt.Errorf("failed to resolve skill: %w", err)
		}
		if result.IsURL {
			defer func() { _ = os.Remove(result.Path) }()
		}

		if result.BaseURL != "" {
			skill, err = parser.ParseWithBaseDir(result.Path, result.BaseURL)
		} else {
			skill, err = parser.Parse(result.Path)
		}
		if err != nil {
			return fmt.Errorf("failed to parse skill file: %w", err)
		}
	}

	// Require --prompt when no skill is provided
	if skill == nil && *prompt == "" {
		printHelp(stdout, *color)
		return nil
	}

	// Build executor config with resolved values
	config := executor.Config{
		Prompt:         resolvePrompt(*prompt, skill),
		SystemPrompt:   buildSystemPrompt(skill),
		Model:          resolveString(*model, skillModel(skill)),
		AllowedTools:   resolveString(*allowedTools, skillAllowedTools(skill)),
		PermissionMode: *permissionMode,
		OutputFormat:   *outputFormat,
	}

	// Create pipe for output
	pr, pw := io.Pipe()

	// Create executor
	exec := executor.New(config, pw, stderr)

	// Handle dry-run
	if *dryRun {
		_, _ = fmt.Fprintf(stdout, "Would execute:\n%s\n", exec.GetCommand())
		return nil
	}

	// Create formatter
	// In quiet mode, discard all output (only program errors go to stderr)
	output := stdout
	if *quiet {
		output = io.Discard
	}

	// If user explicitly set --output-format, we're in passthrough mode
	form := formatter.New(formatter.Config{
		Output:          output,
		Verbose:         *verbose,
		Debug:           *debug,
		ShowUsage:       *showUsage,
		PassthroughMode: *outputFormat != "",
		SkillName:       skillName(skill),
		Color:           *color,
	})

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

func printHelp(w io.Writer, colorMode string) {
	// Determine if we should use colors
	useColors := color.ShouldUseColors(colorMode)

	// Initialize markdown renderer
	var mdRenderer *glamour.TermRenderer
	var err error
	if useColors {
		mdRenderer, err = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(0),
		)
		if err != nil {
			// Fallback to plain text if renderer fails
			mdRenderer = nil
		}
	} else {
		// No colors
		mdRenderer = nil
	}

	// Helper to render markdown or return plain text
	renderMarkdown := func(text string) string {
		if mdRenderer == nil {
			return text
		}
		rendered, err := mdRenderer.Render(text)
		if err != nil {
			return text
		}
		return strings.TrimSpace(rendered)
	}

	// Styles for help text (with conditional colors)
	titleStyle := lipgloss.NewStyle().Bold(true).MarginBottom(1)
	sectionStyle := lipgloss.NewStyle().Bold(true).MarginTop(1)
	optionStyle := lipgloss.NewStyle()
	codeStyle := lipgloss.NewStyle().Italic(true)
	descStyle := lipgloss.NewStyle()

	if useColors {
		titleStyle = titleStyle.Foreground(lipgloss.Color("6"))     // Cyan
		sectionStyle = sectionStyle.Foreground(lipgloss.Color("3")) // Yellow
		optionStyle = optionStyle.Foreground(lipgloss.Color("2"))   // Green
		codeStyle = codeStyle.Foreground(lipgloss.Color("8"))       // Dim
		descStyle = descStyle.Foreground(lipgloss.Color("7"))       // Light gray
	}

	// Build help content
	title := titleStyle.Render("skillet - Run SKILL.md files with Claude CLI")

	usage := lipgloss.JoinVertical(lipgloss.Left,
		sectionStyle.Render("Usage:"),
		"  skillet [options] <skill-path>",
		"  skillet --prompt <prompt> [options]",
	)

	description := lipgloss.JoinVertical(lipgloss.Left,
		sectionStyle.Render("Description:"),
		descStyle.Render("  Skillet parses SKILL.md files and executes them using the Claude CLI."),
		descStyle.Render("  It reads the frontmatter configuration, interpolates variables, and"),
		descStyle.Render("  invokes Claude with the appropriate arguments in headless mode."),
		"",
		descStyle.Render("  You can also run skillet without a skill by providing --prompt directly."),
		"",
		"  The skill path can be:",
		"  • An exact file path "+codeStyle.Render("(e.g., path/to/SKILL.md)"),
		"  • A directory containing SKILL.md "+codeStyle.Render("(e.g., path/to/skill)"),
		"  • A skill name in .claude/skills/ "+codeStyle.Render("(e.g., skill-name)"),
		"  • A URL to a skill file "+codeStyle.Render("(e.g., https://example.com/skill.md)"),
	)

	options := lipgloss.JoinVertical(lipgloss.Left,
		sectionStyle.Render("Options:"),
		fmt.Sprintf("  %s              Show this help message", optionStyle.Render("--help")),
		fmt.Sprintf("  %s           Show version information", optionStyle.Render("--version")),
		fmt.Sprintf("  %s           Show verbose output including raw JSON stream", optionStyle.Render("--verbose")),
		fmt.Sprintf("  %s             Show raw stream JSON as it's received", optionStyle.Render("--debug")),
		fmt.Sprintf("  %s             Show token usage statistics after execution", optionStyle.Render("--usage")),
		fmt.Sprintf("  %s           Show the command that would be executed without running it", optionStyle.Render("--dry-run")),
		fmt.Sprintf("  %s, %s         Quiet mode - suppress all output except errors", optionStyle.Render("-q"), optionStyle.Render("--quiet")),
		fmt.Sprintf("  %s            Prompt to pass to Claude (required if no skill provided)", optionStyle.Render("--prompt")),
		fmt.Sprintf("  %s             Override model to use (overrides SKILL.md setting)", optionStyle.Render("--model")),
		fmt.Sprintf("  %s     Override allowed tools (overrides SKILL.md setting)", optionStyle.Render("--allowed-tools")),
		fmt.Sprintf("  %s   Override permission mode (default: acceptEdits)", optionStyle.Render("--permission-mode")),
		fmt.Sprintf("  %s     Override output format (default: stream-json)", optionStyle.Render("--output-format")),
		fmt.Sprintf("  %s            Control color output (auto, always, never)", optionStyle.Render("--color")),
	)

	// Render examples with markdown
	examplesBlock := `~~~sh
# Run a skill by exact path
skillet path/to/SKILL.md

# Run a skill by directory (looks for SKILL.md inside)
skillet .claude/skills/skill-name

# Run a skill by name (looks in .claude/skills/<skill-name>/SKILL.md)
skillet skill-name

# Run a skill from a URL
skillet https://raw.githubusercontent.com/user/repo/main/skill.md

# Run with a custom prompt (with skill)
skillet --prompt "Analyze this code" skill-name

# Run with just a prompt (no skill)
skillet --prompt "What is the weather today?"

# Show what command would be executed
skillet --dry-run skill-name

# Show verbose output and usage statistics
skillet --verbose --usage skill-name
~~~`

	examples := lipgloss.JoinVertical(lipgloss.Left,
		sectionStyle.Render("Examples:"),
		renderMarkdown(examplesBlock),
	)

	// Render SKILL.md format example with markdown renderer
	skillFormatExample := `~~~yaml
---
name: skill-name
description: What this skill does and when to use it
allowed-tools: Read,Write,Bash
model: claude-opus-4-5-20251101
---

# Skill Instructions

Your skill instructions go here...
~~~`

	skillFormat := lipgloss.JoinVertical(lipgloss.Left,
		sectionStyle.Render("SKILL.md Format:"),
		"  A SKILL.md file must contain YAML frontmatter followed by markdown content:",
		"",
		renderMarkdown(skillFormatExample),
	)

	footerLinkStyle := lipgloss.NewStyle().Underline(true)
	if useColors {
		footerLinkStyle = footerLinkStyle.Foreground(lipgloss.Color("4"))
	}
	footer := "\nFor more information, see: " + footerLinkStyle.Render("https://agentskills.io")

	// Combine all sections
	help := lipgloss.JoinVertical(lipgloss.Left,
		title,
		usage,
		description,
		options,
		examples,
		skillFormat,
		footer,
	)

	_, _ = fmt.Fprintln(w, help)
}

// Helper functions for skill value extraction

func skillName(s *parser.Skill) string {
	if s == nil {
		return ""
	}
	return s.Name
}

func skillModel(s *parser.Skill) string {
	if s == nil || s.Model == "inherit" {
		return ""
	}
	return s.Model
}

func skillAllowedTools(s *parser.Skill) string {
	if s == nil {
		return ""
	}
	return s.AllowedTools
}

func resolvePrompt(cliPrompt string, s *parser.Skill) string {
	if cliPrompt != "" {
		return cliPrompt
	}
	if s != nil {
		return s.Description
	}
	return ""
}

func resolveString(override, fallback string) string {
	if override != "" {
		return override
	}
	return fallback
}

func buildSystemPrompt(s *parser.Skill) string {
	if s == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", s.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", s.Description))
	if s.Compatibility != "" {
		sb.WriteString(fmt.Sprintf("**Compatibility:** %s\n\n", s.Compatibility))
	}
	sb.WriteString(s.Content)
	return sb.String()
}
