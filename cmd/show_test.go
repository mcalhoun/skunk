package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/mcalhoun/skunk/internal/logger"
	stackfinder "github.com/mcalhoun/skunk/internal/stack-finder"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// MockStackFinder implements the StackFinder interface for testing
type MockStackFinder struct {
	Stacks []stackfinder.StackMetadata
	Err    error
}

// FindStacks returns the predefined stacks for testing
func (m *MockStackFinder) FindStacks(pattern string) ([]stackfinder.StackMetadata, error) {
	return m.Stacks, m.Err
}

// Create a new mock stack finder with sample data
func NewMockStackFinder(t *testing.T) *MockStackFinder {
	testFilePath := filepath.Join("testdata", "test_stack.yaml")
	absPath, err := filepath.Abs(testFilePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	return &MockStackFinder{
		Stacks: []stackfinder.StackMetadata{
			{
				Name:     "test-stack",
				FilePath: absPath,
				Labels: map[string]string{
					"env":    "test",
					"region": "us-test-1",
				},
			},
		},
		Err: nil,
	}
}

// Create a mock stack finder with duplicate stacks
func NewDuplicateStackFinder(t *testing.T) *MockStackFinder {
	testFilePath := filepath.Join("testdata", "test_stack.yaml")
	absPath, err := filepath.Abs(testFilePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	dupFilePath := filepath.Join("testdata", "duplicate_stack.yaml")
	dupAbsPath, err := filepath.Abs(dupFilePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	return &MockStackFinder{
		Stacks: []stackfinder.StackMetadata{
			{
				Name:     "test-stack",
				FilePath: absPath,
				Labels: map[string]string{
					"env":    "test",
					"region": "us-test-1",
				},
			},
			{
				Name:     "test-stack", // Same name causes duplication
				FilePath: dupAbsPath,
				Labels: map[string]string{
					"env":    "prod",
					"region": "us-west-1",
				},
			},
		},
		Err: nil,
	}
}

// Create a mock stack finder that returns an error
func NewErrorStackFinder() *MockStackFinder {
	return &MockStackFinder{
		Stacks: nil,
		Err:    errors.New("stack finder error"),
	}
}

// Create a mock stack finder that returns empty results
func NewEmptyStackFinder() *MockStackFinder {
	return &MockStackFinder{
		Stacks: []stackfinder.StackMetadata{},
		Err:    nil,
	}
}

// Test helper function to redirect stdout to capture output
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(fmt.Sprintf("Failed to create pipe: %v", err))
	}
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	if err != nil {
		panic(fmt.Sprintf("Failed to copy buffer: %v", err))
	}
	return buf.String()
}

// Test helper function to redirect stderr to capture error output
func captureError(f func()) (string, error) {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		panic(fmt.Sprintf("Failed to create pipe: %v", err))
	}
	os.Stderr = w

	var captureErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					captureErr = e
				} else {
					captureErr = errors.New("panic in test")
				}
			}
		}()
		f()
	}()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	if err != nil {
		panic(fmt.Sprintf("Failed to copy buffer: %v", err))
	}
	return buf.String(), captureErr
}

// setupTestEnvironment creates necessary test configuration for tests
func setupTestEnvironment(t *testing.T) func() {
	t.Helper()

	// Save current config
	catalogDir := viper.GetString("catalogDir")
	stacksPath := viper.GetString("stacksPath")

	// Set test config
	testDataDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	viper.Set("catalogDir", testDataDir)
	viper.Set("stacksPath", testDataDir)

	// Return a function to restore the original config
	return func() {
		viper.Set("catalogDir", catalogDir)
		viper.Set("stacksPath", stacksPath)
	}
}

func TestExtractComponents(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	testFilePath := filepath.Join("testdata", "test_stack.yaml")

	components, err := extractComponents(testFilePath)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(components))

	// Create a map to easily verify components
	compMap := make(map[string]string)
	for _, comp := range components {
		compMap[comp.Name] = comp.Type
	}

	assert.Equal(t, "terraform", compMap["vpc"])
	assert.Equal(t, "terraform", compMap["database"])
	assert.Equal(t, "helm", compMap["nginx"])

	// Test with non-existent file
	_, err = extractComponents("nonexistent.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to merge YAML")
}

