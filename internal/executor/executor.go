package executor

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Config holds the final resolved configuration for executing Claude CLI.
// All values should be resolved before creating the executor.
type Config struct {
	SystemPrompt   string // appended to system prompt; empty means none
	Prompt         string // user prompt to send
	Model          string // empty means use default
	AllowedTools   string // empty means no restriction
	PermissionMode string // empty defaults to "acceptEdits"
	OutputFormat   string // empty defaults to "stream-json"
}

// Executor executes the Claude CLI
type Executor struct {
	config Config
	stdout io.Writer
	stderr io.Writer
}

// New creates a new Executor
func New(config Config, stdout, stderr io.Writer) *Executor {
	return &Executor{
		config: config,
		stdout: stdout,
		stderr: stderr,
	}
}

// Execute runs the Claude CLI
func (e *Executor) Execute(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "claude", e.buildArgs()...)
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr
	return cmd.Run()
}

// buildArgs constructs the command-line arguments for the Claude CLI
func (e *Executor) buildArgs() []string {
	args := []string{
		"-p",
		"--verbose",
		"--output-format", e.outputFormat(),
		"--permission-mode", e.permissionMode(),
	}

	if e.config.Model != "" {
		args = append(args, "--model", e.config.Model)
	}

	if e.config.AllowedTools != "" {
		args = append(args, "--allowed-tools", e.config.AllowedTools)
	}

	if e.config.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", e.config.SystemPrompt)
	}

	if e.config.Prompt != "" {
		args = append(args, e.config.Prompt)
	}

	return args
}

func (e *Executor) outputFormat() string {
	if e.config.OutputFormat != "" {
		return e.config.OutputFormat
	}
	return "stream-json"
}

func (e *Executor) permissionMode() string {
	if e.config.PermissionMode != "" {
		return e.config.PermissionMode
	}
	return "acceptEdits"
}

// GetCommand returns the command string that would be executed (for dry-run)
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
