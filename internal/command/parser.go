package command

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	// baseDirRegex matches {baseDir} variable references
	baseDirRegex = regexp.MustCompile(`\{baseDir\}`)
	// nameRegex validates command name format (same as skill names)
	nameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
)

// Command represents a parsed command .md file
type Command struct {
	// Frontmatter fields
	Description            string `yaml:"description,omitempty"`
	AllowedTools           string `yaml:"allowed-tools,omitempty"`
	ArgumentHint           string `yaml:"argument-hint,omitempty"`
	Context                string `yaml:"context,omitempty"` // "fork" for forked sub-agent
	Agent                  string `yaml:"agent,omitempty"`   // Agent type when context: fork
	Model                  string `yaml:"model,omitempty"`
	DisableModelInvocation bool   `yaml:"disable-model-invocation,omitempty"`

	// Derived fields
	Name    string // Derived from filename (without .md)
	Content string // Markdown content after frontmatter
	BaseDir string // Directory containing the command file
}

// Parse reads and parses a command .md file
func Parse(commandPath string) (*Command, error) {
	return ParseWithBaseDir(commandPath, "")
}

// ParseWithBaseDir reads and parses a command .md file with an optional custom base directory
// If baseDir is empty, it defaults to the directory containing the command file
func ParseWithBaseDir(commandPath string, baseDir string) (*Command, error) {
	// Resolve absolute path
	absPath, err := filepath.Abs(commandPath)
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

	// Derive name from filename
	filename := filepath.Base(absPath)
	name := strings.TrimSuffix(filename, ".md")

	// Parse frontmatter and content
	cmd, err := parseFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	cmd.Name = name
	cmd.BaseDir = baseDir

	// Interpolate variables
	cmd.Content = interpolateVariables(cmd.Content, baseDir)

	// If description is not set, use the first non-empty line of content
	if cmd.Description == "" {
		cmd.Description = extractFirstLine(cmd.Content)
	}

	// Validate
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return cmd, nil
}

// parseFrontmatter extracts YAML frontmatter and content from the file
func parseFrontmatter(data string) (*Command, error) {
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
		} else if frontmatterCount == 0 {
			// No frontmatter, everything is content
			contentLines = append(contentLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Parse YAML frontmatter if present
	cmd := &Command{}
	if frontmatterCount >= 2 && len(frontmatterLines) > 0 {
		frontmatterYAML := strings.Join(frontmatterLines, "\n")
		if err := yaml.Unmarshal([]byte(frontmatterYAML), cmd); err != nil {
			return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
		}
	}

	// Set content
	cmd.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))

	return cmd, nil
}

// interpolateVariables replaces variables like {baseDir} with actual values
func interpolateVariables(content, baseDir string) string {
	// Replace {baseDir} with the actual base directory
	return baseDirRegex.ReplaceAllString(content, baseDir)
}

// extractFirstLine gets the first non-empty, non-heading line as a description
func extractFirstLine(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and markdown headings
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Truncate if too long
		if len(line) > 200 {
			return line[:197] + "..."
		}
		return line
	}
	return ""
}

// Validate checks that the command is valid
func (c *Command) Validate() error {
	// Name is derived from filename, validate format
	if c.Name == "" {
		return fmt.Errorf("command name is required")
	}

	// Validate name format
	if !nameRegex.MatchString(c.Name) {
		return fmt.Errorf("invalid command name format: must be lowercase letters, numbers, and hyphens, not starting/ending with hyphen")
	}

	if len(c.Name) > 64 {
		return fmt.Errorf("command name too long: max 64 characters, got %d", len(c.Name))
	}

	if strings.Contains(c.Name, "--") {
		return fmt.Errorf("command name cannot contain consecutive hyphens")
	}

	// Content is required (a command must do something)
	if c.Content == "" {
		return fmt.Errorf("command content is required")
	}

	return nil
}
