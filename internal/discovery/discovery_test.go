package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/martinemde/skillet/internal/skillpath"
)

// createSkillDir creates a skill directory with SKILL.md file.
// If namespace is non-empty, creates baseDir/namespace/skillName/SKILL.md
// Otherwise creates baseDir/skillName/SKILL.md
func createSkillDir(t *testing.T, baseDir, skillName, namespace string) string {
	t.Helper()
	var skillDir string
	if namespace != "" {
		skillDir = filepath.Join(baseDir, namespace, skillName)
	} else {
		skillDir = filepath.Join(baseDir, skillName)
	}
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}
	skillFile := filepath.Join(skillDir, skillpath.SkillFile)
	if err := os.WriteFile(skillFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create skill file: %v", err)
	}
	return skillDir
}

func TestDirectoryFinder_Find(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, skillpath.ClaudeDir, skillpath.SkillsDir)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skills directory: %v", err)
	}

	// Create some test skills
	createSkillDir(t, skillsDir, "alpha-skill", "")
	createSkillDir(t, skillsDir, "beta-skill", "")
	createSkillDir(t, skillsDir, "gamma-skill", "")

	// Create a non-skill directory (no SKILL.md)
	nonSkillDir := filepath.Join(skillsDir, "not-a-skill")
	if err := os.MkdirAll(nonSkillDir, 0755); err != nil {
		t.Fatalf("failed to create non-skill directory: %v", err)
	}

	// Create a file (not a directory)
	if err := os.WriteFile(filepath.Join(skillsDir, "random-file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	finder := &DirectoryFinder{}
	source := skillpath.Source{
		Path:     skillsDir,
		Name:     "test",
		Priority: 0,
	}

	skills, err := finder.Find(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 3 {
		t.Errorf("expected 3 skills, got %d", len(skills))
	}

	// Check that all expected skills were found
	foundSkills := make(map[string]bool)
	for _, skill := range skills {
		foundSkills[skill.Name] = true
	}

	for _, expected := range []string{"alpha-skill", "beta-skill", "gamma-skill"} {
		if !foundSkills[expected] {
			t.Errorf("expected to find skill %s", expected)
		}
	}

	// Should not have found non-skill directory or file
	if foundSkills["not-a-skill"] {
		t.Error("should not have found non-skill directory")
	}
	if foundSkills["random-file.txt"] {
		t.Error("should not have found file as skill")
	}
}

func TestDirectoryFinder_Find_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, skillpath.ClaudeDir, skillpath.SkillsDir)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skills directory: %v", err)
	}

	finder := &DirectoryFinder{}
	source := skillpath.Source{
		Path:     skillsDir,
		Name:     "test",
		Priority: 0,
	}

	skills, err := finder.Find(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestDirectoryFinder_Find_NonExistentDirectory(t *testing.T) {
	finder := &DirectoryFinder{}
	source := skillpath.Source{
		Path:     "/non/existent/path",
		Name:     "test",
		Priority: 0,
	}

	skills, err := finder.Find(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("expected 0 skills for non-existent directory, got %d", len(skills))
	}
}

func TestDiscoverer_Discover(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project-scoped skills
	projectSkillsDir := filepath.Join(tmpDir, "project", skillpath.ClaudeDir, skillpath.SkillsDir)
	if err := os.MkdirAll(projectSkillsDir, 0755); err != nil {
		t.Fatalf("failed to create project skills directory: %v", err)
	}
	createSkillDir(t, projectSkillsDir, "common-skill", "")
	createSkillDir(t, projectSkillsDir, "project-only", "")

	// Create user-scoped skills
	userSkillsDir := filepath.Join(tmpDir, "user", skillpath.ClaudeDir, skillpath.SkillsDir)
	if err := os.MkdirAll(userSkillsDir, 0755); err != nil {
		t.Fatalf("failed to create user skills directory: %v", err)
	}
	createSkillDir(t, userSkillsDir, "common-skill", "") // This should be overshadowed
	createSkillDir(t, userSkillsDir, "user-only", "")

	// Create custom sources
	sources := []skillpath.Source{
		{Path: projectSkillsDir, Name: "project", Priority: 0},
		{Path: userSkillsDir, Name: "user", Priority: 1},
	}
	path := skillpath.NewWithSources(sources)
	disc := New(path)

	skills, err := disc.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 4 {
		t.Errorf("expected 4 skills, got %d", len(skills))
	}

	// Verify ordering (by priority, then alphabetically)
	expectedOrder := []string{"common-skill", "project-only", "common-skill", "user-only"}
	for i, skill := range skills {
		if skill.Name != expectedOrder[i] {
			t.Errorf("expected skill %d to be %s, got %s", i, expectedOrder[i], skill.Name)
		}
	}

	// Check overshadowed status
	for _, skill := range skills {
		if skill.Name == "common-skill" {
			if skill.Source.Name == "user" && !skill.Overshadowed {
				t.Error("expected user common-skill to be overshadowed")
			}
			if skill.Source.Name == "project" && skill.Overshadowed {
				t.Error("expected project common-skill to not be overshadowed")
			}
		}
	}
}

func TestDiscoverer_Discover_Sorting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skills in non-alphabetical order
	skillsDir := filepath.Join(tmpDir, skillpath.ClaudeDir, skillpath.SkillsDir)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skills directory: %v", err)
	}

	createSkillDir(t, skillsDir, "zebra", "")
	createSkillDir(t, skillsDir, "alpha", "")
	createSkillDir(t, skillsDir, "middle", "")

	sources := []skillpath.Source{
		{Path: skillsDir, Name: "test", Priority: 0},
	}
	path := skillpath.NewWithSources(sources)
	disc := New(path)

	skills, err := disc.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be sorted alphabetically within the same priority
	expectedOrder := []string{"alpha", "middle", "zebra"}
	for i, skill := range skills {
		if skill.Name != expectedOrder[i] {
			t.Errorf("expected skill %d to be %s, got %s", i, expectedOrder[i], skill.Name)
		}
	}
}

