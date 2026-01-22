// Package converter converts commands to skills.
package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/martinemde/skillet/internal/command"
	"github.com/martinemde/skillet/internal/frontmatter"
	"gopkg.in/yaml.v3"
)

// Config holds the configuration for a command-to-skill conversion.
type Config struct {
	// CommandPath is the resolved absolute path to the command file
	CommandPath string
	// OutputDir is a custom output directory (overrides default)
	// If empty, uses the same scope as the command (project/user)
	OutputDir string
	// Model overrides the model setting (from --model flag)
	Model string
	// AllowedTools overrides the allowed-tools setting (from --allowed-tools flag)
	AllowedTools string
	// Force overwrites existing skill without error
	Force bool
}

// Result contains the outcome of a conversion.
type Result struct {
	// SkillPath is where the skill was written
	SkillPath string
	// SkillName is the name of the skill
	SkillName string
	// Namespace is the preserved namespace (if any)
	Namespace string
	// Guidance is a list of instructions for the user
	Guidance []string
	// AppliedFields lists frontmatter fields that were set
	AppliedFields map[string]string
}

// skillFrontmatter represents the YAML frontmatter for a skill.
// We use a separate struct to control serialization order and omit empty fields.
type skillFrontmatter struct {
	Name                   string `yaml:"name"`
	Description            string `yaml:"description"`
	AllowedTools           string `yaml:"allowed-tools,omitempty"`
	Model                  string `yaml:"model,omitempty"`
	ArgumentHint           string `yaml:"argument-hint,omitempty"`
	Context                string `yaml:"context,omitempty"`
	Agent                  string `yaml:"agent,omitempty"`
	DisableModelInvocation bool   `yaml:"disable-model-invocation,omitempty"`
	UserInvocable          *bool  `yaml:"user-invocable,omitempty"`
}

// Convert converts a command to a skill.
func Convert(cfg Config) (*Result, error) {
	// Parse the command file
	cmd, err := command.Parse(cfg.CommandPath, "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}

	// Build skill frontmatter from command
	fm := buildFrontmatter(cmd, cfg)

	// Determine output path
	outputPath, namespace, err := determineOutputPath(cfg, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to determine output path: %w", err)
	}

	// Check if skill already exists
	if !cfg.Force {
		if _, err := os.Stat(outputPath); err == nil {
			return nil, fmt.Errorf("skill already exists at %s (use --force to overwrite)", outputPath)
		}
	}

	// Get the raw content (without interpolation)
	rawContent, err := getRawContent(cfg.CommandPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read raw content: %w", err)
	}

	// Write the skill file
	if err := writeSkill(outputPath, fm, rawContent); err != nil {
		return nil, fmt.Errorf("failed to write skill: %w", err)
	}

	// Build result
	result := &Result{
		SkillPath:     outputPath,
		SkillName:     fm.Name,
		Namespace:     namespace,
		AppliedFields: make(map[string]string),
		Guidance:      []string{},
	}

	// Track applied fields
	result.AppliedFields["name"] = fm.Name
	if fm.Description != "" {
		if cmd.Description != "" && cmd.Description == fm.Description {
			result.AppliedFields["description"] = "(preserved from command)"
		} else {
			result.AppliedFields["description"] = "(extracted from content)"
		}
	}
	if fm.AllowedTools != "" {
		result.AppliedFields["allowed-tools"] = fm.AllowedTools
	}
	if fm.Model != "" {
		result.AppliedFields["model"] = fm.Model
	}

	// Build guidance
	result.Guidance = buildGuidance(cfg, cmd, fm, rawContent)

	return result, nil
}

// buildFrontmatter creates the skill frontmatter from command data and config.
func buildFrontmatter(cmd *command.Command, cfg Config) skillFrontmatter {
	fm := skillFrontmatter{
		Name:                   cmd.Name,
		Description:            cmd.Description,
		AllowedTools:           cmd.AllowedTools,
		Model:                  cmd.Model,
		ArgumentHint:           cmd.ArgumentHint,
		Context:                cmd.Context,
		Agent:                  cmd.Agent,
		DisableModelInvocation: cmd.DisableModelInvocation,
	}

	// Apply CLI flag overrides
	if cfg.Model != "" {
		fm.Model = cfg.Model
	}
	if cfg.AllowedTools != "" {
		fm.AllowedTools = cfg.AllowedTools
	}

	// Ensure description is set
	if fm.Description == "" {
		fm.Description = "Converted from command: " + cmd.Name
	}

	return fm
}

// determineOutputPath figures out where to write the skill.
func determineOutputPath(cfg Config, cmd *command.Command) (string, string, error) {
	var baseDir string
	namespace := ""

	if cfg.OutputDir != "" {
		// Custom output directory specified
		baseDir = cfg.OutputDir
	} else {
		// Determine scope from command path
		// If command is in .claude/commands/, output to .claude/skills/
		// Extract the root directory (project or home)
		cmdDir := filepath.Dir(cfg.CommandPath)

		// Walk up to find .claude/commands
		for dir := cmdDir; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			if filepath.Base(dir) == "commands" && filepath.Base(filepath.Dir(dir)) == ".claude" {
				// Found .claude/commands, use .claude/skills instead
				baseDir = filepath.Join(filepath.Dir(dir), "skills")
				// Calculate namespace from the path between commands/ and the file
				relPath, err := filepath.Rel(dir, cmdDir)
				if err == nil && relPath != "." && relPath != "" {
					namespace = relPath
				}
				break
			}
		}

		if baseDir == "" {
			// Fallback: use .claude/skills in current directory
			cwd, err := os.Getwd()
			if err != nil {
				return "", "", fmt.Errorf("failed to get current directory: %w", err)
			}
			baseDir = filepath.Join(cwd, ".claude", "skills")
		}
	}

	// Build full path: baseDir/[namespace/]name/SKILL.md
	skillDir := baseDir
	if namespace != "" {
		skillDir = filepath.Join(skillDir, namespace)
	}
	skillDir = filepath.Join(skillDir, cmd.Name)

	return filepath.Join(skillDir, "SKILL.md"), namespace, nil
}

