package stackfinder

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

// StackMetadata contains the metadata extracted from a Stack file
type StackMetadata struct {
	Name     string            // metadata.name from the Stack
	Labels   map[string]string // metadata.labels from the Stack
	FilePath string            // path to the Stack file
}

// Stack represents the minimal structure needed to identify and extract metadata from a Stack file
type Stack struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name   string            `yaml:"name"`
		Labels map[string]string `yaml:"labels"`
	} `yaml:"metadata"`
}

// FindStacks finds all YAML files matching the glob pattern, parses them, and returns
// metadata for those that are of kind: Stack
func FindStacks(globPattern string) ([]StackMetadata, error) {
	// Find all files matching the glob pattern
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, fmt.Errorf("error matching glob pattern %s: %w", globPattern, err)
	}

	var stacks []StackMetadata

	for _, filePath := range matches {
		// Check if it's a file
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue // Skip files with errors
		}
		if fileInfo.IsDir() {
			continue // Skip directories
		}

		// Check if it's a YAML file
		if !isYAMLFile(filePath) {
			continue
		}

		// Try to identify and extract Stack information
		metadata, found, err := extractStackMetadata(filePath)
		if err != nil {
			fmt.Printf("Warning: Error processing %s: %v\n", filePath, err)
			continue
		}
		if !found {
			continue
		}

		stacks = append(stacks, metadata)
	}

	return stacks, nil
}

// FindStacksRecursive finds all Stack files in a directory and its subdirectories
func FindStacksRecursive(root string) ([]StackMetadata, error) {
	var stacks []StackMetadata

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if it's a YAML file
		if !isYAMLFile(path) {
			return nil
		}

		// Try to identify and extract Stack information
		metadata, found, err := extractStackMetadata(path)
		if err != nil {
			fmt.Printf("Warning: Error processing %s: %v\n", path, err)
			return nil
		}
		if !found {
			return nil
		}

		stacks = append(stacks, metadata)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", root, err)
	}

	return stacks, nil
}

// extractStackMetadata attempts to extract Stack metadata from a YAML file
// It first tries to unmarshal the YAML, and if that fails due to unresolved anchors,
// it falls back to regex-based detection
func extractStackMetadata(filePath string) (StackMetadata, bool, error) {
	// Read file content
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return StackMetadata{}, false, fmt.Errorf("failed to read file: %w", err)
	}

	// First attempt: Try standard YAML parsing
	var stack Stack
	err = yaml.Unmarshal(fileData, &stack)

	// If parsing succeeded and it's a Stack, extract metadata
	if err == nil && stack.Kind == "Stack" {
		return StackMetadata{
			Name:     stack.Metadata.Name,
			Labels:   stack.Metadata.Labels,
			FilePath: filePath,
		}, true, nil
	}

	// Second attempt: Use regex-based detection for files that might contain unresolved anchors
	return extractStackMetadataWithRegex(filePath, fileData)
}

// extractStackMetadataWithRegex uses regex to extract Stack information from a YAML file with unresolved anchors
func extractStackMetadataWithRegex(filePath string, fileData []byte) (StackMetadata, bool, error) {
	content := string(fileData)

	// Check if it's a Stack kind
	kindRe := regexp.MustCompile(`(?m)^kind:\s*Stack\s*$`)
	if !kindRe.MatchString(content) {
		return StackMetadata{}, false, nil
	}

	// Extract name
	nameRe := regexp.MustCompile(`(?m)^  name:\s*(.+?)\s*$`)
	nameMatches := nameRe.FindStringSubmatch(content)
	if len(nameMatches) < 2 {
		return StackMetadata{}, false, fmt.Errorf("stack name not found")
	}
	name := nameMatches[1]

	// Extract labels directly with a simpler approach
	labels := make(map[string]string)

	// Use a simpler approach to extract key-value pairs from the labels section
	labelsStartRe := regexp.MustCompile(`(?m)^  labels:`)
	labelsStart := labelsStartRe.FindStringIndex(content)

	if labelsStart != nil {
		// Find the end of metadata section
		metadataEndRe := regexp.MustCompile(`(?m)^spec:`)
		metadataEnd := metadataEndRe.FindStringIndex(content)

		// Extract the labels section
		var labelsSection string
		if metadataEnd != nil {
			labelsSection = content[labelsStart[1]:metadataEnd[0]]
		} else {
			labelsSection = content[labelsStart[1]:]
		}

		// Extract direct key-value pairs (exclude lines with anchors or merge operators)
		labelLineRe := regexp.MustCompile(`(?m)^    ([a-zA-Z0-9_-]+):\s*([^*{}\[\]<]+)\s*$`)
		labelMatches := labelLineRe.FindAllStringSubmatch(labelsSection, -1)

		for _, match := range labelMatches {
			if len(match) >= 3 {
				key := strings.TrimSpace(match[1])
				value := strings.TrimSpace(match[2])
				if key != "" && value != "" {
					labels[key] = value
				}
			}
		}
	}

	return StackMetadata{
		Name:     name,
		Labels:   labels,
		FilePath: filePath,
	}, true, nil
}

// isYAMLFile checks if a file has a YAML extension
func isYAMLFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
}