func TestDiscoverer_DiscoverByName(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skills in two sources
	source1Dir := filepath.Join(tmpDir, "source1")
	source2Dir := filepath.Join(tmpDir, "source2")
	if err := os.MkdirAll(source1Dir, 0755); err != nil {
		t.Fatalf("failed to create source1 directory: %v", err)
	}
	if err := os.MkdirAll(source2Dir, 0755); err != nil {
		t.Fatalf("failed to create source2 directory: %v", err)
	}

	createSkillDir(t, source1Dir, "target-skill", "")
	createSkillDir(t, source1Dir, "other-skill", "")
	createSkillDir(t, source2Dir, "target-skill", "")

	sources := []skillpath.Source{
		{Path: source1Dir, Name: "source1", Priority: 0},
		{Path: source2Dir, Name: "source2", Priority: 1},
	}
	path := skillpath.NewWithSources(sources)
	disc := New(path)

	skills, err := disc.DiscoverByName("target-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}

	// First should be from source1, second from source2
	if skills[0].Source.Name != "source1" {
		t.Errorf("expected first skill to be from source1, got %s", skills[0].Source.Name)
	}
	if skills[1].Source.Name != "source2" {
		t.Errorf("expected second skill to be from source2, got %s", skills[1].Source.Name)
	}

	// Second should be overshadowed
	if !skills[1].Overshadowed {
		t.Error("expected second skill to be overshadowed")
	}
}

func TestRelativePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("could not get home directory")
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Skip("could not get working directory")
	}

	tests := []struct {
		name     string
		skill    Skill
		contains string
	}{
		{
			name: "home directory path",
			skill: Skill{
				Path: filepath.Join(homeDir, ".claude", "skills", "test", "SKILL.md"),
			},
			contains: "~",
		},
		{
			name: "working directory path",
			skill: Skill{
				Path: filepath.Join(wd, ".claude", "skills", "test", "SKILL.md"),
			},
			contains: ".claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RelativePath(tt.skill)
			if len(result) == 0 {
				t.Error("expected non-empty result")
			}
			// The result should be shorter or equal to the original path
			// (unless there's no good relative path available)
		})
	}
}

// MockFinder is a custom finder for testing the Finder interface
type MockFinder struct {
	skills []Skill
	err    error
}

func (f *MockFinder) Find(_ skillpath.Source) ([]Skill, error) {
	return f.skills, f.err
}