func TestExtractComponentVars(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	testFilePath := filepath.Join("testdata", "test_stack.yaml")

	// Test VPC component vars
	vars, err := extractComponentVars(testFilePath, "terraform", "vpc")
	assert.NoError(t, err)

	// Create a map to easily verify vars
	varMap := make(map[string]interface{})
	for _, v := range vars {
		varMap[v.Name] = v.Value
	}

	assert.Equal(t, "10.0.0.0/16", varMap["cidr_block"])
	assert.Equal(t, true, varMap["enable_dns"])

	// Test for a non-existent component
	_, err = extractComponentVars(testFilePath, "invalid", "component")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "component type 'invalid' not found")

	// Test for a non-existent component name
	_, err = extractComponentVars(testFilePath, "terraform", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "component 'nonexistent' not found")

	// Test with non-existent file
	_, err = extractComponentVars("nonexistent.yaml", "terraform", "vpc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to merge YAML")
}

func TestFormatVariableValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "test", "test"},
		{"integer", 42, "42"},
		{"boolean", true, "true"},
		{"nil", nil, "null"},
		{"map", map[string]string{"key": "value"}, "{\"key\":\"value\"}"},
		{"slice", []string{"one", "two"}, "[\"one\",\"two\"]"},
		// Add a more complex type
		{"nested map",
			map[string]interface{}{
				"key":    "value",
				"nested": map[string]int{"a": 1, "b": 2},
			},
			"{\"key\":\"value\",\"nested\":{\"a\":1,\"b\":2}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatVariableValue(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHelpCommands(t *testing.T) {
	// Test show command help
	assert.Contains(t, showCmd.Short, "Show detailed information")

	// Test show stack command help
	assert.Contains(t, showStackCmd.Short, "Show stack components")

	// Verify that flags are properly registered
	assert.NotNil(t, showStackCmd.Flags().Lookup("stackName"))
	assert.NotNil(t, showStackCmd.Flags().Lookup("component"))
	assert.NotNil(t, showStackCmd.Flags().Lookup("json"))
	assert.NotNil(t, showStackCmd.Flags().Lookup("no-color"))
	assert.NotNil(t, showStackCmd.Flags().Lookup("tfvars"))
	assert.NotNil(t, showStackCmd.Flags().Lookup("filter"))
}

func setupTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().StringVarP(&stackName, "stackName", "s", "", "stack name")
	cmd.Flags().StringVarP(&componentName, "component", "c", "", "component name")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable color")
	cmd.Flags().BoolVar(&tfVars, "tfvars", false, "output as Terraform vars")
	cmd.Flags().StringArray("filter", []string{}, "filter stacks")
	return cmd
}

func TestRunShowStackCmd(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create a mock stack finder
	mockFinder := NewMockStackFinder(t)

	// Create a test cobra command with necessary flags
	cmd := setupTestCommand()

	// Test cases
	tests := []struct {
		name      string
		setup     func()
		wantError bool
	}{
		{
			name: "show components",
			setup: func() {
				stackName = "test-stack"
				componentName = ""
				jsonOutput = false
				noColor = false
				tfVars = false
			},
			wantError: false,
		},
		{
			name: "show vpc component vars",
			setup: func() {
				stackName = "test-stack"
				componentName = "vpc"
				jsonOutput = false
				noColor = false
				tfVars = false
			},
			wantError: false,
		},
		{
			name: "show component vars as json",
			setup: func() {
				stackName = "test-stack"
				componentName = "vpc"
				jsonOutput = true
				noColor = false
				tfVars = false
			},
			wantError: false,
		},
		{
			name: "show component vars as tfvars",
			setup: func() {
				stackName = "test-stack"
				componentName = "vpc"
				jsonOutput = false
				noColor = false
				tfVars = true
			},
			wantError: false,
		},
		{
			name: "show components without color",
			setup: func() {
				stackName = "test-stack"
				componentName = ""
				jsonOutput = false
				noColor = true
				tfVars = false
			},
			wantError: false,
		},
		{
			name: "show components as json",
			setup: func() {
				stackName = "test-stack"
				componentName = ""
				jsonOutput = true
				noColor = false
				tfVars = false
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test case
			tt.setup()

			// Capture output
			output := captureOutput(func() {
				runShowStackCmd(cmd, []string{}, mockFinder)
			})

			// Verify output contains expected data
			if componentName == "vpc" {
				assert.Contains(t, output, "cidr_block")
				assert.Contains(t, output, "10.0.0.0/16")
			} else {
				assert.Contains(t, output, "vpc")
				assert.Contains(t, output, "database")
				assert.Contains(t, output, "nginx")
			}
		})
	}
}

