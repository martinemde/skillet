package resolver

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/martinemde/skillet/internal/command"
	"github.com/martinemde/skillet/internal/commandpath"
	"github.com/martinemde/skillet/internal/discovery"
	"github.com/martinemde/skillet/internal/skillpath"
)

const (
	maxURLFileSize = 25 * 1024 // 25kB
	skillFileName  = "SKILL.md"
)

// ResourceType indicates what type of resource was resolved
type ResourceType int

const (
	// ResourceTypeSkill indicates the resolved resource is a skill (SKILL.md)
	ResourceTypeSkill ResourceType = iota
	// ResourceTypeCommand indicates the resolved resource is a command (.md file)
	ResourceTypeCommand
)

// ResolveResult contains the resolved path and metadata
type ResolveResult struct {
	Path    string       // Absolute path to the resolved file
	IsURL   bool         // True if the path was resolved from a URL
	BaseURL string       // Base URL for URL-based resources (empty for local files)
	Type    ResourceType // Type of resource (skill or command)
}

// matchSpecificity indicates how well a query matched a resource
type matchSpecificity int

const (
	// exactNamespaceMatch: Query "frontend:test" matches skill/command "frontend:test"
	exactNamespaceMatch matchSpecificity = iota
	// unnamespacedExact: Query "test" matches unnamespaced "test"
	unnamespacedExact
	// namespacedFallback: Query "test" matches namespaced "frontend:test"
	namespacedFallback
)

// match represents a candidate match during resolution
type match struct {
	path         string
	resourceType ResourceType
	priority     int              // source priority (lower = higher priority)
	specificity  matchSpecificity // how well the query matched
	namespace    string           // for error messages
	name         string           // for error messages
}

// Resolver handles namespace-aware resolution of skills and commands
type Resolver struct {
	skillPath *skillpath.Path
	cmdPath   *commandpath.Path
}

// New creates a new Resolver with default skill and command paths
func New() (*Resolver, error) {
	sp, err := skillpath.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize skill path: %w", err)
	}

	cp, err := commandpath.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize command path: %w", err)
	}

	return &Resolver{
		skillPath: sp,
		cmdPath:   cp,
	}, nil
}

// NewWithPaths creates a Resolver with custom paths (useful for testing)
func NewWithPaths(sp *skillpath.Path, cp *commandpath.Path) *Resolver {
	return &Resolver{
		skillPath: sp,
		cmdPath:   cp,
	}
}

// parseNamespaceQuery splits a query into namespace and name components
// "frontend:test" -> ("frontend", "test")
// "test" -> ("", "test")
func parseNamespaceQuery(query string) (namespace, name string) {
	if idx := strings.Index(query, ":"); idx != -1 {
		return query[:idx], query[idx+1:]
	}
	return "", query
}

// Resolve takes a path argument and resolves it to a SKILL.md or command file.
// Resolution order:
// 1. If URL: download and validate
// 2. If exact file path exists: use it
// 3. If directory with SKILL.md exists: use it
// 4. Namespace-aware discovery:
//   - Parse query for namespace (e.g., "frontend:test")
//   - Collect matching skills and commands
//   - Score by specificity, priority, and resource type
//   - Return best match or collision error
func (r *Resolver) Resolve(input string) (*ResolveResult, error) {
	// Check if it's a URL
	if isURL(input) {
		return resolveURL(input)
	}

	// Try exact path
	if info, err := os.Stat(input); err == nil {
		if !info.IsDir() {
			// It's a file, use it directly
			absPath, err := filepath.Abs(input)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path: %w", err)
			}
			// Determine type based on filename
			resourceType := ResourceTypeCommand
			if filepath.Base(absPath) == skillFileName {
				resourceType = ResourceTypeSkill
			}
			return &ResolveResult{Path: absPath, Type: resourceType}, nil
		}
		// It's a directory, try appending SKILL.md
		skillPath := filepath.Join(input, skillFileName)
		if _, err := os.Stat(skillPath); err == nil {
			absPath, err := filepath.Abs(skillPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path: %w", err)
			}
			return &ResolveResult{Path: absPath, Type: ResourceTypeSkill}, nil
		}
	}

	// Check if it's a bare word (no path separators)
	if !strings.Contains(input, "/") && !strings.Contains(input, "\\") {
		return r.resolveByName(input)
	}

	return nil, fmt.Errorf("skill or command not found: %s", input)
}

