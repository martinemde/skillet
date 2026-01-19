// Package discovery provides skill discovery functionality.
// It can find all available skills across multiple search paths,
// with support for precedence and detecting overshadowed skills.
package discovery

import (
	"os"
	"path/filepath"
	"sort"

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
	// Overshadowed indicates this skill is hidden by a higher-priority skill
	Overshadowed bool
	// OvershadowedBy is the path of the skill that shadows this one
	OvershadowedBy string
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

// Find discovers skills in the given source using the directory pattern
func (f *DirectoryFinder) Find(source skillpath.Source) ([]Skill, error) {
	var skills []Skill

	// Check if the source directory exists
	if _, err := os.Stat(source.Path); os.IsNotExist(err) {
		return skills, nil
	}

	// Read the source directory
	entries, err := os.ReadDir(source.Path)
	if err != nil {
		// If we can't read the directory, just return empty
		return skills, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillName := entry.Name()
		skillPath := skillpath.SkillPath(source.Path, skillName)

		// Check if SKILL.md exists
		if _, err := os.Stat(skillPath); err == nil {
			skills = append(skills, Skill{
				Name:   skillName,
				Path:   skillPath,
				Source: source,
			})
		}
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
// Skills are returned sorted by precedence (source priority), then alphabetically.
// Skills that are overshadowed by higher-priority sources are marked as such.
func (d *Discoverer) Discover() ([]Skill, error) {
	// Track skills we've seen (by name) and their paths
	seen := make(map[string]string) // name -> path of highest-priority version
	var allSkills []Skill

	// Iterate through sources in priority order
	sources := d.path.Sources()
	for _, source := range sources {
		skills, err := d.finder.Find(source)
		if err != nil {
			return nil, err
		}

		for _, skill := range skills {
			if existingPath, exists := seen[skill.Name]; exists {
				// This skill is overshadowed
				skill.Overshadowed = true
				skill.OvershadowedBy = existingPath
			} else {
				// First time seeing this skill
				seen[skill.Name] = skill.Path
			}
			allSkills = append(allSkills, skill)
		}
	}

	// Sort: by source priority, then alphabetically by name
	sort.Slice(allSkills, func(i, j int) bool {
		if allSkills[i].Source.Priority != allSkills[j].Source.Priority {
			return allSkills[i].Source.Priority < allSkills[j].Source.Priority
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
