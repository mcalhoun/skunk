package utils

import (
	"regexp"
	"strings"

	"github.com/mcalhoun/skunk/internal/logger"
	stackfinder "github.com/mcalhoun/skunk/internal/stack-finder"
)

// FilterStacks applies filters to the list of stacks
// Filters can be in the following formats:
// - key=value: Label filter where key equals value
// - key!=value: Label filter where key does not equal value
// - name=pattern: Stack name filter where name matches the pattern (supports wildcards using * and ?)
// - name!=pattern: Stack name filter where name does not match the pattern
// - name~=pattern: Stack name filter where name matches the regex pattern
// - name!~=pattern: Stack name filter where name does not match the regex pattern
// - pattern: Simple wildcard pattern matching against stack name (no key=value)
// - /pattern/: Regex pattern matching against stack name (enclosed in slashes)
func FilterStacks(stacks []stackfinder.StackMetadata, filters []string) []stackfinder.StackMetadata {
	var filteredStacks []stackfinder.StackMetadata

	for _, stack := range stacks {
		if shouldIncludeStack(stack, filters) {
			filteredStacks = append(filteredStacks, stack)
		}
	}

	return filteredStacks
}

// shouldIncludeStack checks if a stack should be included based on the filters
func shouldIncludeStack(stack stackfinder.StackMetadata, filters []string) bool {
	// Default to including each stack
	include := true

	// Apply each filter
	for _, filter := range filters {
		if !applyFilter(stack, filter) {
			include = false
			break
		}
	}

	return include
}

// applyFilter applies a single filter to a stack and returns true if the stack matches the filter
func applyFilter(stack stackfinder.StackMetadata, filter string) bool {
	// Check if it's a regex pattern (enclosed in slashes)
	if isRegexPatternInSlashes(filter) {
		return applyRegexPattern(stack.Name, filter)
	}

	// Check if it's a name filter with prefixes
	if strings.HasPrefix(filter, "name=") {
		pattern := strings.TrimPrefix(filter, "name=")
		return MatchWildcard(stack.Name, pattern)
	}

	if strings.HasPrefix(filter, "name!=") {
		pattern := strings.TrimPrefix(filter, "name!=")
		return !MatchWildcard(stack.Name, pattern)
	}

	if strings.HasPrefix(filter, "name~=") {
		pattern := strings.TrimPrefix(filter, "name~=")
		return applyNameRegexPattern(stack.Name, pattern)
	}

	if strings.HasPrefix(filter, "name!~=") {
		pattern := strings.TrimPrefix(filter, "name!~=")
		return !applyNameRegexPattern(stack.Name, pattern)
	}

	// Check if it's a label filter
	if strings.Contains(filter, "!=") && !strings.HasPrefix(filter, "name") {
		return applyLabelNotEqualsFilter(stack, filter)
	}

	if strings.Contains(filter, "=") && !strings.HasPrefix(filter, "name") {
		return applyLabelEqualsFilter(stack, filter)
	}

	// If we've reached here, it's a naked string - treat as wildcard pattern for stack name
	return MatchWildcard(stack.Name, filter)
}

// isRegexPatternInSlashes checks if the filter is a regex pattern enclosed in slashes
func isRegexPatternInSlashes(filter string) bool {
	return len(filter) > 2 && filter[0] == '/' && filter[len(filter)-1] == '/'
}

// applyRegexPattern applies a regex pattern enclosed in slashes
func applyRegexPattern(name string, filter string) bool {
	// Extract the pattern between slashes
	pattern := filter[1 : len(filter)-1]
	matched, err := regexp.MatchString(pattern, name)
	if err != nil {
		logger.Log.Warnf("Invalid regex pattern: %s, error: %v", pattern, err)
		return true // Skip this filter if it's invalid
	}
	return matched
}

// applyNameRegexPattern applies a regex pattern for name
func applyNameRegexPattern(name string, pattern string) bool {
	matched, err := regexp.MatchString(pattern, name)
	if err != nil {
		logger.Log.Warnf("Invalid regex pattern: %s, error: %v", pattern, err)
		return true // Skip this filter if it's invalid
	}
	return matched
}

// applyLabelEqualsFilter applies a label equals filter
func applyLabelEqualsFilter(stack stackfinder.StackMetadata, filter string) bool {
	parts := strings.SplitN(filter, "=", 2)
	if len(parts) != 2 {
		return true // Invalid filter format, ignore
	}

	key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	// If the label doesn't exist or doesn't match the value, skip this stack
	labelValue, exists := stack.Labels[key]
	return exists && labelValue == value
}

// applyLabelNotEqualsFilter applies a label not equals filter
func applyLabelNotEqualsFilter(stack stackfinder.StackMetadata, filter string) bool {
	parts := strings.SplitN(filter, "!=", 2)
	if len(parts) != 2 {
		return true // Invalid filter format, ignore
	}

	key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	// If the label exists and matches the value we're excluding, skip this stack
	labelValue, exists := stack.Labels[key]
	return !exists || labelValue != value
}

// MatchWildcard checks if the given string matches the wildcard pattern
// Supports * (any number of characters) and ? (exactly one character)
func MatchWildcard(s, pattern string) bool {
	// Convert wildcard pattern to regex pattern
	regexPattern := "^"
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			regexPattern += ".*"
		case '?':
			regexPattern += "."
		case '.', '\\', '+', '(', ')', '[', ']', '{', '}', '^', '$', '|':
			// Escape regex special characters
			regexPattern += "\\" + string(pattern[i])
		default:
			regexPattern += string(pattern[i])
		}
	}
	regexPattern += "$"

	// Match using regex
	matched, err := regexp.MatchString(regexPattern, s)
	if err != nil {
		logger.Log.Warnf("Error matching pattern %s: %v", pattern, err)
		return false
	}
	return matched
}
