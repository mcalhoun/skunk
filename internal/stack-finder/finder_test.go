package stackfinder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindStacks(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "stack-finder-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test stack files
	stackFile1 := filepath.Join(tmpDir, "stack1.yaml")
	createTestStackFile(t, stackFile1, "stack1", map[string]string{"env": "dev", "region": "us-east-1"})

	stackFile2 := filepath.Join(tmpDir, "stack2.yaml")
	createTestStackFile(t, stackFile2, "stack2", map[string]string{"env": "prod", "region": "us-west-2"})

	// Create a non-stack YAML file
	nonStackFile := filepath.Join(tmpDir, "non-stack.yaml")
	createNonStackFile(t, nonStackFile)

	// Create a non-YAML file
	nonYAMLFile := filepath.Join(tmpDir, "not-yaml.txt")
	if err := os.WriteFile(nonYAMLFile, []byte("not a YAML file"), 0644); err != nil {
		t.Fatalf("Failed to write non-YAML file: %v", err)
	}

	// Test FindStacks with a glob pattern
	stacks, err := FindStacks(filepath.Join(tmpDir, "*.yaml"))
	if err != nil {
		t.Fatalf("FindStacks failed: %v", err)
	}

	// Verify we found exactly 2 stacks
	if len(stacks) != 2 {
		t.Errorf("Expected 2 stacks, got %d", len(stacks))
	}

	// Verify the stack metadata
	for _, stack := range stacks {
		switch stack.Name {
		case "stack1":
			if stack.Labels["env"] != "dev" || stack.Labels["region"] != "us-east-1" {
				t.Errorf("stack1 has incorrect labels: %v", stack.Labels)
			}
			if stack.FilePath != stackFile1 {
				t.Errorf("stack1 has incorrect file path: %s, expected: %s", stack.FilePath, stackFile1)
			}
		case "stack2":
			if stack.Labels["env"] != "prod" || stack.Labels["region"] != "us-west-2" {
				t.Errorf("stack2 has incorrect labels: %v", stack.Labels)
			}
			if stack.FilePath != stackFile2 {
				t.Errorf("stack2 has incorrect file path: %s, expected: %s", stack.FilePath, stackFile2)
			}
		default:
			t.Errorf("Unexpected stack name: %s", stack.Name)
		}
	}

	// Test FindStacksRecursive
	// Create a subdirectory with another stack
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	stackFile3 := filepath.Join(subDir, "stack3.yaml")
	createTestStackFile(t, stackFile3, "stack3", map[string]string{"env": "test"})

	// Find stacks recursively
	recursiveStacks, err := FindStacksRecursive(tmpDir)
	if err != nil {
		t.Fatalf("FindStacksRecursive failed: %v", err)
	}

	// Verify we found exactly 3 stacks
	if len(recursiveStacks) != 3 {
		t.Errorf("Expected 3 stacks with recursive search, got %d", len(recursiveStacks))
	}

	// Verify the stack3 metadata
	foundStack3 := false
	for _, stack := range recursiveStacks {
		if stack.Name == "stack3" {
			foundStack3 = true
			if stack.Labels["env"] != "test" {
				t.Errorf("stack3 has incorrect label: %v", stack.Labels)
			}
			if stack.FilePath != stackFile3 {
				t.Errorf("stack3 has incorrect file path: %s, expected: %s", stack.FilePath, stackFile3)
			}
		}
	}

	if !foundStack3 {
		t.Errorf("stack3 was not found in recursive search")
	}
}

func createTestStackFile(t *testing.T, path, name string, labels map[string]string) {
	yamlContent := `apiVersion: atmos.cloudposse.com/v1
kind: Stack
metadata:
  name: ` + name + `
  labels:
`
	for k, v := range labels {
		yamlContent += "    " + k + ": " + v + "\n"
	}

	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write stack file %s: %v", path, err)
	}
}

func createNonStackFile(t *testing.T, path string) {
	yamlContent := `apiVersion: v1
kind: Deployment
metadata:
  name: sample-deployment
`
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write non-stack file: %v", err)
	}
}
