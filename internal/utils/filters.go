package utils

import (
	"strings"

	"github.com/mcalhoun/skunk/internal/logger"
	stackfinder "github.com/mcalhoun/skunk/internal/stack-finder"
)

// FilterStacks applies label filters to the list of stacks
func FilterStacks(stacks []stackfinder.StackMetadata, filters []string) []stackfinder.StackMetadata {
	var filteredStacks []stackfinder.StackMetadata

	for _, stack := range stacks {
		// Default to including each stack
		include := true

		// Apply each filter
		for _, filter := range filters {
			// Check if it's an exclusion filter (contains !=)
			if strings.Contains(filter, "!=") {
				parts := strings.SplitN(filter, "!=", 2)
				if len(parts) == 2 {
					key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
					// If the label exists and matches the value we're excluding, skip this stack
					if labelValue, exists := stack.Labels[key]; exists && labelValue == value {
						include = false
						break
					}
				}
			} else if strings.Contains(filter, "=") {
				parts := strings.SplitN(filter, "=", 2)
				if len(parts) == 2 {
					key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
					// If the label doesn't exist or doesn't match the value, skip this stack
					if labelValue, exists := stack.Labels[key]; !exists || labelValue != value {
						include = false
						break
					}
				}
			} else {
				// Malformed filter, log a warning but continue
				logger.Log.Warnf("Ignoring malformed filter: %s (expected format: key=value or key!=value)", filter)
			}
		}

		if include {
			filteredStacks = append(filteredStacks, stack)
		}
	}

	return filteredStacks
}
