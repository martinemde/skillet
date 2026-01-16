package executor

import (
	"io"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	config := Config{
		Prompt:       "Test prompt",
		SystemPrompt: "System prompt content",
	}

	exec := New(config, io.Discard, io.Discard)

	if exec.config.Prompt != config.Prompt {
		t.Error("Executor should store the config prompt")
	}
	if exec.config.SystemPrompt != config.SystemPrompt {
		t.Error("Executor should store the system prompt")
	}
}

func TestBuildArgs_Basic(t *testing.T) {
	config := Config{
		Prompt:       "Test prompt",
		SystemPrompt: "System prompt content",
	}

	exec := New(config, io.Discard, io.Discard)
	args := exec.buildArgs()

	// Check required args
	expected := []string{"-p", "--verbose", "--output-format", "stream-json", "--permission-mode", "acceptEdits"}
	for _, exp := range expected {
		found := false
		for _, arg := range args {
			if arg == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Args should contain '%s'", exp)
		}
	}

	// Should have system prompt
	hasSystemPrompt := false
	for i, arg := range args {
		if arg == "--append-system-prompt" && i+1 < len(args) {
			hasSystemPrompt = true
			break
		}
	}
	if !hasSystemPrompt {
		t.Error("Args should contain '--append-system-prompt'")
	}
}

func TestBuildArgs_WithModel(t *testing.T) {
	config := Config{
		Prompt: "Test",
		Model:  "claude-opus-4-5-20251101",
	}

	exec := New(config, io.Discard, io.Discard)
	args := exec.buildArgs()

	hasModel := false
	for i, arg := range args {
		if arg == "--model" && i+1 < len(args) && args[i+1] == "claude-opus-4-5-20251101" {
			hasModel = true
			break
		}
	}
	if !hasModel {
		t.Error("Args should contain '--model claude-opus-4-5-20251101'")
	}
}

func TestBuildArgs_NoModel(t *testing.T) {
	config := Config{
		Prompt: "Test",
		Model:  "", // empty means no model flag
	}

	exec := New(config, io.Discard, io.Discard)
	args := exec.buildArgs()

	for _, arg := range args {
		if arg == "--model" {
			t.Error("Args should not contain '--model' when model is empty")
		}
	}
}

func TestBuildArgs_WithAllowedTools(t *testing.T) {
	config := Config{
		Prompt:       "Test",
		AllowedTools: "Read Write Bash(git:*)",
	}

	exec := New(config, io.Discard, io.Discard)
	args := exec.buildArgs()

	hasTools := false
	for i, arg := range args {
		if arg == "--allowed-tools" && i+1 < len(args) && args[i+1] == "Read Write Bash(git:*)" {
			hasTools = true
			break
		}
	}
	if !hasTools {
		t.Error("Args should contain '--allowed-tools Read Write Bash(git:*)'")
	}
}

func TestBuildArgs_PromptOnly(t *testing.T) {
	config := Config{
		Prompt: "Just a prompt",
	}

	exec := New(config, io.Discard, io.Discard)
	args := exec.buildArgs()

	// Should NOT have --append-system-prompt when empty
	for _, arg := range args {
		if arg == "--append-system-prompt" {
			t.Error("Args should not contain '--append-system-prompt' when system prompt is empty")
		}
	}

	// Last arg should be the prompt
	if args[len(args)-1] != "Just a prompt" {
		t.Errorf("Last arg should be the prompt, got '%s'", args[len(args)-1])
	}
}

func TestBuildArgs_CustomOutputFormat(t *testing.T) {
	config := Config{
		Prompt:       "Test",
		OutputFormat: "text",
	}

	exec := New(config, io.Discard, io.Discard)
	args := exec.buildArgs()

	hasFormat := false
	for i, arg := range args {
		if arg == "--output-format" && i+1 < len(args) && args[i+1] == "text" {
			hasFormat = true
			break
		}
	}
	if !hasFormat {
		t.Error("Args should contain '--output-format text'")
	}
}

func TestBuildArgs_CustomPermissionMode(t *testing.T) {
	config := Config{
		Prompt:         "Test",
		PermissionMode: "plan",
	}

	exec := New(config, io.Discard, io.Discard)
	args := exec.buildArgs()

	hasMode := false
	for i, arg := range args {
		if arg == "--permission-mode" && i+1 < len(args) && args[i+1] == "plan" {
			hasMode = true
			break
		}
	}
	if !hasMode {
		t.Error("Args should contain '--permission-mode plan'")
	}
}

func TestGetCommand(t *testing.T) {
	config := Config{
		Prompt:       "Test prompt",
		SystemPrompt: "System content",
		Model:        "claude-opus-4-5-20251101",
		AllowedTools: "Read Write",
	}

	exec := New(config, io.Discard, io.Discard)
	cmd := exec.GetCommand()

	if !strings.HasPrefix(cmd, "claude ") {
		t.Error("Command should start with 'claude '")
	}

	expectedParts := []string{"-p", "--verbose", "--model", "claude-opus-4-5-20251101", "--allowed-tools"}
	for _, part := range expectedParts {
		if !strings.Contains(cmd, part) {
			t.Errorf("Command should contain '%s'", part)
		}
	}
}

func TestGetCommand_QuotesArgs(t *testing.T) {
	config := Config{
		Prompt: "prompt with spaces",
	}

	exec := New(config, io.Discard, io.Discard)
	cmd := exec.GetCommand()

	if !strings.Contains(cmd, `"prompt with spaces"`) {
		t.Error("Command should quote arguments with spaces")
	}
}