// getRawContent reads the command file and returns the content without variable interpolation.
func getRawContent(commandPath string) (string, error) {
	data, err := os.ReadFile(commandPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	result, err := frontmatter.Parse(string(data), false)
	if err != nil {
		return "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	return result.Content, nil
}

// writeSkill writes the skill file with frontmatter and content.
func writeSkill(path string, fm skillFrontmatter, content string) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Serialize frontmatter
	fmYAML, err := yaml.Marshal(fm)
	if err != nil {
		return fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Build file content
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(fmYAML)
	sb.WriteString("---\n")
	if content != "" {
		sb.WriteString("\n")
		sb.WriteString(content)
		// Ensure trailing newline
		if !strings.HasSuffix(content, "\n") {
			sb.WriteString("\n")
		}
	}

	// Write file
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// buildGuidance creates user guidance based on the conversion.
func buildGuidance(cfg Config, cmd *command.Command, fm skillFrontmatter, content string) []string {
	var guidance []string

	// Description guidance
	if fm.Description == "Converted from command: "+cmd.Name {
		guidance = append(guidance, "description - add a description explaining when Claude should use this skill")
	} else {
		guidance = append(guidance, "description - enhance to explain when Claude should use this skill")
	}

	// Check for $ARGUMENTS usage - suggest argument-hint if content uses $ARGUMENTS
	if strings.Contains(content, "$ARGUMENTS") && fm.ArgumentHint == "" {
		guidance = append(guidance, "argument-hint - add suggested command argument hint")
	}

	// user-invocable guidance
	guidance = append(guidance, "user-invocable: true - set to false for model only invocation")

	// disable-model-invocation guidance
	guidance = append(guidance, "disable-model-invocation: false - set to true for /command use only")

	// Suggest removing original command
	guidance = append(guidance, fmt.Sprintf("rm %s - remove original command if no longer needed", cfg.CommandPath))

	return guidance
}