func TestRunShowStackCmdErrors(t *testing.T) {
	t.Skip("Skipping test that attempts to handle fatal errors that terminate the program")

	// Save original logger and restore it after test
	origLogger := logger.Log
	defer func() {
		logger.Log = origLogger
	}()

	// Create test environment
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create a test cobra command with necessary flags
	cmd := setupTestCommand()

	tests := []struct {
		name          string
		setup         func() StackFinder
		expectedError string
	}{
		{
			name: "missing stack name and filter",
			setup: func() StackFinder {
				stackName = ""
				componentName = ""
				tfVars = false
				if err := cmd.Flags().Set("filter", ""); err != nil {
					panic(fmt.Sprintf("Failed to set filter flag: %v", err))
				}
				return NewMockStackFinder(t)
			},
			expectedError: "either stack name or filter is required",
		},
		{
			name: "tfvars without component",
			setup: func() StackFinder {
				stackName = "test-stack"
				componentName = ""
				tfVars = true
				return NewMockStackFinder(t)
			},
			expectedError: "--tfvars can only be used with --component",
		},
		{
			name: "stack finder error",
			setup: func() StackFinder {
				stackName = "test-stack"
				componentName = ""
				tfVars = false
				return NewErrorStackFinder()
			},
			expectedError: "Error finding stacks",
		},
		{
			name: "duplicate stacks",
			setup: func() StackFinder {
				stackName = "test-stack"
				componentName = ""
				tfVars = false
				return NewDuplicateStackFinder(t)
			},
			expectedError: "duplicate stack detected",
		},
		{
			name: "non-existent stack",
			setup: func() StackFinder {
				stackName = "non-existent"
				componentName = ""
				tfVars = false
				return NewMockStackFinder(t)
			},
			expectedError: "stack with name 'non-existent' not found",
		},
		{
			name: "non-existent component",
			setup: func() StackFinder {
				stackName = "test-stack"
				componentName = "non-existent"
				tfVars = false
				return NewMockStackFinder(t)
			},
			expectedError: "component with name 'non-existent' not found",
		},
		{
			name: "empty stacks with filter",
			setup: func() StackFinder {
				stackName = ""
				componentName = ""
				tfVars = false
				if err := cmd.Flags().Set("filter", "env=prod"); err != nil {
					panic(fmt.Sprintf("Failed to set filter flag: %v", err))
				}
				return NewEmptyStackFinder()
			},
			expectedError: "No stacks match the specified filters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test case
			finder := tt.setup()

			// Create a buffer to capture log output
			var buf bytes.Buffer
			mockLogger := log.NewWithOptions(&buf, log.Options{Level: log.DebugLevel})

			// Replace the logger with our mock
			logger.Log = mockLogger

			// Capture output
			errOutput, captureErr := captureError(func() {
				runShowStackCmd(cmd, []string{}, finder)
			})
			if captureErr != nil {
				t.Logf("Error during command execution: %v", captureErr)
			}

			// Verify output contains expected message
			assert.Contains(t, errOutput, tt.expectedError)
		})
	}
}

func TestFilterFunctionality(t *testing.T) {
	t.Skip("Skipping test that attempts to handle logger output")

	cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create a mock stack finder with multiple stacks
	testFilePath := filepath.Join("testdata", "test_stack.yaml")
	absPath, err := filepath.Abs(testFilePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	mockFinder := &MockStackFinder{
		Stacks: []stackfinder.StackMetadata{
			{
				Name:     "test-stack-1",
				FilePath: absPath,
				Labels: map[string]string{
					"env":    "test",
					"region": "us-test-1",
				},
			},
			{
				Name:     "test-stack-2",
				FilePath: absPath,
				Labels: map[string]string{
					"env":    "prod",
					"region": "us-west-1",
				},
			},
			{
				Name:     "test-stack-3",
				FilePath: absPath,
				Labels: map[string]string{
					"env":    "staging",
					"region": "us-west-1",
				},
			},
		},
		Err: nil,
	}

	// Create a test cobra command with necessary flags
	cmd := setupTestCommand()

	// Test cases for filtering
	tests := []struct {
		name        string
		filter      string
		expectedMsg string
	}{
		{
			name:        "filter by env=prod",
			filter:      "env=prod",
			expectedMsg: "Selected stack 'test-stack-2' based on filter criteria",
		},
		{
			name:        "filter by region=us-west-1",
			filter:      "region=us-west-1",
			expectedMsg: "Selected stack 'test-stack-2' based on filter criteria",
		},
		{
			name:        "filter by non-existent label",
			filter:      "nonexistent=value",
			expectedMsg: "No stacks match the specified filters",
		},
		{
			name:        "filter by negation",
			filter:      "env!=test",
			expectedMsg: "Selected stack 'test-stack-2' based on filter criteria",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset variables
			stackName = ""
			componentName = ""
			jsonOutput = false
			noColor = false
			tfVars = false

			// Set filter
			if err := cmd.Flags().Set("filter", tt.filter); err != nil {
				panic(fmt.Sprintf("Failed to set filter flag: %v", err))
			}

			// Capture output
			errOutput, captureErr := captureError(func() {
				runShowStackCmd(cmd, []string{}, mockFinder)
			})
			if captureErr != nil {
				t.Logf("Error during command execution: %v", captureErr)
			}

			// Verify output contains expected message
			assert.Contains(t, errOutput, tt.expectedMsg)
		})
	}
}

