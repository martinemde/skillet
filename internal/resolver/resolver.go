package resolver

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxURLFileSize = 25 * 1024 // 25kB
	skillFileName  = "SKILL.md"
)

// ResolveResult contains the resolved path and metadata
type ResolveResult struct {
	Path    string // Absolute path to the SKILL.md file
	IsURL   bool   // True if the path was resolved from a URL
	BaseURL string // Base URL for URL-based skills (empty for local files)
}

// Resolve takes a path argument and resolves it to a SKILL.md file
// Resolution order:
// 1. If URL: download and validate
// 2. If exact file path exists: use it
// 3. If directory with SKILL.md exists: use it
// 4. If bare word: look in .claude/skills/<name>/SKILL.md
func Resolve(path string) (*ResolveResult, error) {
	// Check if it's a URL
	if isURL(path) {
		return resolveURL(path)
	}

	// Try exact path
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			// It's a file, use it directly
			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path: %w", err)
			}
			return &ResolveResult{Path: absPath}, nil
		}
		// It's a directory, try appending SKILL.md
		skillPath := filepath.Join(path, skillFileName)
		if _, err := os.Stat(skillPath); err == nil {
			absPath, err := filepath.Abs(skillPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path: %w", err)
			}
			return &ResolveResult{Path: absPath}, nil
		}
	}

	// Check if it's a bare word (no path separators)
	if !strings.Contains(path, "/") && !strings.Contains(path, "\\") {
		// Try .claude/skills/<name>/SKILL.md
		claudeSkillPath := filepath.Join(".claude", "skills", path, skillFileName)
		if _, err := os.Stat(claudeSkillPath); err == nil {
			absPath, err := filepath.Abs(claudeSkillPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path: %w", err)
			}
			return &ResolveResult{Path: absPath}, nil
		}
	}

	return nil, fmt.Errorf("skill not found: %s (tried exact path, directory with SKILL.md, and .claude/skills/<name>/SKILL.md)", path)
}

// isURL checks if a string is a valid HTTP(S) URL
func isURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return scheme == "http" || scheme == "https"
}

// resolveURL downloads a skill from a URL and validates it
func resolveURL(urlStr string) (*ResolveResult, error) {
	// Parse the URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Download the file
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to download URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download URL: HTTP %d", resp.StatusCode)
	}

	// Check Content-Type to ensure it's text
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !isTextContentType(contentType) {
		return nil, fmt.Errorf("URL must point to a text file, got Content-Type: %s", contentType)
	}

	// Read the response body with size limit
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxURLFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read URL content: %w", err)
	}

	// Check size limit
	if len(content) > maxURLFileSize {
		return nil, fmt.Errorf("URL content too large: must be ≤25kB, got %d bytes", len(content))
	}

	// Validate that it looks like text (not binary)
	if !isTextContent(content) {
		return nil, fmt.Errorf("URL content appears to be binary, not text")
	}

	// Create a temporary file to store the downloaded content
	tmpFile, err := os.CreateTemp("", "skillet-url-*.md")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Get the base URL (directory containing the file)
	baseURL := parsedURL.Scheme + "://" + parsedURL.Host + filepath.Dir(parsedURL.Path)

	return &ResolveResult{
		Path:    tmpFile.Name(),
		IsURL:   true,
		BaseURL: baseURL,
	}, nil
}

// isTextContentType checks if the Content-Type header indicates text
func isTextContentType(contentType string) bool {
	// Remove parameters (e.g., "text/plain; charset=utf-8" -> "text/plain")
	ct := strings.ToLower(strings.Split(contentType, ";")[0])
	ct = strings.TrimSpace(ct)

	// Allow text/* types and common text formats
	return strings.HasPrefix(ct, "text/") ||
		ct == "application/json" ||
		ct == "application/x-yaml" ||
		ct == "application/yaml"
}

// isTextContent checks if content appears to be text (not binary)
func isTextContent(content []byte) bool {
	// Check for null bytes (common in binary files)
	for _, b := range content {
		if b == 0 {
			return false
		}
	}

	// Check if content is mostly printable ASCII/UTF-8
	printable := 0
	for _, b := range content {
		if b == '\n' || b == '\r' || b == '\t' || (b >= 32 && b < 127) {
			printable++
		}
	}

	// If at least 95% is printable, consider it text
	return float64(printable)/float64(len(content)) >= 0.95
}