// resolveByName resolves a bare word query using namespace-aware matching
func (r *Resolver) resolveByName(query string) (*ResolveResult, error) {
	queryNS, queryName := parseNamespaceQuery(query)

	var matches []match

	// Discover all skills
	skillDisc := discovery.New(r.skillPath)
	skills, err := skillDisc.Discover()
	if err != nil {
		return nil, fmt.Errorf("failed to discover skills: %w", err)
	}

	// Collect skill matches
	for _, skill := range skills {
		if skill.Overshadowed {
			continue // Skip overshadowed skills
		}

		// Case-insensitive name matching
		if !strings.EqualFold(skill.Name, queryName) {
			continue
		}

		var specificity matchSpecificity
		if queryNS != "" {
			// Query has explicit namespace
			if strings.EqualFold(skill.Namespace, queryNS) {
				specificity = exactNamespaceMatch
			} else {
				continue // Namespace doesn't match, skip
			}
		} else {
			// Query has no namespace
			if skill.Namespace == "" {
				specificity = unnamespacedExact
			} else {
				specificity = namespacedFallback
			}
		}

		matches = append(matches, match{
			path:         skill.Path,
			resourceType: ResourceTypeSkill,
			priority:     skill.Source.Priority,
			specificity:  specificity,
			namespace:    skill.Namespace,
			name:         skill.Name,
		})
	}

	// Discover all commands
	cmdDisc := command.NewDiscoverer(r.cmdPath)
	commands, err := cmdDisc.Discover()
	if err != nil {
		return nil, fmt.Errorf("failed to discover commands: %w", err)
	}

	// Collect command matches
	for _, cmd := range commands {
		if cmd.Overshadowed {
			continue // Skip overshadowed commands
		}

		// Case-insensitive name matching
		if !strings.EqualFold(cmd.Name, queryName) {
			continue
		}

		var specificity matchSpecificity
		if queryNS != "" {
			// Query has explicit namespace
			if strings.EqualFold(cmd.Namespace, queryNS) {
				specificity = exactNamespaceMatch
			} else {
				continue // Namespace doesn't match, skip
			}
		} else {
			// Query has no namespace
			if cmd.Namespace == "" {
				specificity = unnamespacedExact
			} else {
				specificity = namespacedFallback
			}
		}

		matches = append(matches, match{
			path:         cmd.Path,
			resourceType: ResourceTypeCommand,
			priority:     cmd.Source.Priority,
			specificity:  specificity,
			namespace:    cmd.Namespace,
			name:         cmd.Name,
		})
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("skill or command not found: %s (tried exact path, directory with SKILL.md, .claude/skills/<name>/SKILL.md, $HOME/.claude/skills/<name>/SKILL.md, .claude/commands/<name>.md, and $HOME/.claude/commands/<name>.md)", query)
	}

	// Sort matches by: specificity → priority → resource type (skills before commands)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].specificity != matches[j].specificity {
			return matches[i].specificity < matches[j].specificity
		}
		if matches[i].priority != matches[j].priority {
			return matches[i].priority < matches[j].priority
		}
		// Skills before commands
		return matches[i].resourceType < matches[j].resourceType
	})

	// Check for ambiguous matches (collision)
	// Two matches are ambiguous if they have the same specificity and priority
	if len(matches) > 1 {
		best := matches[0]
		// Check if there's another match with same specificity and priority but different namespace
		for _, m := range matches[1:] {
			if m.specificity == best.specificity && m.priority == best.priority {
				// Only collision if both are namespacedFallback (ambiguous fallback)
				if best.specificity == namespacedFallback && m.specificity == namespacedFallback {
					return nil, fmt.Errorf("ambiguous match for %q: found both %s:%s and %s:%s at same priority. Use explicit namespace (e.g., %s:%s or %s:%s)",
						query, best.namespace, best.name, m.namespace, m.name, best.namespace, queryName, m.namespace, queryName)
				}
			} else {
				break // Rest of matches have lower priority
			}
		}
	}

	// Return the best match
	return &ResolveResult{
		Path: matches[0].path,
		Type: matches[0].resourceType,
	}, nil
}

// Resolve is a convenience function that creates a default Resolver and resolves the path.
// For multiple resolutions or testing, use New() and call Resolve() on the returned Resolver.
func Resolve(path string) (*ResolveResult, error) {
	r, err := New()
	if err != nil {
		return nil, err
	}
	return r.Resolve(path)
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
	defer func() {
		_ = resp.Body.Close()
	}()

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
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Get the base URL (directory containing the file)
	baseURL := parsedURL.Scheme + "://" + parsedURL.Host + filepath.Dir(parsedURL.Path)

	// Determine type based on filename in URL (default to skill for URL-based)
	resourceType := ResourceTypeSkill
	urlPath := parsedURL.Path
	if !strings.HasSuffix(strings.ToUpper(urlPath), "/SKILL.MD") && strings.HasSuffix(strings.ToLower(urlPath), ".md") {
		// It's an .md file but not SKILL.md, treat as command
		resourceType = ResourceTypeCommand
	}

	return &ResolveResult{
		Path:    tmpFile.Name(),
		IsURL:   true,
		BaseURL: baseURL,
		Type:    resourceType,
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
