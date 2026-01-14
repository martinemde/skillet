package executor

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/martinemde/skillet/internal/parser"
)

// Executor executes the Claude CLI with the parsed skill
type Executor struct {
	skill  *parser.Skill
	prompt string
	stdout io.Writer
	stderr io.Writer
}

// New creates a new Executor for the given skill
func New(skill *parser.Skill, prompt string) *Executor {
	return &Executor{
		skill:  skill,
		prompt: prompt,
	}
}

// SetOutput sets the stdout and stderr writers
func (e *Executor) SetOutput(stdout, stderr io.Writer) {
	e.stdout = stdout
	e.stderr = stderr
}

// Execute runs the Claude CLI with the skill configuration
func (e *Executor) Execute(ctx context.Context) error {
	args := e.buildArgs()

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr

	return cmd.Run()
}

// buildArgs constructs the command-line arguments for the Claude CLI
func (e *Executor) buildArgs() []string {
	args := []string{"-p"} // Print mode

	// Add verbose flag (required for streaming to work properly in print mode)
	args = append(args, "--verbose")

	// Add output format
	args = append(args, "--output-format", "stream-json")

	// Add permission mode to allow edits (otherwise Claude can't execute tools)
	args = append(args, "--permission-mode", "acceptEdits")

	// Add model if specified
	if e.skill.Model != "" && e.skill.Model != "inherit" {
		args = append(args, "--model", e.skill.Model)
	}

	// Add allowed tools if specified
	if e.skill.AllowedTools != "" {
		args = append(args, "--allowed-tools", e.skill.AllowedTools)
	}

	// Add system prompt with the skill content
	// We use --append-system-prompt to add the skill instructions
	// while keeping Claude's default capabilities
	systemPrompt := e.buildSystemPrompt()
	args = append(args, "--append-system-prompt", systemPrompt)

	// Add the user prompt (if any)
	if e.prompt != "" {
		args = append(args, e.prompt)
	} else {
		// If no prompt is provided, use the skill description as the prompt
		args = append(args, e.skill.Description)
	}

	return args
}

// buildSystemPrompt constructs the system prompt from the skill content
func (e *Executor) buildSystemPrompt() string {
	var sb strings.Builder

	// Add skill header
	sb.WriteString(fmt.Sprintf("# %s\n\n", e.skill.Name))

	// Add description
	sb.WriteString(fmt.Sprintf("%s\n\n", e.skill.Description))

	// Add compatibility info if present
	if e.skill.Compatibility != "" {
		sb.WriteString(fmt.Sprintf("**Compatibility:** %s\n\n", e.skill.Compatibility))
	}

	// Add the skill content
	sb.WriteString(e.skill.Content)

	return sb.String()
}

// GetCommand returns the command string that would be executed (for debugging)
func (e *Executor) GetCommand() string {
	args := e.buildArgs()
	quoted := make([]string, len(args))
	for i, arg := range args {
		if strings.Contains(arg, " ") || strings.Contains(arg, "\n") {
			quoted[i] = fmt.Sprintf("%q", arg)
		} else {
			quoted[i] = arg
		}
	}
	return "claude " + strings.Join(quoted, " ")
}
