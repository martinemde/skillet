package parser

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParse_SimpleSkill(t *testing.T) {
	skill, err := Parse("../../testdata/simple-skill/SKILL.md")
	if err != nil {
		t.Fatalf("Failed to parse simple skill: %v", err)
	}

	if skill.Name != "simple-skill" {
		t.Errorf("Expected name 'simple-skill', got '%s'", skill.Name)
	}

	expectedDesc := "A simple skill for testing basic functionality. Use when you need to test the skillet CLI."
	if skill.Description != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, skill.Description)
	}

	if !strings.Contains(skill.Content, "Simple Skill") {
		t.Error("Content should contain 'Simple Skill' heading")
	}

	if skill.BaseDir == "" {
		t.Error("BaseDir should not be empty")
	}
}

func TestParse_ComprehensiveSkill(t *testing.T) {
	skill, err := Parse("../../testdata/comprehensive-skill/SKILL.md")
	if err != nil {
		t.Fatalf("Failed to parse comprehensive skill: %v", err)
	}

	// Test required fields
	if skill.Name != "comprehensive-skill" {
		t.Errorf("Expected name 'comprehensive-skill', got '%s'", skill.Name)
	}

	if skill.Description == "" {
		t.Error("Description should not be empty")
	}

	// Test optional fields
	if skill.License != "Apache-2.0" {
		t.Errorf("Expected license 'Apache-2.0', got '%s'", skill.License)
	}

	if skill.Compatibility != "Requires git, docker, and access to the internet" {
		t.Errorf("Unexpected compatibility: %s", skill.Compatibility)
	}

	if skill.AllowedTools != "Read Write Bash(git:*) Bash(docker:*)" {
		t.Errorf("Unexpected allowed-tools: %s", skill.AllowedTools)
	}

	if skill.Model != "claude-sonnet-4-5-20250929" {
		t.Errorf("Expected model 'claude-sonnet-4-5-20250929', got '%s'", skill.Model)
	}

	// Test metadata
	if skill.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	if skill.Metadata["author"] != "test-org" {
		t.Errorf("Expected metadata.author 'test-org', got '%s'", skill.Metadata["author"])
	}

	if skill.Metadata["version"] != "1.0.0" {
		t.Errorf("Expected metadata.version '1.0.0', got '%s'", skill.Metadata["version"])
	}

	if skill.Metadata["category"] != "testing" {
		t.Errorf("Expected metadata.category 'testing', got '%s'", skill.Metadata["category"])
	}

	// Test content
	if !strings.Contains(skill.Content, "Comprehensive Skill") {
		t.Error("Content should contain 'Comprehensive Skill' heading")
	}
}

func TestParse_InterpolationSkill(t *testing.T) {
	skill, err := Parse("../../testdata/interpolation-skill/SKILL.md")
	if err != nil {
		t.Fatalf("Failed to parse interpolation skill: %v", err)
	}

	// Check that {baseDir} has been interpolated
	if strings.Contains(skill.Content, "{baseDir}") {
		t.Error("Content should not contain '{baseDir}' - it should be interpolated")
	}

	// The content should contain the actual base directory path
	expectedPath, _ := filepath.Abs("../../testdata/interpolation-skill")
	if !strings.Contains(skill.Content, expectedPath) {
		t.Errorf("Content should contain interpolated base directory path '%s'", expectedPath)
	}

	// Check specific interpolations
	if !strings.Contains(skill.Content, expectedPath+"/config.json") {
		t.Errorf("Content should contain '%s/config.json'", expectedPath)
	}

	if !strings.Contains(skill.Content, expectedPath+"/references/data.txt") {
		t.Errorf("Content should contain '%s/references/data.txt'", expectedPath)
	}
}

func TestParse_InvalidName(t *testing.T) {
	_, err := Parse("../../testdata/invalid-skill/SKILL.md")
	if err == nil {
		t.Fatal("Expected error for invalid skill name, got nil")
	}

	if !strings.Contains(err.Error(), "invalid name format") {
		t.Errorf("Expected 'invalid name format' error, got: %v", err)
	}
}

