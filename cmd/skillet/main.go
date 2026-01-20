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
	"github.com/martinemde/skillet/internal/command"
	"github.com/martinemde/skillet/internal/commandpath"
	"github.com/martinemde/skillet/internal/discovery"
	"github.com/martinemde/skillet/internal/executor"
	"github.com/martinemde/skillet/internal/formatter"
	"github.com/martinemde/skillet/internal/parser"
	"github.com/martinemde/skillet/internal/resolver"
	"github.com/martinemde/skillet/internal/skillpath"
)

const version = "0.1.0"

// boolFlags contains all flags that don't take a value
var boolFlags = map[string]bool{
	"-version":  true,
	"--version": true,
	"-help":     true,
	"--help":    true,
	"-list":     true,
	"--list":    true,
	"-verbose":  true,
	"--verbose": true,
	"-debug":    true,
	"--debug":   true,
	"-usage":    true,
	"--usage":   true,
	"-dry-run":  true,
	"--dry-run": true,
	"-q":        true,
	"--quiet":   true,
}

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
				if !boolFlags[arg] {
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
		listSkills     = flags.Bool("list", false, "List all available skills and commands")
		verbose        = flags.Bool("verbose", false, "Show detailed output including thinking and tool details")
		debug          = flags.Bool("debug", false, "Print raw JSON stream to stderr")
		showUsage      = flags.Bool("usage", false, "Show token usage statistics")
		dryRun         = flags.Bool("dry-run", false, "Show the command that would be executed without running it")
		quiet          = flags.Bool("q", false, "Quiet mode - suppress all output except errors")
		prompt         = flags.String("prompt", "", "Prompt to pass to Claude (required if no skill provided)")
		model          = flags.String("model", "", "Override model to use (overrides SKILL.md setting)")
		allowedTools   = flags.String("allowed-tools", "", "Override allowed tools (overrides SKILL.md setting)")
		permissionMode = flags.String("permission-mode", "", "Override permission mode (default: acceptEdits)")
		outputFormat   = flags.String("output-format", "", "Override output format (default: stream-json)")
		colorFlag      = flags.String("color", "auto", "Control color output (auto, always, never)")
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
		printHelp(stdout, *colorFlag)
		return nil
	}

	if *listSkills {
		return listAvailable(stdout, *colorFlag)
	}

	// Parse skill or command if provided
	var skill *parser.Skill
	var cmd *command.Command
	var resourceName string
	var resourcePath string
	if len(posArgs) > 0 {
		result, err := resolver.Resolve(posArgs[0])
		if err != nil {
			return fmt.Errorf("failed to resolve skill or command: %w", err)
		}
		if result.IsURL {
			defer func() { _ = os.Remove(result.Path) }()
		}

		resourcePath = result.Path

		switch result.Type {
		case resolver.ResourceTypeSkill:
			if result.BaseURL != "" {
				skill, err = parser.ParseWithBaseDir(result.Path, result.BaseURL)
			} else {
				skill, err = parser.Parse(result.Path)
			}
			if err != nil {
				return fmt.Errorf("failed to parse skill file: %w", err)
			}
			resourceName = skill.Name
		case resolver.ResourceTypeCommand:
			arguments := strings.Join(posArgs[1:], " ")
			if result.BaseURL != "" {
				cmd, err = command.ParseWithBaseDir(result.Path, result.BaseURL, arguments)
			} else {
				cmd, err = command.Parse(result.Path, arguments)
			}
			if err != nil {
				return fmt.Errorf("failed to parse command file: %w", err)
			}
			resourceName = cmd.Name
		}
	}

	// Require --prompt when no skill/command is provided
	if skill == nil && cmd == nil && *prompt == "" {
		printHelp(stdout, *colorFlag)
		return nil
	}

	// Build executor config with resolved values
	config := executor.Config{
		Prompt:         resolvePromptFromResource(*prompt, skill, cmd),
		SystemPrompt:   buildSystemPromptFromResource(skill, cmd),
		Model:          resolveString(*model, resourceModel(skill, cmd)),
		AllowedTools:   resolveString(*allowedTools, resourceAllowedTools(skill, cmd)),
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
		SkillName:       resourceName,
		SkillPath:       resourcePath,
		Color:           *colorFlag,
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
		descStyle.Render("  Skillet parses SKILL.md and command files and executes them using the Claude CLI."),
		descStyle.Render("  It reads the frontmatter configuration, interpolates variables, and"),
		descStyle.Render("  invokes Claude with the appropriate arguments in headless mode."),
		"",
		descStyle.Render("  You can also run skillet without a skill/command by providing --prompt directly."),
		"",
		"  The skill/command path can be:",
		"  • An exact file path "+codeStyle.Render("(e.g., path/to/SKILL.md or path/to/command.md)"),
		"  • A directory containing SKILL.md "+codeStyle.Render("(e.g., path/to/skill)"),
		"  • A skill name in .claude/skills/ "+codeStyle.Render("(e.g., skill-name)"),
		"  • A command name in .claude/commands/ "+codeStyle.Render("(e.g., command-name)"),
		"  • A URL to a skill/command file "+codeStyle.Render("(e.g., https://example.com/skill.md)"),
	)

	options := lipgloss.JoinVertical(lipgloss.Left,
		sectionStyle.Render("Options:"),
		fmt.Sprintf("  %s              Show this help message", optionStyle.Render("--help")),
		fmt.Sprintf("  %s           Show version information", optionStyle.Render("--version")),
		fmt.Sprintf("  %s              List available skills and commands", optionStyle.Render("--list")),
		fmt.Sprintf("  %s           Show detailed output with thinking and tool details", optionStyle.Render("--verbose")),
		fmt.Sprintf("  %s             Print raw JSON stream to stderr (for debugging)", optionStyle.Render("--debug")),
		fmt.Sprintf("  %s             Show token usage statistics after execution", optionStyle.Render("--usage")),
		fmt.Sprintf("  %s           Show the command without running it", optionStyle.Render("--dry-run")),
		fmt.Sprintf("  %s, %s         Suppress all output except errors", optionStyle.Render("-q"), optionStyle.Render("--quiet")),
		fmt.Sprintf("  %s            Prompt to pass to Claude (required without skill)", optionStyle.Render("--prompt")),
		fmt.Sprintf("  %s             Model to use (overrides skill setting)", optionStyle.Render("--model")),
		fmt.Sprintf("  %s     Allowed tools (overrides skill setting)", optionStyle.Render("--allowed-tools")),
		fmt.Sprintf("  %s   Permission mode (default: acceptEdits)", optionStyle.Render("--permission-mode")),
		fmt.Sprintf("  %s     Output format (default: stream-json)", optionStyle.Render("--output-format")),
		fmt.Sprintf("  %s             Color output: auto, always, never", optionStyle.Render("--color")),
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

func listAvailable(w io.Writer, colorMode string) error {
	useColors := color.ShouldUseColors(colorMode)

	// Create skill path and discoverer
	skillPath, err := skillpath.New()
	if err != nil {
		return fmt.Errorf("failed to initialize skill path: %w", err)
	}

	skillDisc := discovery.New(skillPath)
	skills, err := skillDisc.Discover()
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	// Create command path and discoverer
	cmdPath, err := commandpath.New()
	if err != nil {
		return fmt.Errorf("failed to initialize command path: %w", err)
	}

	cmdDisc := command.NewDiscoverer(cmdPath)
	commands, err := cmdDisc.Discover()
	if err != nil {
		return fmt.Errorf("failed to discover commands: %w", err)
	}

	// Define styles
	titleStyle := lipgloss.NewStyle().Bold(true).MarginBottom(1)
	sectionStyle := lipgloss.NewStyle().Bold(true).MarginTop(1)
	nameStyle := lipgloss.NewStyle().Bold(true)
	pathStyle := lipgloss.NewStyle()
	overshadowedNameStyle := lipgloss.NewStyle()
	overshadowedPathStyle := lipgloss.NewStyle()
	overshadowedLabelStyle := lipgloss.NewStyle()
	noItemsStyle := lipgloss.NewStyle().Italic(true)
	namespaceStyle := lipgloss.NewStyle().Italic(true)

	if useColors {
		titleStyle = titleStyle.Foreground(lipgloss.Color("6"))     // Cyan
		sectionStyle = sectionStyle.Foreground(lipgloss.Color("3")) // Yellow
		nameStyle = nameStyle.Foreground(lipgloss.Color("2"))       // Green
		pathStyle = pathStyle.Foreground(lipgloss.Color("8"))       // Dim gray
		overshadowedNameStyle = overshadowedNameStyle.
			Foreground(lipgloss.Color("8")).
			Strikethrough(true)
		overshadowedPathStyle = overshadowedPathStyle.
			Foreground(lipgloss.Color("8"))
		overshadowedLabelStyle = overshadowedLabelStyle.
			Foreground(lipgloss.Color("8")).
			Italic(true)
		noItemsStyle = noItemsStyle.Foreground(lipgloss.Color("8")) // Dim
		namespaceStyle = namespaceStyle.Foreground(lipgloss.Color("8"))
	}

	var lines []string
	lines = append(lines, titleStyle.Render("Available Skills and Commands"))
	lines = append(lines, "")

	// Skills section
	lines = append(lines, sectionStyle.Render("Skills"))
	if len(skills) == 0 {
		lines = append(lines, noItemsStyle.Render("  No skills found."))
		lines = append(lines, "")
		lines = append(lines, "  Skills are looked for in:")
		for _, source := range skillPath.Sources() {
			lines = append(lines, fmt.Sprintf("    • %s (%s)", source.Path, source.Name))
		}
	} else {
		// Find the longest skill display name for alignment
		maxNameLen := 0
		for _, skill := range skills {
			sourceInfo := formatSourceInfo(skill.Source.Name, skill.Namespace)
			displayLen := len(skill.Name) + len(" ("+sourceInfo+")")
			if displayLen > maxNameLen {
				maxNameLen = displayLen
			}
		}

		for _, skill := range skills {
			relPath := discovery.RelativePath(skill)
			sourceInfo := formatSourceInfo(skill.Source.Name, skill.Namespace)
			rawDisplayLen := len(skill.Name) + len(" ("+sourceInfo+")")
			padding := strings.Repeat(" ", maxNameLen-rawDisplayLen)

			if skill.Overshadowed {
				displayName := skill.Name + " " + namespaceStyle.Render("("+sourceInfo+")")
				name := overshadowedNameStyle.Render(displayName)
				path := overshadowedPathStyle.Render(relPath)
				label := overshadowedLabelStyle.Render(" (overshadowed)")
				lines = append(lines, fmt.Sprintf("  %s%s  %s%s", name, padding, path, label))
			} else {
				name := nameStyle.Render(skill.Name) + " " + namespaceStyle.Render("("+sourceInfo+")")
				path := pathStyle.Render(relPath)
				lines = append(lines, fmt.Sprintf("  %s%s  %s", name, padding, path))
			}
		}
	}

	lines = append(lines, "")

	// Commands section
	lines = append(lines, sectionStyle.Render("Commands"))
	if len(commands) == 0 {
		lines = append(lines, noItemsStyle.Render("  No commands found."))
		lines = append(lines, "")
		lines = append(lines, "  Commands are looked for in:")
		for _, source := range cmdPath.Sources() {
			lines = append(lines, fmt.Sprintf("    • %s (%s)", source.Path, source.Name))
		}
	} else {
		// Find the longest command display name for alignment
		maxNameLen := 0
		for _, cmd := range commands {
			sourceInfo := formatSourceInfo(cmd.Source.Name, cmd.Namespace)
			displayLen := len(cmd.Name) + len(" ("+sourceInfo+")")
			if displayLen > maxNameLen {
				maxNameLen = displayLen
			}
		}

		for _, cmd := range commands {
			relPath := command.RelativePath(cmd)
			sourceInfo := formatSourceInfo(cmd.Source.Name, cmd.Namespace)
			rawDisplayLen := len(cmd.Name) + len(" ("+sourceInfo+")")
			padding := strings.Repeat(" ", maxNameLen-rawDisplayLen)

			if cmd.Overshadowed {
				displayName := cmd.Name + " " + namespaceStyle.Render("("+sourceInfo+")")
				name := overshadowedNameStyle.Render(displayName)
				path := overshadowedPathStyle.Render(relPath)
				label := overshadowedLabelStyle.Render(" (overshadowed)")
				lines = append(lines, fmt.Sprintf("  %s%s  %s%s", name, padding, path, label))
			} else {
				name := nameStyle.Render(cmd.Name) + " " + namespaceStyle.Render("("+sourceInfo+")")
				path := pathStyle.Render(relPath)
				lines = append(lines, fmt.Sprintf("  %s%s  %s", name, padding, path))
			}
		}
	}

	output := lipgloss.JoinVertical(lipgloss.Left, lines...)
	_, _ = fmt.Fprintln(w, output)
	return nil
}

// Helper functions for resource value extraction

// formatSourceInfo returns the source display string: "source" or "source:namespace"
func formatSourceInfo(sourceName, namespace string) string {
	if namespace != "" {
		return sourceName + ":" + namespace
	}
	return sourceName
}

func resourceModel(s *parser.Skill, c *command.Command) string {
	if s != nil && s.Model != "" && s.Model != "inherit" {
		return s.Model
	}
	if c != nil && c.Model != "" && c.Model != "inherit" {
		return c.Model
	}
	return ""
}

func resourceAllowedTools(s *parser.Skill, c *command.Command) string {
	if s != nil && s.AllowedTools != "" {
		return s.AllowedTools
	}
	if c != nil && c.AllowedTools != "" {
		return c.AllowedTools
	}
	return ""
}

func resolvePromptFromResource(cliPrompt string, s *parser.Skill, c *command.Command) string {
	if cliPrompt != "" {
		return cliPrompt
	}
	if s != nil {
		return s.Description
	}
	if c != nil {
		return c.Description
	}
	return ""
}

func resolveString(override, fallback string) string {
	if override != "" {
		return override
	}
	return fallback
}

func buildSystemPromptFromResource(s *parser.Skill, c *command.Command) string {
	if s != nil {
		return buildSkillSystemPrompt(s)
	}
	if c != nil {
		return buildCommandSystemPrompt(c)
	}
	return ""
}

func buildSkillSystemPrompt(s *parser.Skill) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", s.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", s.Description))
	if s.Compatibility != "" {
		sb.WriteString(fmt.Sprintf("**Compatibility:** %s\n\n", s.Compatibility))
	}
	sb.WriteString(s.Content)
	return sb.String()
}

func buildCommandSystemPrompt(c *command.Command) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", c.Name))
	if c.Description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", c.Description))
	}
	sb.WriteString(c.Content)
	return sb.String()
}
