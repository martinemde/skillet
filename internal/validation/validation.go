// Package validation provides shared validation logic for skills and commands.
package validation

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// BaseDirRegex matches {baseDir} variable references for interpolation
	BaseDirRegex = regexp.MustCompile(`\{baseDir\}`)
	// NameRegex validates resource name format (lowercase letters, numbers, hyphens)
	NameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
)

// ValidateName checks that a resource name is valid.
// Names must be lowercase letters, numbers, and hyphens, not starting/ending with hyphen,
// and cannot contain consecutive hyphens.
func ValidateName(name string, resourceType string) error {
	if name == "" {
		return fmt.Errorf("%s name is required", resourceType)
	}

	if !NameRegex.MatchString(name) {
		return fmt.Errorf("invalid %s name format: must be lowercase letters, numbers, and hyphens, not starting/ending with hyphen", resourceType)
	}

	if len(name) > 64 {
		return fmt.Errorf("%s name too long: max 64 characters, got %d", resourceType, len(name))
	}

	if strings.Contains(name, "--") {
		return fmt.Errorf("%s name cannot contain consecutive hyphens", resourceType)
	}

	return nil
}

// InterpolateBaseDir replaces {baseDir} with the actual base directory path
func InterpolateBaseDir(content, baseDir string) string {
	return BaseDirRegex.ReplaceAllString(content, baseDir)
}
