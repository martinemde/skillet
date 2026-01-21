// Package completion provides shell completion scripts and dynamic completion support.
package completion

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/martinemde/skillet/internal/command"
	"github.com/martinemde/skillet/internal/commandpath"
	"github.com/martinemde/skillet/internal/discovery"
	"github.com/martinemde/skillet/internal/skillpath"
)

// Shell completion script generators
var generators = map[string]func(io.Writer) error{
	"bash": GenerateBash,
	"zsh":  GenerateZsh,
	"fish": GenerateFish,
}

// Flag values for completion
var (
	// ColorValues are valid values for --color flag
	ColorValues = []string{"auto", "always", "never"}

	// ModelValues are available Claude models
	ModelValues = []string{
		"claude-opus-4-5-20251101",
		"claude-sonnet-4-20250514",
		"claude-haiku-3-5-20241022",
	}

	// ToolValues are available Claude tools
	ToolValues = []string{
		"Read", "Write", "Bash", "Edit", "Grep", "Glob",
		"LS", "WebFetch", "WebSearch", "Task", "TodoWrite",
	}

	// PermissionModeValues are valid permission modes
	PermissionModeValues = []string{
		"default", "acceptEdits", "bypassPermissions", "plan",
	}

	// OutputFormatValues are valid output formats
	OutputFormatValues = []string{
		"stream-json", "json", "text",
	}
)

// Generate writes the completion script for the given shell to the writer.
func Generate(w io.Writer, shell string) error {
	gen, ok := generators[shell]
	if !ok {
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shell)
	}
	return gen(w)
}

// SupportedShells returns a list of supported shell names.
func SupportedShells() []string {
	shells := make([]string, 0, len(generators))
	for shell := range generators {
		shells = append(shells, shell)
	}
	sort.Strings(shells)
	return shells
}

// CompleteNames returns skill and command names matching the given prefix.
// Names are returned as qualified names (namespace:name or just name).
func CompleteNames(prefix string) []string {
	var names []string

	// Discover skills
	skillPath, err := skillpath.New()
	if err == nil {
		skillDisc := discovery.New(skillPath)
		skills, err := skillDisc.Discover()
		if err == nil {
			for _, s := range skills {
				if !s.Overshadowed {
					name := s.QualifiedName()
					if strings.HasPrefix(name, prefix) {
						names = append(names, name)
					}
				}
			}
		}
	}

	// Discover commands
	cmdPath, err := commandpath.New()
	if err == nil {
		cmdDisc := command.NewDiscoverer(cmdPath)
		commands, err := cmdDisc.Discover()
		if err == nil {
			for _, c := range commands {
				if !c.Overshadowed {
					name := c.QualifiedName()
					if strings.HasPrefix(name, prefix) {
						names = append(names, name)
					}
				}
			}
		}
	}

	sort.Strings(names)
	return names
}

// PrintCompletions writes completion names to stdout, one per line.
func PrintCompletions(w io.Writer, prefix string) {
	for _, name := range CompleteNames(prefix) {
		_, _ = fmt.Fprintln(w, name)
	}
}