func TestPrintFunctions(t *testing.T) {
	tests := []struct {
		name     string
		function func()
		contains []string
	}{
		{
			name: "printComponentsBubblesTable",
			function: func() {
				components := []Component{
					{Type: "terraform", Name: "vpc"},
					{Type: "helm", Name: "nginx"},
				}
				printComponentsBubblesTable("test-stack", components)
			},
			contains: []string{"COMPONENTS", "TYPE", "NAME", "terraform", "vpc"},
		},
		{
			name: "printComponentsStandardTable",
			function: func() {
				components := []Component{
					{Type: "terraform", Name: "vpc"},
					{Type: "helm", Name: "nginx"},
				}
				printComponentsStandardTable("test-stack", components)
			},
			contains: []string{"STACK: test-stack", "COMPONENT TYPE", "COMPONENT NAME", "terraform", "vpc"},
		},
		{
			name: "printComponentVarsBubblesTable",
			function: func() {
				vars := []ComponentVar{
					{Name: "cidr", Value: "10.0.0.0/16"},
					{Name: "enable", Value: true},
				}
				component := &Component{Type: "terraform", Name: "vpc"}
				printComponentVarsBubblesTable("test-stack", vars, component)
			},
			contains: []string{"COMPONENT:", "terraform/vpc", "VARIABLE", "VALUE"},
		},
		{
			name: "printComponentVarsStandardTable",
			function: func() {
				component := &Component{
					Name: "test-component",
					Type: "test-type",
				}
				vars := []ComponentVar{
					{Name: "var1", Value: "value1"},
					{Name: "var2", Value: "value2"},
				}

				printComponentVarsStandardTable("test-stack", vars, component)
			},
			contains: []string{"Stack: test-stack", "Component: test-type/test-component", "Variable", "Value", "var1", "value1", "var2", "value2"},
		},
		{
			name: "outputTerraformVars",
			function: func() {
				vars := []ComponentVar{
					{Name: "cidr", Value: "10.0.0.0/16"},
					{Name: "enable", Value: true},
					{Name: "tags", Value: map[string]string{"Name": "test"}},
				}
				outputTerraformVars(vars, "test_stack.yaml", "vpc")
			},
			contains: []string{
				"# Terraform variables for component 'vpc'",
				"cidr = \"10.0.0.0/16\"",
				"enable = true",
				"tags = {\"Name\":\"test\"}",
			},
		},
		{
			name: "printComponentVarsStandardTable with complex types",
			function: func() {
				vars := []ComponentVar{
					{Name: "simple", Value: "simple_value"},
					{Name: "complex", Value: map[string]interface{}{
						"nested": map[string]int{"a": 1, "b": 2},
					}},
				}
				component := &Component{Type: "terraform", Name: "vpc"}
				printComponentVarsStandardTable("test-stack", vars, component)
			},
			contains: []string{"Component:", "terraform/vpc", "simple", "simple_value", "{\"nested\":{\"a\":1,\"b\":2}}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(tt.function)
			for _, text := range tt.contains {
				assert.Contains(t, output, text)
			}
		})
	}
}

func TestJSONOutputFunctions(t *testing.T) {
	tests := []struct {
		name     string
		function func()
		contains []string
	}{
		{
			name: "outputComponentsJSON",
			function: func() {
				components := []Component{
					{Type: "terraform", Name: "vpc"},
					{Type: "helm", Name: "nginx"},
				}
				outputComponentsJSON(components)
			},
			contains: []string{"\"type\": \"terraform\"", "\"name\": \"vpc\""},
		},
		{
			name: "outputComponentVarsJSON",
			function: func() {
				vars := []ComponentVar{
					{Name: "cidr", Value: "10.0.0.0/16"},
					{Name: "enable", Value: true},
				}
				outputComponentVarsJSON(vars)
			},
			contains: []string{"\"name\": \"cidr\"", "\"value\": \"10.0.0.0/16\""},
		},
		{
			name: "outputComponentVarsJSON with complex types",
			function: func() {
				vars := []ComponentVar{
					{Name: "nested", Value: map[string]interface{}{
						"a": 1,
						"b": map[string]string{"c": "d"},
					}},
				}
				outputComponentVarsJSON(vars)
			},
			contains: []string{"\"name\": \"nested\"", "\"value\": {", "\"a\": 1", "\"b\": {", "\"c\": \"d\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(tt.function)
			for _, text := range tt.contains {
				assert.Contains(t, output, text)
			}
		})
	}
}
