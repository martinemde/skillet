package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet", "--version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "skillet version") {
		t.Errorf("Version output should contain 'skillet version', got: %s", output)
	}
}

func TestRun_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()
	expectedStrings := []string{
		"Usage:",
		"skillet",
		"SKILL.md",
		"Options:",
		"--help",
		"--version",
		"--verbose",
		"--dry-run",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output should contain '%s', got: %s", expected, output)
		}
	}
}

func TestRun_NoArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()
	// Should show help when no arguments provided
	if !strings.Contains(output, "Usage:") {
		t.Errorf("Should show help when no arguments provided, got: %s", output)
	}
}

func TestRun_DryRun(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet", "--dry-run", "../../testdata/simple-skill/SKILL.md"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()

	// Should show the command that would be executed
	if !strings.Contains(output, "Would execute:") {
		t.Errorf("Dry-run should show 'Would execute:', got: %s", output)
	}

	if !strings.Contains(output, "claude") {
		t.Errorf("Dry-run should show the claude command, got: %s", output)
	}

	if !strings.Contains(output, "--output-format") {
		t.Errorf("Dry-run should show command flags, got: %s", output)
	}
}

func TestRun_InvalidSkillFile(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet", "../../testdata/invalid-skill/SKILL.md"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Expected error for invalid skill file, got nil")
	}

	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestRun_NonexistentFile(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet", "../../testdata/nonexistent/SKILL.md"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to resolve skill") {
		t.Errorf("Expected resolve error, got: %v", err)
	}
}

