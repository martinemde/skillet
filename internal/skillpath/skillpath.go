// Package skillpath defines the search path for finding skills.
// The path is a list of directories where skills can be found,
// similar to a shell PATH. Skills are looked up in each directory
// in order, with earlier directories taking precedence.
package skillpath

import (
	"path/filepath"

	"github.com/martinemde/skillet/internal/pluginpath"
	"github.com/martinemde/skillet/internal/resourcepath"
)

const (
	// ClaudeDir is the name of the Claude configuration directory
	ClaudeDir = resourcepath.ClaudeDir
	// SkillsDir is the subdirectory within ClaudeDir that contains skills
	SkillsDir = "skills"
	// SkillFile is the filename for skill definitions
	SkillFile = "SKILL.md"
)

// Source represents a location where skills can be found
type Source = resourcepath.Source

// Path represents a list of sources to search for skills
type Path struct {
	*resourcepath.Path
}

// New creates a new skill path with the default sources:
// 1. Project-scoped: .claude/skills in working directory (priority 0)
// 2. User-scoped: ~/.claude/skills (priority 1)
func New() (*Path, error) {
	return NewWithWorkDir("")
}

// NewWithWorkDir creates a new skill path with a specific working directory.
// If workDir is empty, the current working directory is used.
// Plugin skill sources are automatically loaded and appended with lower priority.
func NewWithWorkDir(workDir string) (*Path, error) {
	p, err := resourcepath.NewWithWorkDir(SkillsDir, workDir)
	if err != nil {
		return nil, err
	}

	// Load plugin sources (priority 2+, after project and user)
	plugins, err := pluginpath.Load()
	if err == nil && len(plugins) > 0 {
		pluginSources := pluginpath.SkillSources(plugins, 2)
		p.AppendSources(pluginSources)
	}

	return &Path{Path: p}, nil
}

// NewWithSources creates a Path with custom sources.
// This is useful for testing or custom configurations.
func NewWithSources(sources []Source) *Path {
	return &Path{Path: resourcepath.NewWithSources(sources)}
}

// SkillPath returns the expected path for a skill with the given name
// in the given source directory.
func SkillPath(sourceDir, skillName string) string {
	return filepath.Join(sourceDir, skillName, SkillFile)
}
