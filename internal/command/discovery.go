package command

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/martinemde/skillet/internal/commandpath"
	"github.com/martinemde/skillet/internal/resourcepath"
)

// DiscoveredCommand represents a discovered command
type DiscoveredCommand struct {
	// Name is the command name (filename without .md)
	Name string
	// Path is the absolute path to the command .md file
	Path string
	// Source is information about where this command was found
	Source commandpath.Source
	// Namespace is the subdirectory within commands (e.g., "frontend" for commands/frontend/foo.md)
	Namespace string
	// Overshadowed indicates this command is hidden by a higher-priority command
	Overshadowed bool
	// OvershadowedBy is the path of the command that shadows this one
	OvershadowedBy string
}

// QualifiedName returns the qualified name for resolution: "namespace:name" or just "name" if no namespace
func (c DiscoveredCommand) QualifiedName() string {
	if c.Namespace != "" {
		return c.Namespace + ":" + c.Name
	}
	return c.Name
}

// Key returns the unique identifier used for overshadowing, same as QualifiedName
func (c DiscoveredCommand) Key() string {
	return c.QualifiedName()
}

// Finder is an interface for finding commands in a source directory.
type Finder interface {
	// Find discovers commands in the given source directory.
	// It returns a list of commands found, without considering precedence.
	Find(source commandpath.Source) ([]DiscoveredCommand, error)
}

// DirectoryFinder finds commands by scanning for .md files
type DirectoryFinder struct{}

// Find discovers commands in the given source by looking for .md files
func (f *DirectoryFinder) Find(source commandpath.Source) ([]DiscoveredCommand, error) {
	var commands []DiscoveredCommand

	// Check if the source directory exists
	if _, err := os.Stat(source.Path); os.IsNotExist(err) {
		return commands, nil
	}

	// Walk the directory to find .md files (supports namespacing via subdirectories).
	// Errors are intentionally skipped: discovery should find all accessible commands rather
	// than failing entirely due to permission issues on a single directory.
	err := filepath.WalkDir(source.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable directories (e.g., permission denied) - continue discovering other commands
			return nil
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .md files
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		// Get name from filename
		name := strings.TrimSuffix(d.Name(), ".md")

		// Determine namespace from relative path
		relPath, err := filepath.Rel(source.Path, filepath.Dir(path))
		if err != nil {
			relPath = ""
		}
		namespace := ""
		if relPath != "." && relPath != "" {
			namespace = relPath
		}

		commands = append(commands, DiscoveredCommand{
			Name:      name,
			Path:      path,
			Source:    source,
			Namespace: namespace,
		})

		return nil
	})

	if err != nil {
		// If we can't walk the directory, just return empty
		return commands, nil
	}

	return commands, nil
}

// Discoverer finds all available commands across a command path
type Discoverer struct {
	path   *commandpath.Path
	finder Finder
}

// NewDiscoverer creates a new Discoverer with the default finder
func NewDiscoverer(path *commandpath.Path) *Discoverer {
	return &Discoverer{
		path:   path,
		finder: &DirectoryFinder{},
	}
}

// NewDiscovererWithFinder creates a new Discoverer with a custom finder
func NewDiscovererWithFinder(path *commandpath.Path, finder Finder) *Discoverer {
	return &Discoverer{
		path:   path,
		finder: finder,
	}
}

// Discover finds all commands across all sources in the path.
// Commands are returned sorted by precedence (source priority), then alphabetically.
// Commands that are overshadowed by higher-priority sources are marked as such.
func (d *Discoverer) Discover() ([]DiscoveredCommand, error) {
	// Track commands we've seen (by name) and their paths
	// Key is "namespace:name" to allow same name in different namespaces
	seen := make(map[string]string)
	var allCommands []DiscoveredCommand

	// Iterate through sources in priority order
	sources := d.path.Sources()
	for _, source := range sources {
		commands, err := d.finder.Find(source)
		if err != nil {
			return nil, err
		}

		for _, cmd := range commands {
			key := cmd.Key()
			if existingPath, exists := seen[key]; exists {
				// This command is overshadowed
				cmd.Overshadowed = true
				cmd.OvershadowedBy = existingPath
			} else {
				// First time seeing this command
				seen[key] = cmd.Path
			}
			allCommands = append(allCommands, cmd)
		}
	}

	// Sort: by source priority, then by namespace, then alphabetically by name
	sort.Slice(allCommands, func(i, j int) bool {
		if allCommands[i].Source.Priority != allCommands[j].Source.Priority {
			return allCommands[i].Source.Priority < allCommands[j].Source.Priority
		}
		if allCommands[i].Namespace != allCommands[j].Namespace {
			return allCommands[i].Namespace < allCommands[j].Namespace
		}
		return allCommands[i].Name < allCommands[j].Name
	})

	return allCommands, nil
}

// DiscoverByName finds all versions of a command with the given name across all sources.
func (d *Discoverer) DiscoverByName(name string) ([]DiscoveredCommand, error) {
	allCommands, err := d.Discover()
	if err != nil {
		return nil, err
	}

	var matches []DiscoveredCommand
	for _, cmd := range allCommands {
		if cmd.Name == name {
			matches = append(matches, cmd)
		}
	}
	return matches, nil
}

// RelativePath returns a display-friendly relative path for the command.
func RelativePath(cmd DiscoveredCommand) string {
	return resourcepath.RelativePath(cmd.Path)
}
