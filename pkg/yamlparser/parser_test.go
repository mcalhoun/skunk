package yamlparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreprocessMultipleMergeKeys(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Single merge key",
			input: `
vars:
  <<: *defaults
  name: value
`,
			expected: `
vars:
  <<: *defaults
  name: value
`,
		},
		{
			name: "Multiple merge keys",
			input: `
vars:
  <<: *defaults1
  <<: *defaults2
  name: value
`,
			expected: `
vars:
  <<: [<<: *defaults1, <<: *defaults2]
  name: value
`,
		},
		{
			name: "Multiple merge keys with different indentation",
			input: `
vars:
  <<: *defaults1
  <<: *defaults2
  name:
    <<: *nested1
    <<: *nested2
    key: value
`,
			expected: `
vars:
  <<: [<<: *defaults1, <<: *defaults2]
  name:
    <<: [<<: *nested1, <<: *nested2]
    key: value
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decoder := NewCustomDecoder(tc.input)
			result := decoder.preprocessMultipleMergeKeys(tc.input)
			if result != tc.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tc.expected, result)
			}
		})
	}
}

func TestMergeYAML(t *testing.T) {
	// Skip if not running in a complete environment
	if _, err := os.Stat("../fixtures"); os.IsNotExist(err) {
		t.Skip("fixtures directory not found, skipping integration test")
	}

	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "yamlparser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test anchor files
	anchorDir := filepath.Join(tmpDir, "anchors")
	if err := os.Mkdir(anchorDir, 0755); err != nil {
		t.Fatalf("Failed to create anchors dir: %v", err)
	}

	// Create a test anchor file
	anchorFile := filepath.Join(anchorDir, "test-anchor.yaml")
	if err := os.WriteFile(anchorFile, []byte(`
test-anchor: &test-anchor
  key1: value1
  key2: value2
`), 0644); err != nil {
		t.Fatalf("Failed to write anchor file: %v", err)
	}

	// Create a test YAML file that references the anchor
	yamlFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(yamlFile, []byte(`
root:
  child:
    <<: *test-anchor
    key3: value3
`), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse the YAML file with the anchor
	result, err := MergeYAML(yamlFile, anchorDir)
	if err != nil {
		t.Fatalf("Failed to merge YAML: %v", err)
	}

	// Verify the result
	expected := `root:
  child:
    key1: value1
    key2: value2
    key3: value3
`
	// Normalize line endings for comparison
	resultStr := string(result)
	if resultStr != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, resultStr)
	}
}