func TestNewWithFinder(t *testing.T) {
	mockSkills := []Skill{
		{Name: "mock-skill-1"},
		{Name: "mock-skill-2"},
	}
	mockFinder := &MockFinder{skills: mockSkills}

	sources := []skillpath.Source{
		{Path: "/test", Name: "test", Priority: 0},
	}
	path := skillpath.NewWithSources(sources)
	disc := NewWithFinder(path, mockFinder)

	skills, err := disc.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("expected 2 skills from mock finder, got %d", len(skills))
	}
}

func TestDiscoverer_OvershadowedBy(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the same skill in two sources
	source1Dir := filepath.Join(tmpDir, "source1")
	source2Dir := filepath.Join(tmpDir, "source2")
	if err := os.MkdirAll(source1Dir, 0755); err != nil {
		t.Fatalf("failed to create source1 directory: %v", err)
	}
	if err := os.MkdirAll(source2Dir, 0755); err != nil {
		t.Fatalf("failed to create source2 directory: %v", err)
	}

	createSkillDir(t, source1Dir, "shared-skill", "")
	createSkillDir(t, source2Dir, "shared-skill", "")

	sources := []skillpath.Source{
		{Path: source1Dir, Name: "high-priority", Priority: 0},
		{Path: source2Dir, Name: "low-priority", Priority: 1},
	}
	path := skillpath.NewWithSources(sources)
	disc := New(path)

	skills, err := disc.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the overshadowed skill
	var overshadowed *Skill
	for i := range skills {
		if skills[i].Overshadowed {
			overshadowed = &skills[i]
			break
		}
	}

	if overshadowed == nil {
		t.Fatal("expected to find an overshadowed skill")
	}

	// OvershadowedBy should point to the higher priority skill's path
	expectedPath := filepath.Join(source1Dir, "shared-skill", skillpath.SkillFile)
	if overshadowed.OvershadowedBy != expectedPath {
		t.Errorf("expected OvershadowedBy to be %s, got %s", expectedPath, overshadowed.OvershadowedBy)
	}
}

func TestDirectoryFinder_Find_NamespacedSkills(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, skillpath.ClaudeDir, skillpath.SkillsDir)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skills directory: %v", err)
	}

	// Create unnamespaced skill
	createSkillDir(t, skillsDir, "unnamespaced-skill", "")

	// Create namespaced skills
	createSkillDir(t, skillsDir, "test-skill", "frontend")
	createSkillDir(t, skillsDir, "test-skill", "backend")
	createSkillDir(t, skillsDir, "other-skill", "frontend")

	finder := &DirectoryFinder{}
	source := skillpath.Source{
		Path:     skillsDir,
		Name:     "test",
		Priority: 0,
	}

	skills, err := finder.Find(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 4 {
		t.Errorf("expected 4 skills, got %d", len(skills))
	}

	// Check that all skills were found with correct namespaces
	foundSkills := make(map[string]Skill)
	for _, skill := range skills {
		foundSkills[skill.Key()] = skill
	}

	// Check unnamespaced skill
	if skill, ok := foundSkills["unnamespaced-skill"]; !ok {
		t.Error("expected to find unnamespaced-skill")
	} else if skill.Namespace != "" {
		t.Errorf("expected unnamespaced-skill to have empty namespace, got %q", skill.Namespace)
	}

	// Check frontend:test-skill
	if skill, ok := foundSkills["frontend:test-skill"]; !ok {
		t.Error("expected to find frontend:test-skill")
	} else if skill.Namespace != "frontend" {
		t.Errorf("expected frontend:test-skill to have namespace 'frontend', got %q", skill.Namespace)
	}

	// Check backend:test-skill
	if skill, ok := foundSkills["backend:test-skill"]; !ok {
		t.Error("expected to find backend:test-skill")
	} else if skill.Namespace != "backend" {
		t.Errorf("expected backend:test-skill to have namespace 'backend', got %q", skill.Namespace)
	}

	// Check frontend:other-skill
	if skill, ok := foundSkills["frontend:other-skill"]; !ok {
		t.Error("expected to find frontend:other-skill")
	} else if skill.Namespace != "frontend" {
		t.Errorf("expected frontend:other-skill to have namespace 'frontend', got %q", skill.Namespace)
	}
}

