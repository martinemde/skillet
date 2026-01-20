// Package discovery provides skill discovery functionality.
// It can find all available skills across multiple search paths,
// with support for precedence and detecting overshadowed skills.
package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/martinemde/skillet/internal/skillpath"
)

// Skill represents a discovered skill
type Skill struct {
	// Name is the skill name (directory name)
	Name string
	// Path is the absolute path to the SKILL.md file
	Path string
	// Source is information about where this skill was found
	Source skillpath.Source
	// Namespace is the subdirectory within skills (e.g., "frontend" for skills/frontend/test/SKILL.md)
	Namespace string
	// Overshadowed indicates this skill is hidden by a higher-priority skill
	Overshadowed bool
	// OvershadowedBy is the path of the skill that shadows this one
	OvershadowedBy string
}

// QualifiedName returns the qualified name for resolution: "namespace:name" or just "name" if no namespace
func (s Skill) QualifiedName() string {
	if s.Namespace != "" {
		return s.Namespace + ":" + s.Name
	}
	return s.Name
}

// Key returns the unique identifier used for overshadowing, same as QualifiedName
func (s Skill) Key() string {
	return s.QualifiedName()
}

// Finder is an interface for finding skills in a source directory.
// This allows for different discovery strategies (e.g., directory pattern,
// registry lookup, etc.)
type Finder interface {
	// Find discovers skills in the given source directory.
	// It returns a list of skills found, without considering precedence.
	Find(source skillpath.Source) ([]Skill, error)
}

// DirectoryFinder finds skills using the standard directory pattern:
// {source}/skills/{name}/SKILL.md
type DirectoryFinder struct{}

// Find discovers skills in the given source using the directory pattern.
// Supports both unnamespaced skills (skills/name/SKILL.md) and namespaced skills
// (skills/namespace/name/SKILL.md).
func (f *DirectoryFinder) Find(source skillpath.Source) ([]Skill, error) {
	var skills []Skill

	// Check if the source directory exists
	if _, err := os.Stat(source.Path); os.IsNotExist(err) {
		return skills, nil
	}

	// Walk the directory to find SKILL.md files (supports namespacing via subdirectories)
	err := filepath.WalkDir(source.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}

		// Only process SKILL.md files
		if d.IsDir() || !strings.EqualFold(d.Name(), skillpath.SkillFile) {
			return nil
		}

		// Get the relative path from source to the SKILL.md file's directory
		skillDir := filepath.Dir(path)
		relPath, err := filepath.Rel(source.Path, skillDir)
		if err != nil {
			return nil
		}

		// Determine name and namespace from the relative path
		// depth 1: skills/name/SKILL.md -> namespace="", name="name"
		// depth 2: skills/namespace/name/SKILL.md -> namespace="namespace", name="name"
		parts := strings.Split(relPath, string(filepath.Separator))
		var namespace, name string

		switch len(parts) {
		case 1:
			// Unnamespaced: skills/name/SKILL.md
			name = parts[0]
			namespace = ""
		case 2:
			// Namespaced: skills/namespace/name/SKILL.md
			namespace = parts[0]
			name = parts[1]
		default:
			// Deeper nesting not supported, skip
			return nil
		}

		// Skip empty or invalid names
		if name == "" || name == "." {
			return nil
		}

		skills = append(skills, Skill{
			Name:      name,
			Path:      path,
			Source:    source,
			Namespace: namespace,
		})

		return nil
	})

	if err != nil {
		// If we can't walk the directory, just return empty
		return skills, nil
	}

	return skills, nil
}

// Discoverer finds all available skills across a skill path
type Discoverer struct {
	path   *skillpath.Path
	finder Finder
}

// New creates a new Discoverer with the default finder
func New(path *skillpath.Path) *Discoverer {
	return &Discoverer{
		path:   path,
		finder: &DirectoryFinder{},
	}
}

// NewWithFinder creates a new Discoverer with a custom finder
func NewWithFinder(path *skillpath.Path, finder Finder) *Discoverer {
	return &Discoverer{
		path:   path,
		finder: finder,
	}
}

// Discover finds all skills across all sources in the path.
// Skills are returned sorted by precedence (source priority), then by namespace, then alphabetically.
// Skills that are overshadowed by higher-priority sources are marked as such.
func (d *Discoverer) Discover() ([]Skill, error) {
	// Track skills we've seen (by namespace:name) and their paths
	seen := make(map[string]string) // key -> path of highest-priority version
	var allSkills []Skill

	// Iterate through sources in priority order
	sources := d.path.Sources()
	for _, source := range sources {
		skills, err := d.finder.Find(source)
		if err != nil {
			return nil, err
		}

		for _, skill := range skills {
			key := skill.Key()
			if existingPath, exists := seen[key]; exists {
				// This skill is overshadowed
				skill.Overshadowed = true
				skill.OvershadowedBy = existingPath
			} else {
				// First time seeing this skill
				seen[key] = skill.Path
			}
			allSkills = append(allSkills, skill)
		}
	}

	// Sort: by source priority, then by namespace, then alphabetically by name
	sort.Slice(allSkills, func(i, j int) bool {
		if allSkills[i].Source.Priority != allSkills[j].Source.Priority {
			return allSkills[i].Source.Priority < allSkills[j].Source.Priority
		}
		if allSkills[i].Namespace != allSkills[j].Namespace {
			return allSkills[i].Namespace < allSkills[j].Namespace
		}
		return allSkills[i].Name < allSkills[j].Name
	})

	return allSkills, nil
}

// DiscoverByName finds all versions of a skill with the given name across all sources.
// This is useful for debugging to see all locations where a skill is defined.
func (d *Discoverer) DiscoverByName(name string) ([]Skill, error) {
	allSkills, err := d.Discover()
	if err != nil {
		return nil, err
	}

	var matches []Skill
	for _, skill := range allSkills {
		if skill.Name == name {
			matches = append(matches, skill)
		}
	}
	return matches, nil
}

// RelativePath returns a display-friendly relative path for the skill.
// It tries to make the path relative to common reference points.
func RelativePath(skill Skill) string {
	// Try to make relative to home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		if rel, err := filepath.Rel(homeDir, skill.Path); err == nil {
			if len(rel) < len(skill.Path) {
				return "~/" + rel
			}
		}
	}

	// Try to make relative to current directory
	wd, err := os.Getwd()
	if err == nil {
		if rel, err := filepath.Rel(wd, skill.Path); err == nil {
			if len(rel) < len(skill.Path) && rel[0] != '.' {
				return "./" + rel
			} else if len(rel) < len(skill.Path) {
				return rel
			}
		}
	}

	return skill.Path
}
