package utils

import (
	"testing"

	stackfinder "github.com/mcalhoun/skunk/internal/stack-finder"
	"github.com/stretchr/testify/assert"
)

func TestFilterStacks(t *testing.T) {
	// Create test stacks
	stacks := []stackfinder.StackMetadata{
		{
			Name: "dev-stack-1",
			Labels: map[string]string{
				"env":    "dev",
				"region": "us-east-1",
			},
		},
		{
			Name: "prod-stack-1",
			Labels: map[string]string{
				"env":    "prod",
				"region": "us-west-1",
			},
		},
		{
			Name: "prod-stack-2",
			Labels: map[string]string{
				"env":    "prod",
				"region": "us-east-1",
			},
		},
		{
			Name: "test-stack",
			Labels: map[string]string{
				"env":    "test",
				"region": "eu-west-1",
			},
		},
	}

	// Test cases
	tests := []struct {
		name     string
		filters  []string
		expected []string // Expected stack names after filtering
	}{
		{
			name:     "No filters",
			filters:  []string{},
			expected: []string{"dev-stack-1", "prod-stack-1", "prod-stack-2", "test-stack"},
		},
		{
			name:     "Filter by label key=value",
			filters:  []string{"env=prod"},
			expected: []string{"prod-stack-1", "prod-stack-2"},
		},
		{
			name:     "Filter by label key!=value",
			filters:  []string{"env!=prod"},
			expected: []string{"dev-stack-1", "test-stack"},
		},
		{
			name:     "Multiple label filters (AND logic)",
			filters:  []string{"env=prod", "region=us-east-1"},
			expected: []string{"prod-stack-2"},
		},
		{
			name:     "Name wildcard filter with *",
			filters:  []string{"name=prod-*"},
			expected: []string{"prod-stack-1", "prod-stack-2"},
		},
		{
			name:     "Name wildcard filter with ? (single character)",
			filters:  []string{"name=*-stack-?"},
			expected: []string{"dev-stack-1", "prod-stack-1", "prod-stack-2"},
		},
		{
			name:     "Name negative wildcard filter",
			filters:  []string{"name!=*-stack-?"},
			expected: []string{"test-stack"},
		},
		{
			name:     "Name regex filter",
			filters:  []string{"name~=^prod-.*$"},
			expected: []string{"prod-stack-1", "prod-stack-2"},
		},
		{
			name:     "Name negative regex filter",
			filters:  []string{"name!~=^prod-.*$"},
			expected: []string{"dev-stack-1", "test-stack"},
		},
		{
			name:     "Name wildcard and label filter combination",
			filters:  []string{"name=*-stack-*", "region=us-east-1"},
			expected: []string{"dev-stack-1", "prod-stack-2"},
		},
		{
			name:     "Complex regex pattern",
			filters:  []string{"name~=^(dev|prod)-stack-\\d+$"},
			expected: []string{"dev-stack-1", "prod-stack-1", "prod-stack-2"},
		},
		// New test cases for naked strings as wildcards
		{
			name:     "Naked string as wildcard",
			filters:  []string{"prod-*"},
			expected: []string{"prod-stack-1", "prod-stack-2"},
		},
		{
			name:     "Naked string with question mark",
			filters:  []string{"*-stack-?"},
			expected: []string{"dev-stack-1", "prod-stack-1", "prod-stack-2"},
		},
		{
			name:     "Multiple naked string wildcards (AND logic)",
			filters:  []string{"prod-*", "*-1"},
			expected: []string{"prod-stack-1"},
		},
		// New test cases for regex patterns enclosed in slashes
		{
			name:     "Regex pattern in slashes",
			filters:  []string{"/^prod-.*$/"},
			expected: []string{"prod-stack-1", "prod-stack-2"},
		},
		{
			name:     "Complex regex in slashes",
			filters:  []string{"/^(dev|prod)-stack-\\d+$/"},
			expected: []string{"dev-stack-1", "prod-stack-1", "prod-stack-2"},
		},
		{
			name:     "Combined regex in slashes and label filter",
			filters:  []string{"/.*-stack-[12]$/", "region=us-east-1"},
			expected: []string{"dev-stack-1", "prod-stack-2"},
		},
		// Mixed filter formats
		{
			name:     "Mixed filter formats (naked string + label)",
			filters:  []string{"*-stack-*", "env=prod"},
			expected: []string{"prod-stack-1", "prod-stack-2"},
		},
		{
			name:     "Mixed filter formats (regex in slashes + naked string)",
			filters:  []string{"/^prod/", "*-1"},
			expected: []string{"prod-stack-1"},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterStacks(stacks, tt.filters)
			var resultNames []string
			for _, stack := range result {
				resultNames = append(resultNames, stack.Name)
			}
			assert.ElementsMatch(t, tt.expected, resultNames)
		})
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		pattern  string
		expected bool
	}{
		{
			name:     "Exact match",
			str:      "test-stack",
			pattern:  "test-stack",
			expected: true,
		},
		{
			name:     "Star wildcard at end",
			str:      "test-stack-1",
			pattern:  "test-*",
			expected: true,
		},
		{
			name:     "Star wildcard at beginning",
			str:      "test-stack",
			pattern:  "*-stack",
			expected: true,
		},
		{
			name:     "Star wildcard in middle",
			str:      "test-stack-1",
			pattern:  "test-*-1",
			expected: true,
		},
		{
			name:     "Multiple star wildcards",
			str:      "test-stack-prod-1",
			pattern:  "*-*-*-*",
			expected: true,
		},
		{
			name:     "Question mark wildcard",
			str:      "test-stack-1",
			pattern:  "test-stack-?",
			expected: true,
		},
		{
			name:     "Multiple question marks",
			str:      "abc",
			pattern:  "???",
			expected: true,
		},
		{
			name:     "Mixed wildcards",
			str:      "test-stack-prod",
			pattern:  "test-*-p???",
			expected: true,
		},
		{
			name:     "No match - different lengths",
			str:      "test-stack-1",
			pattern:  "test-stack",
			expected: false,
		},
		{
			name:     "No match - same length but different characters",
			str:      "test-stack-1",
			pattern:  "test-stack-2",
			expected: false,
		},
		{
			name:     "No match with wildcard",
			str:      "test-stack-1",
			pattern:  "prod-*",
			expected: false,
		},
		{
			name:     "Escaped special characters",
			str:      "test.stack+1",
			pattern:  "test.stack+?",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchWildcard(tt.str, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}