func TestSkill_QualifiedName(t *testing.T) {
	tests := []struct {
		name     string
		skill    Skill
		expected string
	}{
		{
			name:     "unnamespaced",
			skill:    Skill{Name: "test", Namespace: ""},
			expected: "test",
		},
		{
			name:     "namespaced",
			skill:    Skill{Name: "test", Namespace: "frontend"},
			expected: "frontend:test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.skill.QualifiedName(); got != tt.expected {
				t.Errorf("QualifiedName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSkill_Key(t *testing.T) {
	tests := []struct {
		name     string
		skill    Skill
		expected string
	}{
		{
			name:     "unnamespaced",
			skill:    Skill{Name: "test", Namespace: ""},
			expected: "test",
		},
		{
			name:     "namespaced",
			skill:    Skill{Name: "test", Namespace: "frontend"},
			expected: "frontend:test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.skill.Key(); got != tt.expected {
				t.Errorf("Key() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDiscoverer_Discover_NamespacedOvershadowing(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project-scoped skills (higher priority)
	projectSkillsDir := filepath.Join(tmpDir, "project", skillpath.ClaudeDir, skillpath.SkillsDir)
	if err := os.MkdirAll(projectSkillsDir, 0755); err != nil {
		t.Fatalf("failed to create project skills directory: %v", err)
	}
	createSkillDir(t, projectSkillsDir, "common-skill", "frontend")

	// Create user-scoped skills (lower priority)
	userSkillsDir := filepath.Join(tmpDir, "user", skillpath.ClaudeDir, skillpath.SkillsDir)
	if err := os.MkdirAll(userSkillsDir, 0755); err != nil {
		t.Fatalf("failed to create user skills directory: %v", err)
	}
	createSkillDir(t, userSkillsDir, "common-skill", "frontend") // Should be overshadowed
	createSkillDir(t, userSkillsDir, "common-skill", "backend")  // Different namespace, not overshadowed

	sources := []skillpath.Source{
		{Path: projectSkillsDir, Name: "project", Priority: 0},
		{Path: userSkillsDir, Name: "user", Priority: 1},
	}
	path := skillpath.NewWithSources(sources)
	disc := New(path)

	skills, err := disc.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 3 {
		t.Errorf("expected 3 skills, got %d", len(skills))
	}

	// Check that only frontend:common-skill from user is overshadowed
	for _, skill := range skills {
		if skill.Key() == "frontend:common-skill" && skill.Source.Name == "user" {
			if !skill.Overshadowed {
				t.Error("expected user frontend:common-skill to be overshadowed")
			}
		}
		if skill.Key() == "frontend:common-skill" && skill.Source.Name == "project" {
			if skill.Overshadowed {
				t.Error("expected project frontend:common-skill to not be overshadowed")
			}
		}
		if skill.Key() == "backend:common-skill" {
			if skill.Overshadowed {
				t.Error("expected backend:common-skill to not be overshadowed (different namespace)")
			}
		}
	}
}

func TestDiscoverer_Discover_NamespaceSorting(t *testing.T) {
	tmpDir := t.TempDir()

	skillsDir := filepath.Join(tmpDir, skillpath.ClaudeDir, skillpath.SkillsDir)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skills directory: %v", err)
	}

	// Create skills with various namespaces
	createSkillDir(t, skillsDir, "zebra", "")         // unnamespaced
	createSkillDir(t, skillsDir, "alpha", "")         // unnamespaced
	createSkillDir(t, skillsDir, "test", "frontend")  // frontend:test
	createSkillDir(t, skillsDir, "test", "backend")   // backend:test
	createSkillDir(t, skillsDir, "alpha", "frontend") // frontend:alpha

	sources := []skillpath.Source{
		{Path: skillsDir, Name: "test", Priority: 0},
	}
	path := skillpath.NewWithSources(sources)
	disc := New(path)

	skills, err := disc.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be sorted: unnamespaced first (alphabetically), then by namespace, then by name
	// Expected order: alpha, zebra, backend:test, frontend:alpha, frontend:test
	expectedOrder := []string{"alpha", "zebra", "backend:test", "frontend:alpha", "frontend:test"}
	for i, skill := range skills {
		if skill.Key() != expectedOrder[i] {
			t.Errorf("expected skill %d to be %s, got %s", i, expectedOrder[i], skill.Key())
		}
	}
}
