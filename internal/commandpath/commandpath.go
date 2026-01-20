// Package commandpath defines the search path for finding commands.
// The path is a list of directories where commands can be found,
// similar to a shell PATH. Commands are looked up in each directory
// in order, with earlier directories taking precedence.
package commandpath

import (
	"path/filepath"

	"github.com/martinemde/skillet/internal/resourcepath"
)

const (
	// ClaudeDir is the name of the Claude configuration directory
	ClaudeDir = resourcepath.ClaudeDir
	// CommandsDir is the subdirectory within ClaudeDir that contains commands
	CommandsDir = "commands"
)

// Source represents a location where commands can be found
type Source = resourcepath.Source

// Path represents a list of sources to search for commands
type Path struct {
	*resourcepath.Path
}

// New creates a new command path with the default sources:
// 1. Project-scoped: .claude/commands in working directory (priority 0)
// 2. User-scoped: ~/.claude/commands (priority 1)
func New() (*Path, error) {
	return NewWithWorkDir("")
}

// NewWithWorkDir creates a new command path with a specific working directory.
// If workDir is empty, the current working directory is used.
func NewWithWorkDir(workDir string) (*Path, error) {
	p, err := resourcepath.NewWithWorkDir(CommandsDir, workDir)
	if err != nil {
		return nil, err
	}
	return &Path{Path: p}, nil
}

// NewWithSources creates a Path with custom sources.
// This is useful for testing or custom configurations.
func NewWithSources(sources []Source) *Path {
	return &Path{Path: resourcepath.NewWithSources(sources)}
}

// CommandPath returns the expected path for a command with the given name
// in the given source directory.
func CommandPath(sourceDir, commandName string) string {
	return filepath.Join(sourceDir, commandName+".md")
}
