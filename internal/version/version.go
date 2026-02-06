// Package version provides semantic version comparison utilities for SDK version metadata handling.
//
// These utilities are used to compare the current SDK version against
// server-provided version requirements (min, recommended, latest).
package version

import (
	"regexp"
	"strconv"
	"strings"
)

// Version represents a parsed semantic version.
type Version struct {
	Major int
	Minor int
	Patch int
}

// semverRegex matches semantic version strings (allows pre-release suffix but ignores it).
var semverRegex = regexp.MustCompile(`^[vV]?(\d+)\.(\d+)\.(\d+)`)

// maxVersionComponent is the maximum allowed value for version components (defensive limit).
const maxVersionComponent = 999999999

// Parse parses a semantic version string into numeric components.
// Returns nil if the version is not a valid semver.
func Parse(version string) *Version {
	// Trim whitespace
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return nil
	}

	match := semverRegex.FindStringSubmatch(trimmed)
	if match == nil {
		return nil
	}

	major, err := strconv.Atoi(match[1])
	if err != nil || major < 0 || major > maxVersionComponent {
		return nil
	}

	minor, err := strconv.Atoi(match[2])
	if err != nil || minor < 0 || minor > maxVersionComponent {
		return nil
	}

	patch, err := strconv.Atoi(match[3])
	if err != nil || patch < 0 || patch > maxVersionComponent {
		return nil
	}

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

// Compare compares two semantic versions.
// Returns:
//   - negative number if a < b
//   - 0 if a == b
//   - positive number if a > b
//
// Returns 0 if either version is invalid.
func Compare(a, b string) int {
	parsedA := Parse(a)
	parsedB := Parse(b)

	if parsedA == nil || parsedB == nil {
		return 0
	}

	// Compare major
	if parsedA.Major != parsedB.Major {
		return parsedA.Major - parsedB.Major
	}

	// Compare minor
	if parsedA.Minor != parsedB.Minor {
		return parsedA.Minor - parsedB.Minor
	}

	// Compare patch
	return parsedA.Patch - parsedB.Patch
}

// IsLessThan checks if version a is less than version b.
func IsLessThan(a, b string) bool {
	return Compare(a, b) < 0
}

// IsAtLeast checks if version a is greater than or equal to version b.
func IsAtLeast(a, b string) bool {
	return Compare(a, b) >= 0
}
