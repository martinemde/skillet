package resolver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolve_ExactFilePath(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "test-skill.md")
	if err := os.WriteFile(skillFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Resolve(skillFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsURL {
		t.Error("expected IsURL to be false")
	}

	if result.BaseURL != "" {
		t.Errorf("expected empty BaseURL, got %s", result.BaseURL)
	}

	// Path should be absolute
	if !filepath.IsAbs(result.Path) {
		t.Errorf("expected absolute path, got %s", result.Path)
	}
}

func TestResolve_DirectoryWithSKILLmd(t *testing.T) {
	// Create a temporary directory with SKILL.md
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Resolve(skillDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsURL {
		t.Error("expected IsURL to be false")
	}

	if !strings.HasSuffix(result.Path, "SKILL.md") {
		t.Errorf("expected path to end with SKILL.md, got %s", result.Path)
	}
}

func TestResolve_BareWord(t *testing.T) {
	// Create .claude/skills/<name>/SKILL.md structure
	claudeDir := filepath.Join(".", ".claude", "skills", "test-skill")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(".claude")
	}()

	skillFile := filepath.Join(claudeDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Resolve("test-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsURL {
		t.Error("expected IsURL to be false")
	}

	if !strings.Contains(result.Path, ".claude") {
		t.Errorf("expected path to contain .claude, got %s", result.Path)
	}

	if !strings.HasSuffix(result.Path, "SKILL.md") {
		t.Errorf("expected path to end with SKILL.md, got %s", result.Path)
	}
}

func TestResolve_BareWord_HomeDirectory(t *testing.T) {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("could not get home directory")
	}

	// Create $HOME/.claude/skills/<name>/SKILL.md structure
	homeClaudeDir := filepath.Join(homeDir, ".claude", "skills", "home-test-skill")
	if err := os.MkdirAll(homeClaudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(filepath.Join(homeDir, ".claude", "skills", "home-test-skill"))
	}()

	skillFile := filepath.Join(homeClaudeDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("test content from home"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Resolve("home-test-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsURL {
		t.Error("expected IsURL to be false")
	}

	if !strings.Contains(result.Path, ".claude") {
		t.Errorf("expected path to contain .claude, got %s", result.Path)
	}

	if !strings.HasSuffix(result.Path, "SKILL.md") {
		t.Errorf("expected path to end with SKILL.md, got %s", result.Path)
	}

	// Verify content matches what we wrote
	content, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "test content from home") {
		t.Errorf("expected content from home directory, got: %s", content)
	}
}

func TestResolve_BareWord_PrioritizesWorkingDirectory(t *testing.T) {
	// Create both ./.claude/skills and $HOME/.claude/skills with same skill name
	// Working directory version should take priority

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("could not get home directory")
	}

	// Create working directory version
	workingClaudeDir := filepath.Join(".", ".claude", "skills", "priority-test")
	if err := os.MkdirAll(workingClaudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(".claude")
	}()

	workingSkillFile := filepath.Join(workingClaudeDir, "SKILL.md")
	if err := os.WriteFile(workingSkillFile, []byte("working directory version"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create home directory version
	homeClaudeDir := filepath.Join(homeDir, ".claude", "skills", "priority-test")
	if err := os.MkdirAll(homeClaudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(filepath.Join(homeDir, ".claude", "skills", "priority-test"))
	}()

	homeSkillFile := filepath.Join(homeClaudeDir, "SKILL.md")
	if err := os.WriteFile(homeSkillFile, []byte("home directory version"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Resolve("priority-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify we got the working directory version
	content, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "working directory version") {
		t.Errorf("expected working directory version, got: %s", content)
	}
}

func TestResolve_BareWordNotFound(t *testing.T) {
	_, err := Resolve("nonexistent-skill")
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestResolve_BareWord_Command(t *testing.T) {
	// Create .claude/commands/<name>.md structure
	claudeDir := filepath.Join(".", ".claude", "commands")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(".claude")
	}()

	commandFile := filepath.Join(claudeDir, "test-command.md")
	if err := os.WriteFile(commandFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Resolve("test-command")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsURL {
		t.Error("expected IsURL to be false")
	}

	if result.Type != ResourceTypeCommand {
		t.Errorf("expected ResourceTypeCommand, got %d", result.Type)
	}

	if !strings.Contains(result.Path, ".claude") {
		t.Errorf("expected path to contain .claude, got %s", result.Path)
	}

	if !strings.HasSuffix(result.Path, ".md") {
		t.Errorf("expected path to end with .md, got %s", result.Path)
	}
}

func TestResolve_BareWord_SkillPrioritizedOverCommand(t *testing.T) {
	// Create both skill and command with same name
	// Skill should take priority

	// Create skill
	skillDir := filepath.Join(".", ".claude", "skills", "priority-test-2")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(".claude")
	}()

	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("skill content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create command
	commandDir := filepath.Join(".", ".claude", "commands")
	if err := os.MkdirAll(commandDir, 0755); err != nil {
		t.Fatal(err)
	}

	commandFile := filepath.Join(commandDir, "priority-test-2.md")
	if err := os.WriteFile(commandFile, []byte("command content"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Resolve("priority-test-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should resolve to skill, not command
	if result.Type != ResourceTypeSkill {
		t.Errorf("expected ResourceTypeSkill, got %d", result.Type)
	}

	if !strings.HasSuffix(result.Path, "SKILL.md") {
		t.Errorf("expected path to end with SKILL.md, got %s", result.Path)
	}
}

func TestResolve_ExactFilePath_ResourceType(t *testing.T) {
	tmpDir := t.TempDir()

	// Test SKILL.md file
	skillFile := filepath.Join(tmpDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("skill"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Resolve(skillFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != ResourceTypeSkill {
		t.Errorf("expected ResourceTypeSkill for SKILL.md, got %d", result.Type)
	}

	// Test command .md file
	commandFile := filepath.Join(tmpDir, "test-command.md")
	if err := os.WriteFile(commandFile, []byte("command"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err = Resolve(commandFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != ResourceTypeCommand {
		t.Errorf("expected ResourceTypeCommand for .md file, got %d", result.Type)
	}
}

func TestResolve_URL_Success(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("---\nname: test\ndescription: test skill\n---\nTest content"))
	}))
	defer server.Close()

	result, err := Resolve(server.URL + "/skill.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		_ = os.Remove(result.Path)
	}()

	if !result.IsURL {
		t.Error("expected IsURL to be true")
	}

	if result.BaseURL == "" {
		t.Error("expected non-empty BaseURL")
	}

	// Verify the file was created and contains the content
	content, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "Test content") {
		t.Errorf("expected content to contain 'Test content', got: %s", content)
	}
}

func TestResolve_URL_TooLarge(t *testing.T) {
	// Create a test HTTP server that returns a file larger than 25kB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		// Write more than 25kB
		largeContent := strings.Repeat("a", 26*1024)
		_, _ = w.Write([]byte(largeContent))
	}))
	defer server.Close()

	_, err := Resolve(server.URL + "/skill.md")
	if err == nil {
		t.Error("expected error for file too large")
	}

	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("expected 'too large' error, got: %v", err)
	}
}

func TestResolve_URL_BinaryContent(t *testing.T) {
	// Create a test HTTP server that returns binary content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		// Write binary content (with null bytes)
		binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE}
		_, _ = w.Write(binaryContent)
	}))
	defer server.Close()

	_, err := Resolve(server.URL + "/binary")
	if err == nil {
		t.Error("expected error for binary content")
	}

	if !strings.Contains(err.Error(), "binary") && !strings.Contains(err.Error(), "text") {
		t.Errorf("expected binary/text error, got: %v", err)
	}
}

func TestResolve_URL_404(t *testing.T) {
	// Create a test HTTP server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := Resolve(server.URL + "/notfound.md")
	if err == nil {
		t.Error("expected error for 404 response")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 error, got: %v", err)
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"https://example.com/path/to/file.md", true},
		{"file:///path/to/file", false},
		{"ftp://example.com", false},
		{"/path/to/file", false},
		{"relative/path", false},
		{"skill-name", false},
	}

	for _, tt := range tests {
		got := isURL(tt.input)
		if got != tt.want {
			t.Errorf("isURL(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsTextContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"text/plain", true},
		{"text/html", true},
		{"text/markdown", true},
		{"text/plain; charset=utf-8", true},
		{"application/json", true},
		{"application/yaml", true},
		{"application/x-yaml", true},
		{"application/octet-stream", false},
		{"image/png", false},
		{"video/mp4", false},
	}

	for _, tt := range tests {
		got := isTextContentType(tt.contentType)
		if got != tt.want {
			t.Errorf("isTextContentType(%q) = %v, want %v", tt.contentType, got, tt.want)
		}
	}
}

func TestIsTextContent(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    bool
	}{
		{
			name:    "plain text",
			content: []byte("This is plain text content"),
			want:    true,
		},
		{
			name:    "text with newlines",
			content: []byte("Line 1\nLine 2\nLine 3"),
			want:    true,
		},
		{
			name:    "binary with null bytes",
			content: []byte{0x00, 0x01, 0x02, 0x03},
			want:    false,
		},
		{
			name:    "mixed text and binary",
			content: []byte("Some text\x00with null"),
			want:    false,
		},
		{
			name:    "markdown content",
			content: []byte("# Heading\n\nSome **bold** text"),
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTextContent(tt.content)
			if got != tt.want {
				t.Errorf("isTextContent() = %v, want %v", got, tt.want)
			}
		})
	}
}
