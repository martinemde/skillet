// Package resourcepath provides a generic search path for finding resources.
// The path is a list of directories where resources can be found,
// similar to a shell PATH. Resources are looked up in each directory
// in order, with earlier directories taking precedence.
package resourcepath

import (
	"os"
	"path/filepath"
)

const (
	// ClaudeDir is the name of the Claude configuration directory
	ClaudeDir = ".claude"
)

// Source represents a location where resources can be found
type Source struct {
	// Path is the absolute path to the resource directory
	Path string
	// Name is a human-readable name for this source (e.g., "project", "user")
	Name string
	// Priority determines precedence (lower numbers = higher priority)
	Priority int
	// Namespace is an optional prefix for all resources found in this source.
	// This is used for plugin sources where the plugin name becomes the namespace.
	// For example, a plugin "beads" would have Namespace="beads", so a skill
	// named "workflow" becomes "beads:workflow".
	Namespace string
}

// Path represents a list of sources to search for resources
type Path struct {
	sources []Source
}

// New creates a new resource path with the default sources:
// 1. Project-scoped: .claude/<subdir> in working directory (priority 0)
// 2. User-scoped: ~/.claude/<subdir> (priority 1)
func New(subdir string) (*Path, error) {
	return NewWithWorkDir(subdir, "")
}

// NewWithWorkDir creates a new resource path with a specific working directory.
// If workDir is empty, the current working directory is used.
func NewWithWorkDir(subdir, workDir string) (*Path, error) {
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	var sources []Source

	// Add project-scoped source (working directory)
	projectPath := filepath.Join(workDir, ClaudeDir, subdir)
	sources = append(sources, Source{
		Path:     projectPath,
		Name:     "project",
		Priority: 0,
	})

	// Add user-scoped source (home directory)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userPath := filepath.Join(homeDir, ClaudeDir, subdir)
		sources = append(sources, Source{
			Path:     userPath,
			Name:     "user",
			Priority: 1,
		})
	}

	return &Path{sources: sources}, nil
}

// NewWithSources creates a Path with custom sources.
// This is useful for testing or custom configurations.
func NewWithSources(sources []Source) *Path {
	return &Path{sources: sources}
}

// Sources returns the list of sources in this path
func (p *Path) Sources() []Source {
	return p.sources
}

// AppendSources adds additional sources to the path.
// This is used to add plugin sources after the default sources are created.
func (p *Path) AppendSources(sources []Source) {
	p.sources = append(p.sources, sources...)
}

// RelativePath returns a display-friendly relative path.
// It tries to make the path relative to common reference points:
// 1. Home directory (displayed as ~/...)
// 2. Current directory (displayed as ./... or relative path)
// 3. Falls back to absolute path if neither is shorter
func RelativePath(absPath string) string {
	// Try to make relative to home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		if rel, err := filepath.Rel(homeDir, absPath); err == nil {
			if len(rel) < len(absPath) {
				return "~/" + rel
			}
		}
	}

	// Try to make relative to current directory
	wd, err := os.Getwd()
	if err == nil {
		if rel, err := filepath.Rel(wd, absPath); err == nil {
			if len(rel) < len(absPath) && rel[0] != '.' {
				return "./" + rel
			} else if len(rel) < len(absPath) {
				return rel
			}
		}
	}

	return absPath
}
