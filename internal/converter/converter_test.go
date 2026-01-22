package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/martinemde/skillet/internal/skill"
)

func TestConvert_SimpleCommand(t *testing.T) {
	// Setup: create temp directory for output
	tmpDir := t.TempDir()

	// Get absolute path to test command
	commandPath, err := filepath.Abs("../../testdata/commands/simple-command.md")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cfg := Config{
		CommandPath: commandPath,
		OutputDir:   tmpDir,
	}

	result, err := Convert(cfg)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	// Verify result fields
	if result.SkillName != "simple-command" {
		t.Errorf("Expected skill name 'simple-command', got '%s'", result.SkillName)
	}

	if result.SkillPath == "" {
		t.Error("SkillPath should not be empty")
	}

	// Verify skill file was created
	if _, err := os.Stat(result.SkillPath); os.IsNotExist(err) {
		t.Errorf("Skill file was not created at %s", result.SkillPath)
	}

	// Verify the generated skill is valid by parsing it
	parsedSkill, err := skill.Parse(result.SkillPath, "")
	if err != nil {
		t.Fatalf("Generated skill failed validation: %v", err)
	}

	// Check parsed values
	if parsedSkill.Name != "simple-command" {
		t.Errorf("Parsed skill name mismatch: expected 'simple-command', got '%s'", parsedSkill.Name)
	}

	if parsedSkill.Description != "A simple test command" {
		t.Errorf("Parsed skill description mismatch: expected 'A simple test command', got '%s'", parsedSkill.Description)
	}
}

func TestConvert_ComprehensiveCommand(t *testing.T) {
	tmpDir := t.TempDir()

	commandPath, err := filepath.Abs("../../testdata/commands/comprehensive-command.md")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cfg := Config{
		CommandPath: commandPath,
		OutputDir:   tmpDir,
	}

	result, err := Convert(cfg)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	// Parse the generated skill
	parsedSkill, err := skill.Parse(result.SkillPath, "")
	if err != nil {
		t.Fatalf("Generated skill failed validation: %v", err)
	}

	// Check all frontmatter fields were preserved
	if parsedSkill.AllowedTools != "Read Write Bash(git:*)" {
		t.Errorf("AllowedTools mismatch: got '%s'", parsedSkill.AllowedTools)
	}

	if parsedSkill.Model != "claude-sonnet-4-5-20250929" {
		t.Errorf("Model mismatch: got '%s'", parsedSkill.Model)
	}

	if parsedSkill.ArgumentHint != "[filename] [options]" {
		t.Errorf("ArgumentHint mismatch: got '%s'", parsedSkill.ArgumentHint)
	}

	if !parsedSkill.DisableModelInvocation {
		t.Error("DisableModelInvocation should be true")
	}

	// Check raw content was preserved with $ARGUMENTS (before interpolation)
	rawContent, err := os.ReadFile(result.SkillPath)
	if err != nil {
		t.Fatalf("Failed to read skill file: %v", err)
	}
	if !strings.Contains(string(rawContent), "$ARGUMENTS") {
		t.Error("Raw content should contain $ARGUMENTS (not interpolated)")
	}
}

func TestConvert_WithCLIOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	commandPath, err := filepath.Abs("../../testdata/commands/simple-command.md")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cfg := Config{
		CommandPath:  commandPath,
		OutputDir:    tmpDir,
		Model:        "claude-opus-4-5-20251101",
		AllowedTools: "Read,Write,Bash",
	}

	result, err := Convert(cfg)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	// Parse the generated skill
	parsedSkill, err := skill.Parse(result.SkillPath, "")
	if err != nil {
		t.Fatalf("Generated skill failed validation: %v", err)
	}

	// Check CLI overrides were applied
	if parsedSkill.Model != "claude-opus-4-5-20251101" {
		t.Errorf("Model override not applied: got '%s'", parsedSkill.Model)
	}

	if parsedSkill.AllowedTools != "Read,Write,Bash" {
		t.Errorf("AllowedTools override not applied: got '%s'", parsedSkill.AllowedTools)
	}

	// Check applied fields in result
	if result.AppliedFields["model"] != "claude-opus-4-5-20251101" {
		t.Error("Applied fields should include model")
	}
}

