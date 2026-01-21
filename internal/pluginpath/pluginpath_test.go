package pluginpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPluginName(t *testing.T) {
	tests := []struct {
		fullName string
		expected string
	}{
		{"beads@beads-marketplace", "beads"},
		{"plugin-dev@claude-plugins-official", "plugin-dev"},
		{"simple-plugin", "simple-plugin"},
		{"@weird-name", ""},
		{"name@with@multiple@ats", "name"},
	}

	for _, tt := range tests {
		t.Run(tt.fullName, func(t *testing.T) {
			got := extractPluginName(tt.fullName)
			if got != tt.expected {
				t.Errorf("extractPluginName(%q) = %q, want %q", tt.fullName, got, tt.expected)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	t.Run("loads valid plugins file", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "installed_plugins.json")
		content := `{
			"version": 2,
			"plugins": {
				"beads@beads-marketplace": [
					{
						"scope": "user",
						"installPath": "/path/to/beads",
						"version": "1.0.0"
					}
				],
				"plugin-dev@claude-plugins-official": [
					{
						"scope": "user",
						"installPath": "/path/to/plugin-dev",
						"version": "2.0.0"
					}
				]
			}
		}`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		plugins, err := LoadFromFile(configPath)
		if err != nil {
			t.Fatalf("LoadFromFile() error = %v", err)
		}

		if len(plugins) != 2 {
			t.Errorf("LoadFromFile() returned %d plugins, want 2", len(plugins))
		}

		// Plugins should be sorted by name
		if plugins[0].Name != "beads" {
			t.Errorf("First plugin name = %q, want %q", plugins[0].Name, "beads")
		}
		if plugins[0].InstallPath != "/path/to/beads" {
			t.Errorf("First plugin installPath = %q, want %q", plugins[0].InstallPath, "/path/to/beads")
		}

		if plugins[1].Name != "plugin-dev" {
			t.Errorf("Second plugin name = %q, want %q", plugins[1].Name, "plugin-dev")
		}
	})

	t.Run("returns nil for non-existent file", func(t *testing.T) {
		plugins, err := LoadFromFile(filepath.Join(tmpDir, "nonexistent.json"))
		if err != nil {
			t.Errorf("LoadFromFile() error = %v, want nil", err)
		}
		if plugins != nil {
			t.Errorf("LoadFromFile() = %v, want nil", plugins)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "invalid.json")
		if err := os.WriteFile(configPath, []byte("not json"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadFromFile(configPath)
		if err == nil {
			t.Error("LoadFromFile() expected error for invalid JSON")
		}
	})

	t.Run("handles multiple installations per plugin", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "multi_install.json")
		content := `{
			"version": 2,
			"plugins": {
				"test-plugin@marketplace": [
					{
						"scope": "user",
						"installPath": "/user/path",
						"version": "1.0.0"
					},
					{
						"scope": "project",
						"installPath": "/project/path",
						"version": "1.0.0"
					}
				]
			}
		}`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		plugins, err := LoadFromFile(configPath)
		if err != nil {
			t.Fatalf("LoadFromFile() error = %v", err)
		}

		// Should have 2 entries for the same plugin (different scopes)
		if len(plugins) != 2 {
			t.Errorf("LoadFromFile() returned %d plugins, want 2", len(plugins))
		}
	})
}

func TestSkillSources(t *testing.T) {
	plugins := []PluginSource{
		{Name: "alpha", FullName: "alpha@marketplace", InstallPath: "/path/to/alpha"},
		{Name: "beta", FullName: "beta@marketplace", InstallPath: "/path/to/beta"},
	}

	sources := SkillSources(plugins, 2)

	if len(sources) != 2 {
		t.Fatalf("SkillSources() returned %d sources, want 2", len(sources))
	}

	// Check first source
	if sources[0].Path != "/path/to/alpha/skills" {
		t.Errorf("sources[0].Path = %q, want %q", sources[0].Path, "/path/to/alpha/skills")
	}
	if sources[0].Name != "plugin:alpha" {
		t.Errorf("sources[0].Name = %q, want %q", sources[0].Name, "plugin:alpha")
	}
	if sources[0].Priority != 2 {
		t.Errorf("sources[0].Priority = %d, want %d", sources[0].Priority, 2)
	}
	if sources[0].Namespace != "alpha" {
		t.Errorf("sources[0].Namespace = %q, want %q", sources[0].Namespace, "alpha")
	}

	// Check second source
	if sources[1].Path != "/path/to/beta/skills" {
		t.Errorf("sources[1].Path = %q, want %q", sources[1].Path, "/path/to/beta/skills")
	}
	if sources[1].Priority != 3 {
		t.Errorf("sources[1].Priority = %d, want %d", sources[1].Priority, 3)
	}
}

func TestCommandSources(t *testing.T) {
	plugins := []PluginSource{
		{Name: "alpha", FullName: "alpha@marketplace", InstallPath: "/path/to/alpha"},
		{Name: "beta", FullName: "beta@marketplace", InstallPath: "/path/to/beta"},
	}

	sources := CommandSources(plugins, 5)

	if len(sources) != 2 {
		t.Fatalf("CommandSources() returned %d sources, want 2", len(sources))
	}

	// Check first source
	if sources[0].Path != "/path/to/alpha/commands" {
		t.Errorf("sources[0].Path = %q, want %q", sources[0].Path, "/path/to/alpha/commands")
	}
	if sources[0].Name != "plugin:alpha" {
		t.Errorf("sources[0].Name = %q, want %q", sources[0].Name, "plugin:alpha")
	}
	if sources[0].Priority != 5 {
		t.Errorf("sources[0].Priority = %d, want %d", sources[0].Priority, 5)
	}
	if sources[0].Namespace != "alpha" {
		t.Errorf("sources[0].Namespace = %q, want %q", sources[0].Namespace, "alpha")
	}

	// Check second source
	if sources[1].Priority != 6 {
		t.Errorf("sources[1].Priority = %d, want %d", sources[1].Priority, 6)
	}
}