func TestParse_NoFrontmatter(t *testing.T) {
	_, err := Parse("../../testdata/no-frontmatter/SKILL.md")
	if err == nil {
		t.Fatal("Expected error for missing frontmatter, got nil")
	}

	if !strings.Contains(err.Error(), "frontmatter") {
		t.Errorf("Expected frontmatter error, got: %v", err)
	}
}

func TestParse_NonexistentFile(t *testing.T) {
	_, err := Parse("../../testdata/nonexistent/SKILL.md")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
}

func TestValidate_NameValidation(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid lowercase",
			skillName: "my-skill",
			wantErr:   false,
		},
		{
			name:      "valid with numbers",
			skillName: "skill-123",
			wantErr:   false,
		},
		{
			name:      "valid single character",
			skillName: "a",
			wantErr:   false,
		},
		{
			name:      "invalid uppercase",
			skillName: "My-Skill",
			wantErr:   true,
			errMsg:    "invalid name format",
		},
		{
			name:      "invalid starting with hyphen",
			skillName: "-myskill",
			wantErr:   true,
			errMsg:    "invalid name format",
		},
		{
			name:      "invalid ending with hyphen",
			skillName: "myskill-",
			wantErr:   true,
			errMsg:    "invalid name format",
		},
		{
			name:      "invalid consecutive hyphens",
			skillName: "my--skill",
			wantErr:   true,
			errMsg:    "consecutive hyphens",
		},
		{
			name:      "invalid too long",
			skillName: strings.Repeat("a", 65),
			wantErr:   true,
			errMsg:    "name too long",
		},
		{
			name:      "invalid empty",
			skillName: "",
			wantErr:   true,
			errMsg:    "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &Skill{
				Name:        tt.skillName,
				Description: "Test description",
			}
			err := skill.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidate_DescriptionValidation(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid description",
			description: "A valid skill description",
			wantErr:     false,
		},
		{
			name:        "empty description",
			description: "",
			wantErr:     true,
			errMsg:      "description is required",
		},
		{
			name:        "too long description",
			description: strings.Repeat("a", 1025),
			wantErr:     true,
			errMsg:      "description too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &Skill{
				Name:        "valid-name",
				Description: tt.description,
			}
			err := skill.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidate_CompatibilityValidation(t *testing.T) {
	skill := &Skill{
		Name:          "test-skill",
		Description:   "Test description",
		Compatibility: strings.Repeat("a", 501),
	}

	err := skill.Validate()
	if err == nil {
		t.Error("Expected error for compatibility too long")
	}

	if !strings.Contains(err.Error(), "compatibility too long") {
		t.Errorf("Expected 'compatibility too long' error, got: %v", err)
	}
}

func TestInterpolateVariables(t *testing.T) {
	baseDir := "/path/to/skill"
	content := "Base directory is {baseDir} and config is at {baseDir}/config.json"

	result := interpolateVariables(content, baseDir)

	expected := "Base directory is /path/to/skill and config is at /path/to/skill/config.json"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestParse_YAMLSyntaxError(t *testing.T) {
	_, err := Parse("../../testdata/yaml-syntax-error/SKILL.md")
	if err == nil {
		t.Fatal("Expected error for malformed YAML, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse YAML frontmatter") {
		t.Errorf("Expected 'failed to parse YAML frontmatter' error, got: %v", err)
	}
}

func TestParseWithBaseDir_ExplicitBaseDir(t *testing.T) {
	customBaseDir := "/custom/base/directory"
	skill, err := ParseWithBaseDir("../../testdata/interpolation-skill/SKILL.md", customBaseDir)
	if err != nil {
		t.Fatalf("Failed to parse with explicit base dir: %v", err)
	}

	// Verify that the custom base directory is used
	if skill.BaseDir != customBaseDir {
		t.Errorf("Expected BaseDir '%s', got '%s'", customBaseDir, skill.BaseDir)
	}

	// Verify that interpolation uses the custom base directory
	if !strings.Contains(skill.Content, customBaseDir) {
		t.Errorf("Content should contain custom base directory '%s'", customBaseDir)
	}

	if !strings.Contains(skill.Content, customBaseDir+"/config.json") {
		t.Errorf("Content should contain '%s/config.json'", customBaseDir)
	}
}
