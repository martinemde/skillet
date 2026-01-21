// Package pluginpath loads Claude Code plugin install paths from the plugins configuration.
// Each installed plugin's skills and commands directories are added as sources for discovery.
package pluginpath

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/martinemde/skillet/internal/resourcepath"
)

const (
	// PluginsDir is the directory within .claude that contains plugin configuration
	PluginsDir = "plugins"
	// InstalledPluginsFile is the name of the installed plugins configuration file
	InstalledPluginsFile = "installed_plugins.json"
)

// InstalledPlugins represents the structure of installed_plugins.json
type InstalledPlugins struct {
	Version int                        `json:"version"`
	Plugins map[string][]PluginInstall `json:"plugins"`
}

// PluginInstall represents a single plugin installation
type PluginInstall struct {
	Scope       string `json:"scope"`
	InstallPath string `json:"installPath"`
	Version     string `json:"version"`
}

// PluginSource represents a plugin with its sources for skills and commands
type PluginSource struct {
	// Name is the plugin name (e.g., "beads" from "beads@beads-marketplace")
	Name string
	// FullName is the full plugin identifier (e.g., "beads@beads-marketplace")
	FullName string
	// InstallPath is the path to the plugin installation directory
	InstallPath string
	// Scope is the installation scope (user, project, local, managed)
	Scope string
}

// Load reads the installed plugins configuration and returns plugin sources.
// It looks for ~/.claude/plugins/installed_plugins.json by default.
func Load() ([]PluginSource, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(homeDir, resourcepath.ClaudeDir, PluginsDir, InstalledPluginsFile)
	return LoadFromFile(configPath)
}

// LoadFromFile reads the installed plugins configuration from a specific file path.
func LoadFromFile(configPath string) ([]PluginSource, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No plugins installed, return empty list
			return nil, nil
		}
		return nil, err
	}

	var config InstalledPlugins
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	var sources []PluginSource
	for fullName, installs := range config.Plugins {
		// Extract the plugin name from "name@marketplace" format
		name := extractPluginName(fullName)

		// Use the first (or most recent) installation
		// In practice, each plugin typically has one installation per scope
		for _, install := range installs {
			sources = append(sources, PluginSource{
				Name:        name,
				FullName:    fullName,
				InstallPath: install.InstallPath,
				Scope:       install.Scope,
			})
		}
	}

	// Sort by name for consistent ordering
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Name < sources[j].Name
	})

	return sources, nil
}

// extractPluginName extracts the plugin name from "name@marketplace" format
func extractPluginName(fullName string) string {
	if idx := strings.Index(fullName, "@"); idx != -1 {
		return fullName[:idx]
	}
	return fullName
}

// SkillSources returns resourcepath.Source entries for plugin skills directories.
// Sources are ordered alphabetically by plugin name and start at the given priority.
func SkillSources(plugins []PluginSource, startPriority int) []resourcepath.Source {
	var sources []resourcepath.Source
	for i, plugin := range plugins {
		skillsPath := filepath.Join(plugin.InstallPath, "skills")
		sources = append(sources, resourcepath.Source{
			Path:      skillsPath,
			Name:      "plugin:" + plugin.Name,
			Priority:  startPriority + i,
			Namespace: plugin.Name,
		})
	}
	return sources
}

// CommandSources returns resourcepath.Source entries for plugin commands directories.
// Sources are ordered alphabetically by plugin name and start at the given priority.
func CommandSources(plugins []PluginSource, startPriority int) []resourcepath.Source {
	var sources []resourcepath.Source
	for i, plugin := range plugins {
		commandsPath := filepath.Join(plugin.InstallPath, "commands")
		sources = append(sources, resourcepath.Source{
			Path:      commandsPath,
			Name:      "plugin:" + plugin.Name,
			Priority:  startPriority + i,
			Namespace: plugin.Name,
		})
	}
	return sources
}