func TestConvert_ExistingSkillError(t *testing.T) {
	tmpDir := t.TempDir()

	commandPath, err := filepath.Abs("../../testdata/commands/simple-command.md")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cfg := Config{
		CommandPath: commandPath,
		OutputDir:   tmpDir,
	}

	// First conversion should succeed
	_, err = Convert(cfg)
	if err != nil {
		t.Fatalf("First convert failed: %v", err)
	}

	// Second conversion should fail without --force
	_, err = Convert(cfg)
	if err == nil {
		t.Error("Expected error when skill already exists")
	}

	if !strings.Contains(err.Error(), "skill already exists") {
		t.Errorf("Expected 'skill already exists' error, got: %v", err)
	}
}

func TestConvert_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	commandPath, err := filepath.Abs("../../testdata/commands/simple-command.md")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cfg := Config{
		CommandPath: commandPath,
		OutputDir:   tmpDir,
	}

	// First conversion
	_, err = Convert(cfg)
	if err != nil {
		t.Fatalf("First convert failed: %v", err)
	}

	// Second conversion with Force should succeed
	cfg.Force = true
	_, err = Convert(cfg)
	if err != nil {
		t.Errorf("Convert with Force should succeed: %v", err)
	}
}

func TestConvert_NamespacedCommand(t *testing.T) {
	tmpDir := t.TempDir()

	commandPath, err := filepath.Abs("../../testdata/commands/frontend/component.md")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cfg := Config{
		CommandPath: commandPath,
		OutputDir:   tmpDir,
	}

	result, err := Convert(cfg)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	// Verify skill name is from filename (not namespace)
	if result.SkillName != "component" {
		t.Errorf("Expected skill name 'component', got '%s'", result.SkillName)
	}

	// Verify the skill was created at correct path
	if _, err := os.Stat(result.SkillPath); os.IsNotExist(err) {
		t.Errorf("Skill file was not created at %s", result.SkillPath)
	}
}

func TestConvert_NoFrontmatterCommand(t *testing.T) {
	tmpDir := t.TempDir()

	commandPath, err := filepath.Abs("../../testdata/commands/no-frontmatter-command.md")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cfg := Config{
		CommandPath: commandPath,
		OutputDir:   tmpDir,
	}

	result, err := Convert(cfg)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	// Parse the generated skill
	parsedSkill, err := skill.Parse(result.SkillPath, "")
	if err != nil {
		t.Fatalf("Generated skill failed validation: %v", err)
	}

	// Name should be from filename
	if parsedSkill.Name != "no-frontmatter-command" {
		t.Errorf("Skill name mismatch: got '%s'", parsedSkill.Name)
	}

	// Description should be set (extracted from content by command parser)
	if parsedSkill.Description == "" {
		t.Error("Description should not be empty")
	}
}

func TestConvert_Guidance(t *testing.T) {
	tmpDir := t.TempDir()

	commandPath, err := filepath.Abs("../../testdata/commands/comprehensive-command.md")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cfg := Config{
		CommandPath: commandPath,
		OutputDir:   tmpDir,
	}

	result, err := Convert(cfg)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	// Verify guidance is provided
	if len(result.Guidance) == 0 {
		t.Error("Expected guidance to be provided")
	}

	// Check for expected guidance items
	hasUserInvocable := false
	hasDisableModelInvocation := false
	for _, g := range result.Guidance {
		if strings.Contains(g, "user-invocable") {
			hasUserInvocable = true
		}
		if strings.Contains(g, "disable-model-invocation") {
			hasDisableModelInvocation = true
		}
	}

	if !hasUserInvocable {
		t.Error("Guidance should include user-invocable")
	}
	if !hasDisableModelInvocation {
		t.Error("Guidance should include disable-model-invocation")
	}
}

func TestConvert_OutputPathStructure(t *testing.T) {
	tmpDir := t.TempDir()

	commandPath, err := filepath.Abs("../../testdata/commands/simple-command.md")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cfg := Config{
		CommandPath: commandPath,
		OutputDir:   tmpDir,
	}

	result, err := Convert(cfg)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	// Verify path structure: outputDir/name/SKILL.md
	expectedPath := filepath.Join(tmpDir, "simple-command", "SKILL.md")
	if result.SkillPath != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, result.SkillPath)
	}
}

func TestDetermineOutputPath_CustomOutputDir(t *testing.T) {
	tmpDir := t.TempDir()

	commandPath, err := filepath.Abs("../../testdata/commands/simple-command.md")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	customOutput := filepath.Join(tmpDir, "custom", "skills")
	cfg := Config{
		CommandPath: commandPath,
		OutputDir:   customOutput,
	}

	result, err := Convert(cfg)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	// Verify custom output directory was used
	if !strings.HasPrefix(result.SkillPath, customOutput) {
		t.Errorf("Expected path to start with '%s', got '%s'", customOutput, result.SkillPath)
	}
}
