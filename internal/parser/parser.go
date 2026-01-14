package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a parsed SKILL.md file
type Skill struct {
	// Frontmatter fields
	Name                   string            `yaml:"name"`
	Description            string            `yaml:"description"`
	License                string            `yaml:"license,omitempty"`
	Compatibility          string            `yaml:"compatibility,omitempty"`
	Metadata               map[string]string `yaml:"metadata,omitempty"`
	AllowedTools           string            `yaml:"allowed-tools,omitempty"`
	Model                  string            `yaml:"model,omitempty"`
	Version                string            `yaml:"version,omitempty"`
	DisableModelInvocation bool              `yaml:"disable-model-invocation,omitempty"`
	Mode                   bool              `yaml:"mode,omitempty"`

	// Parsed content
	Content string
	BaseDir string // Directory containing the SKILL.md file
}

// Parse reads and parses a SKILL.md file
func Parse(skillPath string) (*Skill, error) {
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

	// Get base directory
	baseDir := filepath.Dir(absPath)

	// Parse frontmatter and content
	skill, err := parseFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	skill.BaseDir = baseDir

	// Interpolate variables
	skill.Content = interpolateVariables(skill.Content, baseDir)

	// Validate required fields
	if err := skill.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return skill, nil
}

// parseFrontmatter extracts YAML frontmatter and content from the file
func parseFrontmatter(data string) (*Skill, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))

	var inFrontmatter bool
	var frontmatterLines []string
	var contentLines []string
	var frontmatterCount int

	for scanner.Scan() {
		line := scanner.Text()

		// Check for frontmatter delimiters
		if strings.TrimSpace(line) == "---" {
			frontmatterCount++
			if frontmatterCount == 1 {
				inFrontmatter = true
				continue
			} else if frontmatterCount == 2 {
				inFrontmatter = false
				continue
			}
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else if frontmatterCount >= 2 {
			contentLines = append(contentLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	if frontmatterCount < 2 {
		return nil, fmt.Errorf("invalid frontmatter: expected two '---' delimiters, found %d", frontmatterCount)
	}

	// Parse YAML frontmatter
	skill := &Skill{}
	frontmatterYAML := strings.Join(frontmatterLines, "\n")
	if err := yaml.Unmarshal([]byte(frontmatterYAML), skill); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// Set content
	skill.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))

	return skill, nil
}

// interpolateVariables replaces variables like {baseDir} with actual values
func interpolateVariables(content, baseDir string) string {
	// Replace {baseDir} with the actual base directory
	re := regexp.MustCompile(`\{baseDir\}`)
	return re.ReplaceAllString(content, baseDir)
}

// Validate checks that required fields are present and valid
func (s *Skill) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Validate name format
	nameRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	if !nameRegex.MatchString(s.Name) {
		return fmt.Errorf("invalid name format: must be lowercase letters, numbers, and hyphens, not starting/ending with hyphen")
	}

	if len(s.Name) > 64 {
		return fmt.Errorf("name too long: max 64 characters, got %d", len(s.Name))
	}

	if strings.Contains(s.Name, "--") {
		return fmt.Errorf("name cannot contain consecutive hyphens")
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
