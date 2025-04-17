package yamlparser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

// FindSubdirectories recursively finds all subdirectories of the given directory
func FindSubdirectories(root string) ([]string, error) {
	var dirs []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", root, err)
	}

	return dirs, nil
}

// CustomDecoder is a wrapper around go-yaml's decoder to handle multiple merge keys
type CustomDecoder struct {
	yamlText string
	options  []yaml.DecodeOption
}

// NewCustomDecoder creates a new CustomDecoder
func NewCustomDecoder(yamlText string, options ...yaml.DecodeOption) *CustomDecoder {
	return &CustomDecoder{
		yamlText: yamlText,
		options:  options,
	}
}

// Decode handles YAML text with multiple merge keys by preprocessing it
func (cd *CustomDecoder) Decode() (map[string]interface{}, error) {
	// Preprocess YAML to handle multiple merge keys
	processedYAML := cd.preprocessMultipleMergeKeys(cd.yamlText)

	// Use go-yaml to decode the processed YAML
	decoder := yaml.NewDecoder(strings.NewReader(processedYAML), cd.options...)

	var result map[string]interface{}
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	return result, nil
}

// preprocessMultipleMergeKeys combines multiple "<<: *anchor" entries into a single entry with array
func (cd *CustomDecoder) preprocessMultipleMergeKeys(yamlText string) string {
	// Split the YAML text into lines
	lines := strings.Split(yamlText, "\n")
	resultLines := make([]string, 0, len(lines))

	// Maps to track merge keys at each indentation level and path
	type pathKey struct {
		indent int
		path   string // Track parent path to differentiate same-indent keys in different sections
	}
	pathToMergeKeys := make(map[pathKey][]string)

	// First pass: collect merge keys by indentation and parent path
	currentPath := []string{}
	previousIndent := 0

	for _, line := range lines {
		if line == "" || strings.TrimSpace(line) == "" {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)

		// Handle path tracking for nested structures
		if indent < previousIndent {
			// Going back up the tree, pop from the path stack
			levelsToRemove := (previousIndent - indent) / 2 // Assuming 2-space indentation
			if len(currentPath) >= levelsToRemove {
				currentPath = currentPath[:len(currentPath)-levelsToRemove]
			} else {
				currentPath = []string{} // Reset if we can't properly track
			}
		} else if indent > previousIndent && !strings.HasPrefix(trimmed, "<<:") {
			// Going deeper in the tree, push to the path stack
			// Only add to path if it's a new section (not a merge key)
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) > 0 {
				currentPath = append(currentPath, parts[0])
			}
		}

		previousIndent = indent

		// Collect merge keys
		if strings.HasPrefix(trimmed, "<<:") {
			pathStr := strings.Join(currentPath, ".")
			key := pathKey{indent: indent, path: pathStr}
			pathToMergeKeys[key] = append(pathToMergeKeys[key], trimmed)
		}
	}

	// Track processed keys
	processedKeys := make(map[pathKey]bool)

	// Second pass: reconstruct the YAML
	currentPath = []string{}
	previousIndent = 0

	for _, line := range lines {
		if line == "" || strings.TrimSpace(line) == "" {
			resultLines = append(resultLines, line)
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)

		// Update path tracking (same logic as first pass)
		if indent < previousIndent {
			levelsToRemove := (previousIndent - indent) / 2
			if len(currentPath) >= levelsToRemove {
				currentPath = currentPath[:len(currentPath)-levelsToRemove]
			} else {
				currentPath = []string{}
			}
		} else if indent > previousIndent && !strings.HasPrefix(trimmed, "<<:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) > 0 {
				currentPath = append(currentPath, parts[0])
			}
		}

		previousIndent = indent

		if strings.HasPrefix(trimmed, "<<:") {
			pathStr := strings.Join(currentPath, ".")
			key := pathKey{indent: indent, path: pathStr}
			mergeKeys := pathToMergeKeys[key]

			if len(mergeKeys) > 1 && !processedKeys[key] {
				// Create a line with an array of merge keys
				spaces := strings.Repeat(" ", indent)
				mergeKeysLine := spaces + "<<: [" + strings.Join(mergeKeys, ", ") + "]"
				resultLines = append(resultLines, mergeKeysLine)
				processedKeys[key] = true
			} else if len(mergeKeys) == 1 || !processedKeys[key] {
				resultLines = append(resultLines, line)
				processedKeys[key] = true
			}
			// Skip already processed merge keys
		} else {
			resultLines = append(resultLines, line)
		}
	}

	return strings.Join(resultLines, "\n")
}

// ParseYAMLWithAnchors parses a YAML file with anchor references from specified directories
func ParseYAMLWithAnchors(yamlFile string, anchorDirs []string) (map[string]interface{}, error) {
	// Read the YAML file
	yamlData, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file %s: %w", yamlFile, err)
	}

	// Process the YAML text to handle multiple merge keys first
	processedYAML := string(yamlData)
	decoder := NewCustomDecoder(processedYAML, yaml.ReferenceDirs(anchorDirs...))

	// Parse the YAML with anchors
	result, err := decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	return result, nil
}

// ParseStack parses a stack YAML file using the catalog directory for anchors
func ParseStack(stackFile string, catalogDir string) (map[string]interface{}, error) {
	// Find all subdirectories in the catalog directory
	subdirs, err := FindSubdirectories(catalogDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find subdirectories: %w", err)
	}

	// Add the catalog directory itself to the list
	subdirs = append([]string{catalogDir}, subdirs...)

	// Parse the YAML file with anchors
	result, err := ParseYAMLWithAnchors(stackFile, subdirs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML with anchors: %w", err)
	}

	return result, nil
}

// MergeYAML reads a stack YAML file and returns the merged content
func MergeYAML(stackFile string, catalogDir string) ([]byte, error) {
	// Parse the stack file with anchors
	merged, err := ParseStack(stackFile, catalogDir)
	if err != nil {
		return nil, err
	}

	// Marshal the result back to YAML
	result, err := yaml.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged YAML: %w", err)
	}

	return result, nil
}
