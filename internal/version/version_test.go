package version

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected *Version
	}{
		{
			name:     "valid semver",
			version:  "1.2.3",
			expected: &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "valid semver with v prefix",
			version:  "v1.2.3",
			expected: &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "valid semver with V prefix (uppercase)",
			version:  "V1.2.3",
			expected: &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "valid semver with prerelease",
			version:  "1.2.3-beta.1",
			expected: &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "valid semver with build metadata",
			version:  "1.2.3+build.456",
			expected: &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "valid semver with leading whitespace",
			version:  "  1.2.3",
			expected: &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "valid semver with trailing whitespace",
			version:  "1.2.3  ",
			expected: &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "valid semver with surrounding whitespace",
			version:  "  1.2.3  ",
			expected: &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "valid semver with v prefix and whitespace",
			version:  "  v1.2.3  ",
			expected: &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "empty string",
			version:  "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			version:  "   ",
			expected: nil,
		},
		{
			name:     "invalid version",
			version:  "not-a-version",
			expected: nil,
		},
		{
			name:     "partial version",
			version:  "1.2",
			expected: nil,
		},
		{
			name:     "version exceeding max component",
			version:  "1000000000.0.0",
			expected: nil,
		},
		{
			name:     "version at max component boundary",
			version:  "999999999.999999999.999999999",
			expected: &Version{Major: 999999999, Minor: 999999999, Patch: 999999999},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.version)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
			} else {
				if result == nil {
					t.Errorf("expected %+v, got nil", tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("expected %+v, got %+v", tt.expected, result)
				}
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int // -1, 0, or 1
	}{
		{
			name:     "equal versions",
			a:        "1.2.3",
			b:        "1.2.3",
			expected: 0,
		},
		{
			name:     "a less than b - major",
			a:        "1.0.0",
			b:        "2.0.0",
			expected: -1,
		},
		{
			name:     "a less than b - minor",
			a:        "1.1.0",
			b:        "1.2.0",
			expected: -1,
		},
		{
			name:     "a less than b - patch",
			a:        "1.2.2",
			b:        "1.2.3",
			expected: -1,
		},
		{
			name:     "a greater than b - major",
			a:        "2.0.0",
			b:        "1.0.0",
			expected: 1,
		},
		{
			name:     "a greater than b - minor",
			a:        "1.3.0",
			b:        "1.2.0",
			expected: 1,
		},
		{
			name:     "a greater than b - patch",
			a:        "1.2.4",
			b:        "1.2.3",
			expected: 1,
		},
		{
			name:     "with v prefix",
			a:        "v1.2.3",
			b:        "1.2.3",
			expected: 0,
		},
		{
			name:     "invalid a returns 0",
			a:        "invalid",
			b:        "1.2.3",
			expected: 0,
		},
		{
			name:     "invalid b returns 0",
			a:        "1.2.3",
			b:        "invalid",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Compare(tt.a, tt.b)
			// Normalize to -1, 0, or 1 for comparison
			var normalized int
			if result < 0 {
				normalized = -1
			} else if result > 0 {
				normalized = 1
			}
			if normalized != tt.expected {
				t.Errorf("Compare(%q, %q) = %d, expected %d", tt.a, tt.b, normalized, tt.expected)
			}
		})
	}
}

func TestIsLessThan(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "a less than b",
			a:        "1.0.0",
			b:        "2.0.0",
			expected: true,
		},
		{
			name:     "a equal to b",
			a:        "1.0.0",
			b:        "1.0.0",
			expected: false,
		},
		{
			name:     "a greater than b",
			a:        "2.0.0",
			b:        "1.0.0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLessThan(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("IsLessThan(%q, %q) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestIsAtLeast(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "a less than b",
			a:        "1.0.0",
			b:        "2.0.0",
			expected: false,
		},
		{
			name:     "a equal to b",
			a:        "1.0.0",
			b:        "1.0.0",
			expected: true,
		},
		{
			name:     "a greater than b",
			a:        "2.0.0",
			b:        "1.0.0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAtLeast(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("IsAtLeast(%q, %q) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
