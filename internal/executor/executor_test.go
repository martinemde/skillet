package executor

import (
	"strings"
	"testing"

	"github.com/martinemde/skillet/internal/parser"
)

func TestNew(t *testing.T) {
	skill := &parser.Skill{
		Name:        "test-skill",
		Description: "Test description",
	}
	prompt := "Test prompt"

	exec := New(skill, prompt)

	if exec.skill != skill {
		t.Error("Executor should store the skill")
	}

	if exec.prompt != prompt {
		t.Error("Executor should store the prompt")
	}
}

func TestBuildArgs_Basic(t *testing.T) {
	skill := &parser.Skill{
		Name:        "test-skill",
		Description: "Test description",
		Content:     "Test content",
	}

	exec := New(skill, "")
	args := exec.buildArgs()

	// Check for required args
	if args[0] != "-p" {
		t.Error("First arg should be '-p'")
	}

	// Check for output format
	hasOutputFormat := false
	for i, arg := range args {
		if arg == "--output-format" && i+1 < len(args) && args[i+1] == "stream-json" {
			hasOutputFormat = true
			break
		}
	}
	if !hasOutputFormat {
		t.Error("Args should contain '--output-format stream-json'")
	}

	// Check for system prompt
	hasSystemPrompt := false
	for i, arg := range args {
		if arg == "--system-prompt" && i+1 < len(args) {
			hasSystemPrompt = true
			break
		}
	}
	if !hasSystemPrompt {
		t.Error("Args should contain '--system-prompt'")
	}
}

func TestBuildArgs_WithModel(t *testing.T) {
	skill := &parser.Skill{
		Name:        "test-skill",
		Description: "Test description",
		Model:       "claude-opus-4-5-20251101",
	}

	exec := New(skill, "")
	args := exec.buildArgs()

	// Check for model arg
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

func TestBuildArgs_WithModelInherit(t *testing.T) {
	skill := &parser.Skill{
		Name:        "test-skill",
		Description: "Test description",
		Model:       "inherit",
	}

	exec := New(skill, "")
	args := exec.buildArgs()

	// Check that model arg is NOT included when set to "inherit"
	for i, arg := range args {
		if arg == "--model" {
			t.Errorf("Args should not contain '--model' when model is 'inherit', found at index %d", i)
		}
	}
}

func TestBuildArgs_WithAllowedTools(t *testing.T) {
	skill := &parser.Skill{
		Name:         "test-skill",
		Description:  "Test description",
		AllowedTools: "Read Write Bash(git:*)",
	}

	exec := New(skill, "")
	args := exec.buildArgs()

	// Check for allowed-tools arg
	hasAllowedTools := false
	for i, arg := range args {
		if arg == "--allowed-tools" && i+1 < len(args) && args[i+1] == "Read Write Bash(git:*)" {
			hasAllowedTools = true
			break
		}
	}
	if !hasAllowedTools {
		t.Error("Args should contain '--allowed-tools Read Write Bash(git:*)'")
	}
}

func TestBuildArgs_WithPrompt(t *testing.T) {
	skill := &parser.Skill{
		Name:        "test-skill",
		Description: "Test description",
	}

	prompt := "Custom user prompt"
	exec := New(skill, prompt)
	args := exec.buildArgs()

	// The last argument should be the prompt
	lastArg := args[len(args)-1]
	if lastArg != prompt {
		t.Errorf("Last arg should be the prompt '%s', got '%s'", prompt, lastArg)
	}
}

func TestBuildArgs_WithoutPrompt(t *testing.T) {
	skill := &parser.Skill{
		Name:        "test-skill",
		Description: "Test description",
	}

	exec := New(skill, "")
	args := exec.buildArgs()

	// The last argument should be the description when no prompt is provided
	lastArg := args[len(args)-1]
	if lastArg != skill.Description {
		t.Errorf("Last arg should be the description '%s', got '%s'", skill.Description, lastArg)
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	skill := &parser.Skill{
		Name:        "test-skill",
		Description: "Test description for the skill",
		Content:     "Detailed instructions go here",
	}

	exec := New(skill, "")
	prompt := exec.buildSystemPrompt()

	// Check that the prompt contains the skill name
	if !strings.Contains(prompt, "test-skill") {
		t.Error("System prompt should contain the skill name")
	}

	// Check that the prompt contains the description
	if !strings.Contains(prompt, "Test description for the skill") {
		t.Error("System prompt should contain the description")
	}

	// Check that the prompt contains the content
	if !strings.Contains(prompt, "Detailed instructions go here") {
		t.Error("System prompt should contain the content")
	}
}

func TestBuildSystemPrompt_WithCompatibility(t *testing.T) {
	skill := &parser.Skill{
		Name:          "test-skill",
		Description:   "Test description",
		Compatibility: "Requires git and docker",
		Content:       "Instructions",
	}

	exec := New(skill, "")
	prompt := exec.buildSystemPrompt()

	// Check that the prompt contains compatibility info
	if !strings.Contains(prompt, "Compatibility:") {
		t.Error("System prompt should contain compatibility section")
	}

	if !strings.Contains(prompt, "Requires git and docker") {
		t.Error("System prompt should contain the compatibility text")
	}
}

func TestGetCommand(t *testing.T) {
	skill := &parser.Skill{
		Name:         "test-skill",
		Description:  "Test description",
		Model:        "claude-opus-4-5-20251101",
		AllowedTools: "Read Write",
	}

	exec := New(skill, "Test prompt")
	cmd := exec.GetCommand()

	// Check that the command starts with "claude"
	if !strings.HasPrefix(cmd, "claude ") {
		t.Error("Command should start with 'claude '")
	}

	// Check that it contains expected flags
	expectedFlags := []string{
		"-p",
		"--output-format",
		"stream-json",
		"--model",
		"claude-opus-4-5-20251101",
		"--allowed-tools",
	}

	for _, flag := range expectedFlags {
		if !strings.Contains(cmd, flag) {
			t.Errorf("Command should contain '%s'", flag)
		}
	}
}