func TestSeparateFlags(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedFlags   []string
		expectedPosArgs []string
	}{
		{
			name:            "flags before positional args",
			args:            []string{"--verbose", "--usage", "skill-name"},
			expectedFlags:   []string{"--verbose", "--usage"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "flags after positional args",
			args:            []string{"skill-name", "--verbose", "--usage"},
			expectedFlags:   []string{"--verbose", "--usage"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "flags mixed with positional args",
			args:            []string{"--verbose", "skill-name", "--usage"},
			expectedFlags:   []string{"--verbose", "--usage"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "flags with values",
			args:            []string{"--prompt", "test prompt", "skill-name"},
			expectedFlags:   []string{"--prompt", "test prompt"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "flags with values after positional args",
			args:            []string{"skill-name", "--prompt", "test prompt"},
			expectedFlags:   []string{"--prompt", "test prompt"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "flags with equals syntax",
			args:            []string{"--prompt=test prompt", "skill-name"},
			expectedFlags:   []string{"--prompt=test prompt"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "boolean flags mixed",
			args:            []string{"skill-name", "--verbose", "--dry-run", "--usage"},
			expectedFlags:   []string{"--verbose", "--dry-run", "--usage"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "multiple positional args",
			args:            []string{"skill-name", "extra-arg", "--verbose"},
			expectedFlags:   []string{"--verbose"},
			expectedPosArgs: []string{"skill-name", "extra-arg"},
		},
		{
			name:            "only flags",
			args:            []string{"--verbose", "--usage"},
			expectedFlags:   []string{"--verbose", "--usage"},
			expectedPosArgs: []string{},
		},
		{
			name:            "only positional args",
			args:            []string{"skill-name", "extra-arg"},
			expectedFlags:   []string{},
			expectedPosArgs: []string{"skill-name", "extra-arg"},
		},
		{
			name:            "empty args",
			args:            []string{},
			expectedFlags:   []string{},
			expectedPosArgs: []string{},
		},
		{
			name:            "complex mix",
			args:            []string{"--model", "opus", "skill-name", "--verbose", "--prompt", "test", "--usage"},
			expectedFlags:   []string{"--model", "opus", "--verbose", "--prompt", "test", "--usage"},
			expectedPosArgs: []string{"skill-name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagArgs, posArgs := separateFlags(tt.args)

			// Check flag args
			if len(flagArgs) != len(tt.expectedFlags) {
				t.Errorf("Expected %d flag args, got %d: %v", len(tt.expectedFlags), len(flagArgs), flagArgs)
			}
			for i, expected := range tt.expectedFlags {
				if i >= len(flagArgs) || flagArgs[i] != expected {
					t.Errorf("Expected flag arg[%d] = %q, got %q", i, expected, flagArgs[i])
				}
			}

			// Check positional args
			if len(posArgs) != len(tt.expectedPosArgs) {
				t.Errorf("Expected %d positional args, got %d: %v", len(tt.expectedPosArgs), len(posArgs), posArgs)
			}
			for i, expected := range tt.expectedPosArgs {
				if i >= len(posArgs) || posArgs[i] != expected {
					t.Errorf("Expected positional arg[%d] = %q, got %q", i, expected, posArgs[i])
				}
			}
		})
	}
}

func TestRun_VerboseFlagAfterSkillName(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// Test that --verbose works when placed after the skill name
	err := run([]string{"skillet", "--dry-run", "../../testdata/simple-skill/SKILL.md", "--verbose"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()

	// Dry-run should work
	if !strings.Contains(output, "Would execute:") {
		t.Errorf("Dry-run should work with --verbose after skill name, got: %s", output)
	}

	// The command should include --verbose flag
	if !strings.Contains(output, "claude") {
		t.Errorf("Should show the claude command, got: %s", output)
	}
}

func TestRun_QuietFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// Test quiet flag with dry-run (should still show output in dry-run)
	err := run([]string{"skillet", "--dry-run", "--quiet", "../../testdata/simple-skill/SKILL.md"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()
	// Dry-run output should still be shown even with quiet flag
	if !strings.Contains(output, "Would execute:") {
		t.Errorf("Dry-run should show output even with quiet flag, got: %s", output)
	}
}

func TestRun_QuietFlagShortForm(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// Test -q short form
	err := run([]string{"skillet", "--dry-run", "-q", "../../testdata/simple-skill/SKILL.md"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()
	// Dry-run output should still be shown even with -q flag
	if !strings.Contains(output, "Would execute:") {
		t.Errorf("Dry-run should show output even with -q flag, got: %s", output)
	}
}

func TestRun_ColorFlag(t *testing.T) {
	tests := []struct {
		name       string
		colorValue string
	}{
		{"auto", "auto"},
		{"always", "always"},
		{"never", "never"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			err := run([]string{"skillet", "--dry-run", "--color=" + tt.colorValue, "../../testdata/simple-skill/SKILL.md"}, &stdout, &stderr)
			if err != nil {
				t.Fatalf("Run failed: %v", err)
			}

			output := stdout.String()
			if !strings.Contains(output, "Would execute:") {
				t.Errorf("Should work with --color=%s, got: %s", tt.colorValue, output)
			}
		})
	}
}

func TestRun_ColorFlagHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// Test that help shows color flag
	err := run([]string{"skillet", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "--color") {
		t.Errorf("Help should mention --color flag, got: %s", output)
	}
	if !strings.Contains(output, "--quiet") || !strings.Contains(output, "-q") {
		t.Errorf("Help should mention --quiet/-q flag, got: %s", output)
	}
}

func TestSeparateFlags_QuietFlag(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedFlags   []string
		expectedPosArgs []string
	}{
		{
			name:            "quiet flag before skill",
			args:            []string{"--quiet", "skill-name"},
			expectedFlags:   []string{"--quiet"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "quiet flag after skill",
			args:            []string{"skill-name", "--quiet"},
			expectedFlags:   []string{"--quiet"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "q flag before skill",
			args:            []string{"-q", "skill-name"},
			expectedFlags:   []string{"-q"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "q flag after skill",
			args:            []string{"skill-name", "-q"},
			expectedFlags:   []string{"-q"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "color flag with value",
			args:            []string{"--color", "never", "skill-name"},
			expectedFlags:   []string{"--color", "never"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "color flag with equals",
			args:            []string{"--color=always", "skill-name"},
			expectedFlags:   []string{"--color=always"},
			expectedPosArgs: []string{"skill-name"},
		},
		{
			name:            "list flag",
			args:            []string{"--list"},
			expectedFlags:   []string{"--list"},
			expectedPosArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagArgs, posArgs := separateFlags(tt.args)

			// Check flag args
			if len(flagArgs) != len(tt.expectedFlags) {
				t.Errorf("Expected %d flag args, got %d: %v", len(tt.expectedFlags), len(flagArgs), flagArgs)
			}
			for i, expected := range tt.expectedFlags {
				if i >= len(flagArgs) || flagArgs[i] != expected {
					t.Errorf("Expected flag arg[%d] = %q, got %q", i, expected, flagArgs[i])
				}
			}

			// Check positional args
			if len(posArgs) != len(tt.expectedPosArgs) {
				t.Errorf("Expected %d positional args, got %d: %v", len(tt.expectedPosArgs), len(posArgs), posArgs)
			}
			for i, expected := range tt.expectedPosArgs {
				if i >= len(posArgs) || posArgs[i] != expected {
					t.Errorf("Expected positional arg[%d] = %q, got %q", i, expected, posArgs[i])
				}
			}
		})
	}
}

func TestRun_List(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet", "--list"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()
	// Should show the "Available Skills" header
	if !strings.Contains(output, "Available Skills") {
		t.Errorf("List output should contain 'Available Skills', got: %s", output)
	}
}

func TestRun_List_WithColorNever(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet", "--list", "--color=never"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()
	// Should show the "Available Skills" header
	if !strings.Contains(output, "Available Skills") {
		t.Errorf("List output should contain 'Available Skills', got: %s", output)
	}
}

func TestRun_ParseFile(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet", "--parse", "../../testdata/parse/tool-operations.jsonl", "--color=never"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()

	// Check for expected content from the fixture
	expectedStrings := []string{
		"Starting",
		"Read test.txt",
		"Glob **/*.go",
		"Bash Print hello",
		"Completed",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Parse output should contain '%s', got: %s", expected, output)
		}
	}
}

func TestRun_ParseFileConversationLog(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet", "--parse", "../../testdata/parse/conversation-log.jsonl", "--color=never"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()

	// Check for expected content - includes todo items and tool operations
	expectedStrings := []string{
		"Starting",
		"Create greeting.txt file",
		"Write greeting.txt",
		"Completed",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Parse output should contain '%s', got: %s", expected, output)
		}
	}
}

func TestRun_ParseStdin(t *testing.T) {
	// We can't easily test actual stdin in unit tests, but we can test the flag parsing
	// by verifying --parse=- is accepted and the error is about reading, not flag parsing
	var stdout, stderr bytes.Buffer

	// Test with explicit stdin flag
	err := run([]string{"skillet", "--parse=-", "--color=never"}, &stdout, &stderr)
	// This will fail because stdin is empty in tests, but it should not error on flag parsing
	// The error should be about reading/formatting, not about the flag
	if err != nil && !strings.Contains(err.Error(), "formatting failed") {
		t.Fatalf("Unexpected error type: %v", err)
	}
}

func TestRun_ParseWithVerbose(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet", "--parse", "../../testdata/parse/tool-operations.jsonl", "--verbose", "--color=never"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stdout.String()

	// Verbose should still show tool operations
	if !strings.Contains(output, "Read test.txt") {
		t.Errorf("Verbose parse should show tool operations, got: %s", output)
	}
}

func TestRun_ParseNonexistentFile(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"skillet", "--parse", "nonexistent.jsonl"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to open input file") {
		t.Errorf("Expected file open error, got: %v", err)
	}
}

func TestSeparateFlags_ParseFlag(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedFlags   []string
		expectedPosArgs []string
	}{
		{
			name:            "parse with file",
			args:            []string{"--parse", "file.jsonl"},
			expectedFlags:   []string{"--parse", "file.jsonl"},
			expectedPosArgs: []string{},
		},
		{
			name:            "parse with equals",
			args:            []string{"--parse=file.jsonl"},
			expectedFlags:   []string{"--parse=file.jsonl"},
			expectedPosArgs: []string{},
		},
		{
			name:            "parse with stdin",
			args:            []string{"--parse=-"},
			expectedFlags:   []string{"--parse=-"},
			expectedPosArgs: []string{},
		},
		{
			name:            "parse alone defaults to stdin",
			args:            []string{"--parse"},
			expectedFlags:   []string{"--parse=-"},
			expectedPosArgs: []string{},
		},
		{
			name:            "parse with other flags",
			args:            []string{"--parse", "file.jsonl", "--verbose", "--color=never"},
			expectedFlags:   []string{"--parse", "file.jsonl", "--verbose", "--color=never"},
			expectedPosArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagArgs, posArgs := separateFlags(tt.args)

			if len(flagArgs) != len(tt.expectedFlags) {
				t.Errorf("Expected %d flag args, got %d: %v", len(tt.expectedFlags), len(flagArgs), flagArgs)
			}
			for i, expected := range tt.expectedFlags {
				if i >= len(flagArgs) || flagArgs[i] != expected {
					t.Errorf("Expected flag arg[%d] = %q, got %q", i, expected, flagArgs[i])
				}
			}

			if len(posArgs) != len(tt.expectedPosArgs) {
				t.Errorf("Expected %d positional args, got %d: %v", len(tt.expectedPosArgs), len(posArgs), posArgs)
			}
		})
	}
}
