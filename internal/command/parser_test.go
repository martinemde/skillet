package command

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParse_SimpleCommand(t *testing.T) {
	cmd, err := Parse("../../testdata/commands/simple-command.md", "")
	if err != nil {
		t.Fatalf("Failed to parse simple command: %v", err)
	}

	if cmd.Name != "simple-command" {
		t.Errorf("Expected name 'simple-command', got '%s'", cmd.Name)
	}

	expectedDesc := "A simple test command"
	if cmd.Description != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, cmd.Description)
	}

	if !strings.Contains(cmd.Content, "Simple Command") {
		t.Error("Content should contain 'Simple Command' heading")
	}

	if cmd.BaseDir == "" {
		t.Error("BaseDir should not be empty")
	}
}

func TestParse_ComprehensiveCommand(t *testing.T) {
	cmd, err := Parse("../../testdata/commands/comprehensive-command.md", "")
	if err != nil {
		t.Fatalf("Failed to parse comprehensive command: %v", err)
	}

	// Test name from filename
	if cmd.Name != "comprehensive-command" {
		t.Errorf("Expected name 'comprehensive-command', got '%s'", cmd.Name)
	}

	// Test description from frontmatter
	if cmd.Description != "A comprehensive test command" {
		t.Errorf("Unexpected description: %s", cmd.Description)
	}

	// Test optional fields
	if cmd.AllowedTools != "Read Write Bash(git:*)" {
		t.Errorf("Unexpected allowed-tools: %s", cmd.AllowedTools)
	}

	if cmd.ArgumentHint != "[filename] [options]" {
		t.Errorf("Unexpected argument-hint: %s", cmd.ArgumentHint)
	}

	if cmd.Model != "claude-sonnet-4-5-20250929" {
		t.Errorf("Expected model 'claude-sonnet-4-5-20250929', got '%s'", cmd.Model)
	}

	if !cmd.DisableModelInvocation {
		t.Error("Expected disable-model-invocation to be true")
	}

	// Test content
	if !strings.Contains(cmd.Content, "Comprehensive Command") {
		t.Error("Content should contain 'Comprehensive Command' heading")
	}
}

func TestParse_CommandWithoutFrontmatter(t *testing.T) {
	cmd, err := Parse("../../testdata/commands/no-frontmatter-command.md", "")
	if err != nil {
		t.Fatalf("Failed to parse command without frontmatter: %v", err)
	}

	// Name should be derived from filename
	if cmd.Name != "no-frontmatter-command" {
		t.Errorf("Expected name 'no-frontmatter-command', got '%s'", cmd.Name)
	}

	// Description should be derived from first line of content
	if !strings.HasPrefix(cmd.Description, "This is a command without any frontmatter.") {
		t.Errorf("Expected description from content, got '%s'", cmd.Description)
	}

	// Content should include the full content
	if !strings.Contains(cmd.Content, "Command Without Frontmatter") {
		t.Error("Content should contain heading")
	}
}

func TestParse_InvalidNameCommand(t *testing.T) {
	_, err := Parse("../../testdata/commands/invalid-name-UPPERCASE.md", "")
	if err == nil {
		t.Fatal("Expected error for invalid command name, got nil")
	}

	if !strings.Contains(err.Error(), "invalid command name format") {
		t.Errorf("Expected 'invalid command name format' error, got: %v", err)
	}
}

func TestParse_NonexistentFile(t *testing.T) {
	_, err := Parse("../../testdata/commands/nonexistent.md", "")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
}

