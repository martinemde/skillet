// Package frontmatter provides utilities for parsing YAML frontmatter from markdown files.
package frontmatter

import (
	"bufio"
	"fmt"
	"strings"
)

// ParseResult contains the extracted frontmatter and content from a markdown file.
type ParseResult struct {
	// FrontmatterYAML is the raw YAML frontmatter (without delimiters)
	FrontmatterYAML string
	// Content is the markdown content after the frontmatter
	Content string
	// HasFrontmatter indicates if frontmatter was present in the file
	HasFrontmatter bool
}

// Parse extracts YAML frontmatter and content from a markdown file.
// Frontmatter is delimited by "---" at the start and end.
// If requireFrontmatter is true, returns an error if frontmatter is missing.
func Parse(data string, requireFrontmatter bool) (*ParseResult, error) {
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

	hasFrontmatter := frontmatterCount >= 2

	if requireFrontmatter && !hasFrontmatter {
		return nil, fmt.Errorf("invalid frontmatter: expected two '---' delimiters, found %d", frontmatterCount)
	}

	return &ParseResult{
		FrontmatterYAML: strings.Join(frontmatterLines, "\n"),
		Content:         strings.TrimSpace(strings.Join(contentLines, "\n")),
		HasFrontmatter:  hasFrontmatter,
	}, nil
}
