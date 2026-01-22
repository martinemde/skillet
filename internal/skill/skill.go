package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/martinemde/skillet/internal/frontmatter"
	"github.com/martinemde/skillet/internal/validation"
	"gopkg.in/yaml.v3"
)

var (
	// argumentsRegex matches $ARGUMENTS variable references
	argumentsRegex = regexp.MustCompile(`\$ARGUMENTS`)
)

// Skill represents a parsed SKILL.md file
type Skill struct {
	// Claude Code spec fields
	Name                   string `yaml:"name,omitempty"`                     // Display name (defaults to directory name)
	Description            string `yaml:"description"`                        // What the skill does and when to use it
	ArgumentHint           string `yaml:"argument-hint,omitempty"`            // Hint shown during autocomplete
	DisableModelInvocation bool   `yaml:"disable-model-invocation,omitempty"` // Prevent automatic loading
	UserInvocable          *bool  `yaml:"user-invocable,omitempty"`           // Show in / menu (default: true)
	AllowedTools           string `yaml:"allowed-tools,omitempty"`            // Tools allowed without permission
	Model                  string `yaml:"model,omitempty"`                    // Model to use
	Context                string `yaml:"context,omitempty"`                  // "fork" for forked subagent
	Agent                  string `yaml:"agent,omitempty"`                    // Subagent type when context: fork
	Hooks                  any    `yaml:"hooks,omitempty"`                    // Hooks scoped to skill lifecycle

	// agentskills.io spec fields
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	Version       string            `yaml:"version,omitempty"`

	// Parsed content
	Content string
	BaseDir string // Directory containing the SKILL.md file
}

// Parse reads and parses a SKILL.md file
func Parse(skillPath string, arguments string) (*Skill, error) {
	return ParseWithBaseDir(skillPath, "", arguments)
}

// ParseWithBaseDir reads and parses a SKILL.md file with an optional custom base directory
// If baseDir is empty, it defaults to the directory containing the skill file
// The arguments string replaces $ARGUMENTS in the skill content
func ParseWithBaseDir(skillPath string, baseDir string, arguments string) (*Skill, error) {
	// Resolve absolute path
	absPath, err := filepath.Abs(skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Get base directory if not provided
	if baseDir == "" {
		baseDir = filepath.Dir(absPath)
	}

	// Parse frontmatter and content
	skill, err := parseFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	skill.BaseDir = baseDir

	// Derive name from directory if not provided
	if skill.Name == "" {
		skill.Name = filepath.Base(baseDir)
	}

	// Interpolate variables
	skill.Content = interpolateVariables(skill.Content, baseDir, arguments)

	// Validate required fields
	if err := skill.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return skill, nil
}

// parseFrontmatter extracts YAML frontmatter and content from the file
func parseFrontmatter(data string) (*Skill, error) {
	result, err := frontmatter.Parse(data, true)
	if err != nil {
		return nil, err
	}

	// Parse YAML frontmatter
	skill := &Skill{}
	if err := yaml.Unmarshal([]byte(result.FrontmatterYAML), skill); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	skill.Content = result.Content
	return skill, nil
}

// interpolateVariables replaces variables like {baseDir} and $ARGUMENTS with actual values
// If $ARGUMENTS is not present in the content and arguments are provided,
// appends "ARGUMENTS: <value>" to the content per the agentskills.io spec.
func interpolateVariables(content, baseDir, arguments string) string {
	content = validation.InterpolateBaseDir(content, baseDir)

	if argumentsRegex.MatchString(content) {
		// $ARGUMENTS is present, replace it with arguments
		content = argumentsRegex.ReplaceAllString(content, arguments)
	} else if arguments != "" {
		// $ARGUMENTS not present but arguments provided, append them
		content = content + "\n\nARGUMENTS: " + arguments
	}

	return content
}

// IsUserInvocable returns whether the skill should appear in the / menu.
// Returns true if UserInvocable is nil (default) or explicitly true.
func (s *Skill) IsUserInvocable() bool {
	return s.UserInvocable == nil || *s.UserInvocable
}

// Validate checks that required fields are present and valid
func (s *Skill) Validate() error {
	if err := validation.ValidateName(s.Name, "skill"); err != nil {
		return err
	}

	if s.Description == "" {
		return fmt.Errorf("description is required")
	}

	if len(s.Description) > 1024 {
		return fmt.Errorf("description too long: max 1024 characters, got %d", len(s.Description))
	}

	if s.Compatibility != "" && len(s.Compatibility) > 500 {
		return fmt.Errorf("compatibility too long: max 500 characters, got %d", len(s.Compatibility))
	}

	return nil
}