func TestValidate_NameValidation(t *testing.T) {
	tests := []struct {
		name        string
		commandName string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid lowercase",
			commandName: "my-command",
			wantErr:     false,
		},
		{
			name:        "valid with numbers",
			commandName: "command-123",
			wantErr:     false,
		},
		{
			name:        "valid single character",
			commandName: "a",
			wantErr:     false,
		},
		{
			name:        "invalid uppercase",
			commandName: "My-Command",
			wantErr:     true,
			errMsg:      "invalid command name format",
		},
		{
			name:        "invalid starting with hyphen",
			commandName: "-mycommand",
			wantErr:     true,
			errMsg:      "invalid command name format",
		},
		{
			name:        "invalid ending with hyphen",
			commandName: "mycommand-",
			wantErr:     true,
			errMsg:      "invalid command name format",
		},
		{
			name:        "invalid consecutive hyphens",
			commandName: "my--command",
			wantErr:     true,
			errMsg:      "consecutive hyphens",
		},
		{
			name:        "invalid too long",
			commandName: strings.Repeat("a", 65),
			wantErr:     true,
			errMsg:      "command name too long",
		},
		{
			name:        "invalid empty",
			commandName: "",
			wantErr:     true,
			errMsg:      "command name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Command{
				Name:    tt.commandName,
				Content: "Test content",
			}
			err := cmd.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidate_ContentRequired(t *testing.T) {
	cmd := &Command{
		Name:    "test-command",
		Content: "",
	}

	err := cmd.Validate()
	if err == nil {
		t.Error("Expected error for empty content")
	}

	if !strings.Contains(err.Error(), "content is required") {
		t.Errorf("Expected 'content is required' error, got: %v", err)
	}
}

func TestInterpolateVariables(t *testing.T) {
	baseDir := "/path/to/command"
	content := "Base directory is {baseDir} and config is at {baseDir}/config.json"

	result := interpolateVariables(content, baseDir, "")

	expected := "Base directory is /path/to/command and config is at /path/to/command/config.json"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestInterpolateVariables_Arguments(t *testing.T) {
	content := "Process file $ARGUMENTS with options"
	result := interpolateVariables(content, "/base", "myfile.txt --verbose")

	expected := "Process file myfile.txt --verbose with options"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestInterpolateVariables_MultipleArguments(t *testing.T) {
	content := "First: $ARGUMENTS, Second: $ARGUMENTS"
	result := interpolateVariables(content, "/base", "arg1 arg2")

	expected := "First: arg1 arg2, Second: arg1 arg2"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestInterpolateVariables_EmptyArguments(t *testing.T) {
	content := "Process $ARGUMENTS here"
	result := interpolateVariables(content, "/base", "")

	expected := "Process  here"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestParseWithBaseDir(t *testing.T) {
	customBaseDir := "/custom/base"
	cmd, err := ParseWithBaseDir("../../testdata/commands/simple-command.md", customBaseDir, "")
	if err != nil {
		t.Fatalf("Failed to parse command: %v", err)
	}

	if cmd.BaseDir != customBaseDir {
		t.Errorf("Expected BaseDir '%s', got '%s'", customBaseDir, cmd.BaseDir)
	}
}

func TestExtractFirstLine(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple text",
			content:  "First line\nSecond line",
			expected: "First line",
		},
		{
			name:     "skip heading",
			content:  "# Heading\n\nFirst paragraph",
			expected: "First paragraph",
		},
		{
			name:     "skip multiple headings",
			content:  "# H1\n## H2\n\nActual content",
			expected: "Actual content",
		},
		{
			name:     "skip empty lines",
			content:  "\n\n\nActual content",
			expected: "Actual content",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
		{
			name:     "only headings",
			content:  "# Heading\n## Another",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFirstLine(tt.content)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestParse_NamespacedCommand(t *testing.T) {
	cmd, err := Parse("../../testdata/commands/frontend/component.md", "")
	if err != nil {
		t.Fatalf("Failed to parse namespaced command: %v", err)
	}

	// Name should be derived from filename (not including namespace)
	if cmd.Name != "component" {
		t.Errorf("Expected name 'component', got '%s'", cmd.Name)
	}

	// BaseDir should be the directory containing the command
	absPath, _ := filepath.Abs("../../testdata/commands/frontend")
	if cmd.BaseDir != absPath {
		t.Errorf("Expected BaseDir '%s', got '%s'", absPath, cmd.BaseDir)
	}
}

func TestParse_WithArguments(t *testing.T) {
	// comprehensive-command.md contains "$ARGUMENTS" in its content
	cmd, err := Parse("../../testdata/commands/comprehensive-command.md", "myfile.txt --verbose")
	if err != nil {
		t.Fatalf("Failed to parse command with arguments: %v", err)
	}

	// Verify $ARGUMENTS was replaced in the content
	if strings.Contains(cmd.Content, "$ARGUMENTS") {
		t.Error("Content should not contain literal $ARGUMENTS after interpolation")
	}

	// Verify the arguments were inserted
	if !strings.Contains(cmd.Content, "myfile.txt --verbose") {
		t.Errorf("Content should contain interpolated arguments, got: %s", cmd.Content)
	}
}

func TestInterpolateVariables_AppendArgumentsWhenNotPresent(t *testing.T) {
	content := "No arguments placeholder in content"
	result := interpolateVariables(content, "/base", "myarg --flag")

	expected := "No arguments placeholder in content\n\nARGUMENTS: myarg --flag"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestInterpolateVariables_NoAppendWhenArgumentsEmpty(t *testing.T) {
	content := "No arguments placeholder in content"
	result := interpolateVariables(content, "/base", "")

	// Content should remain unchanged when arguments are empty
	if result != content {
		t.Errorf("Expected '%s', got '%s'", content, result)
	}
}

func TestInterpolateVariables_NoAppendWhenPlaceholderPresent(t *testing.T) {
	content := "Use $ARGUMENTS here"
	result := interpolateVariables(content, "/base", "myarg")

	// $ARGUMENTS should be replaced, not appended
	expected := "Use myarg here"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
	// Should NOT contain "ARGUMENTS:" appended at end
	if strings.Contains(result, "\n\nARGUMENTS:") {
		t.Error("Should not append ARGUMENTS: when $ARGUMENTS placeholder exists")
	}
}

func TestParse_ArgumentsAppendedWhenNotInContent(t *testing.T) {
	// simple-command.md doesn't contain "$ARGUMENTS" in its content
	cmd, err := Parse("../../testdata/commands/simple-command.md", "extra args here")
	if err != nil {
		t.Fatalf("Failed to parse command: %v", err)
	}

	// Arguments should be appended as "ARGUMENTS: <value>"
	if !strings.Contains(cmd.Content, "\n\nARGUMENTS: extra args here") {
		t.Errorf("Content should have ARGUMENTS appended, got: %s", cmd.Content)
	}
}
