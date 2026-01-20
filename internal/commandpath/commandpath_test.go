package commandpath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	path, err := New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sources := path.Sources()
	if len(sources) < 1 {
		t.Fatal("expected at least one source")
	}

	// First source should be project-scoped
	if sources[0].Name != "project" {
		t.Errorf("expected first source name to be 'project', got %s", sources[0].Name)
	}
	if sources[0].Priority != 0 {
		t.Errorf("expected first source priority to be 0, got %d", sources[0].Priority)
	}
	if !strings.HasSuffix(sources[0].Path, filepath.Join(ClaudeDir, CommandsDir)) {
		t.Errorf("expected first source path to end with %s, got %s", filepath.Join(ClaudeDir, CommandsDir), sources[0].Path)
	}

	// Second source should be user-scoped (if home directory is available)
	if len(sources) >= 2 {
		if sources[1].Name != "user" {
			t.Errorf("expected second source name to be 'user', got %s", sources[1].Name)
		}
		if sources[1].Priority != 1 {
			t.Errorf("expected second source priority to be 1, got %d", sources[1].Priority)
		}
	}
}

func TestNewWithWorkDir(t *testing.T) {
	workDir := t.TempDir()

	path, err := NewWithWorkDir(workDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sources := path.Sources()
	if len(sources) < 1 {
		t.Fatal("expected at least one source")
	}

	expectedPath := filepath.Join(workDir, ClaudeDir, CommandsDir)
	if sources[0].Path != expectedPath {
		t.Errorf("expected first source path to be %s, got %s", expectedPath, sources[0].Path)
	}
}

func TestNewWithSources(t *testing.T) {
	customSources := []Source{
		{Path: "/custom/path1", Name: "custom1", Priority: 0},
		{Path: "/custom/path2", Name: "custom2", Priority: 1},
	}

	path := NewWithSources(customSources)
	sources := path.Sources()

	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}

	if sources[0].Path != "/custom/path1" {
		t.Errorf("expected first source path to be /custom/path1, got %s", sources[0].Path)
	}
	if sources[1].Path != "/custom/path2" {
		t.Errorf("expected second source path to be /custom/path2, got %s", sources[1].Path)
	}
}

func TestCommandPath(t *testing.T) {
	tests := []struct {
		name        string
		sourceDir   string
		commandName string
		expected    string
	}{
		{
			name:        "simple path",
			sourceDir:   "/home/user/.claude/commands",
			commandName: "my-command",
			expected:    "/home/user/.claude/commands/my-command.md",
		},
		{
			name:        "relative path",
			sourceDir:   ".claude/commands",
			commandName: "test-command",
			expected:    ".claude/commands/test-command.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CommandPath(tt.sourceDir, tt.commandName)
			// Normalize paths for comparison on different OS
			expected := filepath.FromSlash(tt.expected)
			if result != expected {
				t.Errorf("expected %s, got %s", expected, result)
			}
		})
	}
}

func TestSourcePriority(t *testing.T) {
	path, err := New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sources := path.Sources()

	// Verify priorities are in order
	for i := 1; i < len(sources); i++ {
		if sources[i].Priority < sources[i-1].Priority {
			t.Errorf("expected priorities to be in ascending order, but source %d has priority %d and source %d has priority %d",
				i-1, sources[i-1].Priority, i, sources[i].Priority)
		}
	}
}

func TestNewWithWorkDir_EmptyString(t *testing.T) {
	// When workDir is empty, it should use current working directory
	path, err := NewWithWorkDir("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sources := path.Sources()
	if len(sources) < 1 {
		t.Fatal("expected at least one source")
	}

	// Should match current working directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	expectedPath := filepath.Join(wd, ClaudeDir, CommandsDir)
	if sources[0].Path != expectedPath {
		t.Errorf("expected first source path to be %s, got %s", expectedPath, sources[0].Path)
	}
}
