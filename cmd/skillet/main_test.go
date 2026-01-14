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
