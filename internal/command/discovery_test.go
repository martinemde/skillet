package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/martinemde/skillet/internal/commandpath"
)

func TestDirectoryFinder_Find(t *testing.T) {
	// Use testdata directory
	absPath, err := filepath.Abs("../../testdata/commands")
	if err != nil {
		t.Fatal(err)
	}

	source := commandpath.Source{
		Path:     absPath,
		Name:     "test",
		Priority: 0,
	}

	finder := &DirectoryFinder{}
	commands, err := finder.Find(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find at least the commands we created
	if len(commands) < 3 {
		t.Errorf("expected at least 3 commands, got %d", len(commands))
	}

	// Check for specific commands
	foundSimple := false
	foundComprehensive := false
	foundComponent := false

	for _, cmd := range commands {
		switch cmd.Name {
		case "simple-command":
			foundSimple = true
		case "comprehensive-command":
			foundComprehensive = true
		case "component":
			foundComponent = true
			// Should have namespace
			if cmd.Namespace != "frontend" {
				t.Errorf("expected namespace 'frontend', got '%s'", cmd.Namespace)
			}
		}
	}

	if !foundSimple {
		t.Error("expected to find simple-command")
	}
	if !foundComprehensive {
		t.Error("expected to find comprehensive-command")
	}
	if !foundComponent {
		t.Error("expected to find component (namespaced)")
	}
}

func TestDirectoryFinder_NonexistentDirectory(t *testing.T) {
	source := commandpath.Source{
		Path:     "/nonexistent/path",
		Name:     "test",
		Priority: 0,
	}

	finder := &DirectoryFinder{}
	commands, err := finder.Find(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty list, not error
	if len(commands) != 0 {
		t.Errorf("expected 0 commands, got %d", len(commands))
	}
}

func TestDiscoverer_Discover(t *testing.T) {
	// Use testdata directory
	absPath, err := filepath.Abs("../../testdata/commands")
	if err != nil {
		t.Fatal(err)
	}

	sources := []commandpath.Source{
		{Path: absPath, Name: "test", Priority: 0},
	}

	path := commandpath.NewWithSources(sources)
	disc := NewDiscoverer(path)

	commands, err := disc.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(commands) < 3 {
		t.Errorf("expected at least 3 commands, got %d", len(commands))
	}
}

func TestDiscoverer_DiscoverWithOvershadowing(t *testing.T) {
	// Create temp directories for testing
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create same command in both directories
	cmdContent := "---\ndescription: Test command\n---\nTest content"

	// High priority version
	if err := os.WriteFile(filepath.Join(tmpDir1, "test-cmd.md"), []byte(cmdContent+" v1"), 0644); err != nil {
		t.Fatal(err)
	}

	// Low priority version
	if err := os.WriteFile(filepath.Join(tmpDir2, "test-cmd.md"), []byte(cmdContent+" v2"), 0644); err != nil {
		t.Fatal(err)
	}

	sources := []commandpath.Source{
		{Path: tmpDir1, Name: "high-priority", Priority: 0},
		{Path: tmpDir2, Name: "low-priority", Priority: 1},
	}

	path := commandpath.NewWithSources(sources)
	disc := NewDiscoverer(path)

	commands, err := disc.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}

	// First should be high priority, not overshadowed
	if commands[0].Overshadowed {
		t.Error("high-priority command should not be overshadowed")
	}
	if commands[0].Source.Name != "high-priority" {
		t.Errorf("expected first command from high-priority source, got %s", commands[0].Source.Name)
	}

	// Second should be low priority, overshadowed
	if !commands[1].Overshadowed {
		t.Error("low-priority command should be overshadowed")
	}
	if commands[1].OvershadowedBy == "" {
		t.Error("expected OvershadowedBy to be set")
	}
}

func TestDiscoverer_DiscoverByName(t *testing.T) {
	absPath, err := filepath.Abs("../../testdata/commands")
	if err != nil {
		t.Fatal(err)
	}

	sources := []commandpath.Source{
		{Path: absPath, Name: "test", Priority: 0},
	}

	path := commandpath.NewWithSources(sources)
	disc := NewDiscoverer(path)

	commands, err := disc.DiscoverByName("simple-command")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(commands))
	}

	if commands[0].Name != "simple-command" {
		t.Errorf("expected 'simple-command', got '%s'", commands[0].Name)
	}
}

func TestRelativePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("could not get home directory")
	}

	// Test with home directory path
	cmd := DiscoveredCommand{
		Path: filepath.Join(homeDir, ".claude", "commands", "test.md"),
	}

	relPath := RelativePath(cmd)
	if !strings.HasPrefix(relPath, "~/") {
		t.Errorf("expected path to start with ~/, got %s", relPath)
	}

	// Test with current directory path
	wd, err := os.Getwd()
	if err != nil {
		t.Skip("could not get working directory")
	}

	cmd2 := DiscoveredCommand{
		Path: filepath.Join(wd, ".claude", "commands", "test.md"),
	}

	relPath2 := RelativePath(cmd2)
	if strings.HasPrefix(relPath2, wd) {
		t.Errorf("expected relative path, got absolute: %s", relPath2)
	}
}

func TestDiscoveredCommand_QualifiedName(t *testing.T) {
	tests := []struct {
		name     string
		cmd      DiscoveredCommand
		expected string
	}{
		{
			name:     "unnamespaced",
			cmd:      DiscoveredCommand{Name: "test", Namespace: ""},
			expected: "test",
		},
		{
			name:     "namespaced",
			cmd:      DiscoveredCommand{Name: "test", Namespace: "frontend"},
			expected: "frontend:test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cmd.QualifiedName(); got != tt.expected {
				t.Errorf("QualifiedName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDiscoveredCommand_Key(t *testing.T) {
	tests := []struct {
		name     string
		cmd      DiscoveredCommand
		expected string
	}{
		{
			name:     "unnamespaced",
			cmd:      DiscoveredCommand{Name: "test", Namespace: ""},
			expected: "test",
		},
		{
			name:     "namespaced",
			cmd:      DiscoveredCommand{Name: "test", Namespace: "frontend"},
			expected: "frontend:test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cmd.Key(); got != tt.expected {
				t.Errorf("Key() = %q, want %q", got, tt.expected)
			}
		})
	}
}
