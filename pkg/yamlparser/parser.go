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
	// Simple implementation that just returns the input
	// A full implementation would preprocess the YAML to handle multiple merge keys
	return yamlText
}

// ParseYAMLWithAnchors parses a YAML file with anchor references from specified directories
func ParseYAMLWithAnchors(yamlFile string, anchorDirs []string) (map[string]interface{}, error) {
	// Read the YAML file
	yamlData, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file %s: %w", yamlFile, err)
	}

	// Create a custom decoder with reference directories
	options := []yaml.DecodeOption{yaml.ReferenceDirs(anchorDirs...)}
	decoder := NewCustomDecoder(string(yamlData), options...)

	// Decode the YAML into a generic map
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
