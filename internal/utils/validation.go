package utils

import (
	stackfinder "github.com/mcalhoun/skunk/internal/stack-finder"
)

// FindDuplicateStacks checks for stacks with the same name and returns a map of duplicates
func FindDuplicateStacks(stacks []stackfinder.StackMetadata) map[string][]string {
	// Map to track occurrences of stack names
	nameToPath := make(map[string][]string)

	// Collect all paths for each stack name
	for _, stack := range stacks {
		nameToPath[stack.Name] = append(nameToPath[stack.Name], stack.FilePath)
	}

	// Filter to include only duplicate names (where there's more than one path)
	duplicates := make(map[string][]string)
	for name, paths := range nameToPath {
		if len(paths) > 1 {
			duplicates[name] = paths
		}
	}

	return duplicates
}
