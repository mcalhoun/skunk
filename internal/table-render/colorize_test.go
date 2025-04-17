package tablerender

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Store original environment variable state
	originalCliColorForce := os.Getenv("CLICOLOR_FORCE")
	originalNoColor := os.Getenv("NO_COLOR")

	// Set a fixed maxTableWidth for tests
	viper.Set("maxTableWidth", 80)

	// Run tests
	code := m.Run()

	// Restore original environment variables
	if originalCliColorForce != "" {
		os.Setenv("CLICOLOR_FORCE", originalCliColorForce)
	} else {
		os.Unsetenv("CLICOLOR_FORCE")
	}

	if originalNoColor != "" {
		os.Setenv("NO_COLOR", originalNoColor)
	} else {
		os.Unsetenv("NO_COLOR")
	}

	// Exit with the test status code
	os.Exit(code)
}

// Helper functions to control color mode for tests
func enableColorOutput() {
	os.Setenv("CLICOLOR_FORCE", "1")
	os.Unsetenv("NO_COLOR")
}

func disableColorOutput() {
	os.Unsetenv("CLICOLOR_FORCE")
	os.Setenv("NO_COLOR", "1")
}

func TestFormatValueWithColor(t *testing.T) {
	// Enable colors for this test
	enableColorOutput()

	// We test that each type gets different styling
	scheme := DefaultColorScheme()

	// Test different value types
	testCases := []struct {
		name  string
		value interface{}
	}{
		{"string", "test string"},
		{"number", 42},
		{"bool_true", true},
		{"bool_false", false},
		{"nil", nil},
		{"map", map[string]interface{}{"key": "value"}},
		{"array", []interface{}{"item1", "item2"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatted := FormatValueWithColor(tc.value, scheme)

			// The formatted value should contain ANSI color codes
			if !strings.Contains(formatted, "\x1b[") {
				t.Errorf("Expected ANSI color codes in formatting, but none found: %q", formatted)
			}

			// The formatted value should contain the actual value
			strippedFormatted := stripAnsiCodes(formatted)
			switch v := tc.value.(type) {
			case nil:
				if strippedFormatted != "null" {
					t.Errorf("Expected stripped value to be 'null', got: %q", strippedFormatted)
				}
			case string:
				if strippedFormatted != v {
					t.Errorf("Expected stripped value to be %q, got: %q", v, strippedFormatted)
				}
			case bool:
				expected := fmt.Sprintf("%v", v)
				if strippedFormatted != expected {
					t.Errorf("Expected stripped value to be %q, got: %q", expected, strippedFormatted)
				}
			case int:
				expected := fmt.Sprintf("%d", v)
				if strippedFormatted != expected {
					t.Errorf("Expected stripped value to be %q, got: %q", expected, strippedFormatted)
				}
			default:
				// For complex types, just check that the value isn't empty
				if strippedFormatted == "" {
					t.Errorf("Expected non-empty stripped value for %v", tc.value)
				}
			}
		})
	}
}

// TestFormatValueWithoutColor tests that color is disabled in non-TTY mode
func TestFormatValueWithoutColor(t *testing.T) {
	// Disable colors for this test
	disableColorOutput()
	defer enableColorOutput() // Restore colors for other tests

	scheme := DefaultColorScheme()

	// Test different value types
	testCases := []struct {
		name  string
		value interface{}
	}{
		{"string", "test string"},
		{"number", 42},
		{"bool_true", true},
		{"bool_false", false},
		{"nil", nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatted := FormatValueWithColor(tc.value, scheme)

			// The formatted value should NOT contain ANSI color codes
			if strings.Contains(formatted, "\x1b[") {
				t.Errorf("Expected no ANSI color codes in non-TTY mode, but found some: %q", formatted)
			}

			// The value should be rendered correctly
			switch v := tc.value.(type) {
			case nil:
				if formatted != "null" {
					t.Errorf("Expected value to be 'null', got: %q", formatted)
				}
			case string:
				if formatted != v {
					t.Errorf("Expected value to be %q, got: %q", v, formatted)
				}
			case bool:
				expected := fmt.Sprintf("%v", v)
				if formatted != expected {
					t.Errorf("Expected value to be %q, got: %q", expected, formatted)
				}
			case int:
				expected := fmt.Sprintf("%d", v)
				if formatted != expected {
					t.Errorf("Expected value to be %q, got: %q", expected, formatted)
				}
			}
		})
	}
}

func TestFormatKeyValueDataWithColor(t *testing.T) {
	// Enable colors for this test
	enableColorOutput()

	data := map[string]interface{}{
		"string":    "text",
		"number":    42,
		"boolean":   true,
		"nil_value": nil,
		"map":       map[string]interface{}{"key": "value"},
		"array":     []interface{}{1, 2, 3},
	}

	scheme := DefaultColorScheme()
	rows := FormatKeyValueDataWithColor(data, scheme)

	// Check we have the right number of rows
	if len(rows) != len(data) {
		t.Errorf("Expected %d rows, got %d", len(data), len(rows))
	}

	// Collect keys for verification
	foundKeys := make(map[string]bool)

	// Check all rows have 2 columns and the value contains color codes
	for _, row := range rows {
		if len(row) != 2 {
			t.Errorf("Expected 2 columns per row, got %d", len(row))
			continue
		}

		// Mark this key as found
		key := row[0]
		foundKeys[key] = true

		// Value column should contain ANSI color codes
		valueCol := row[1]
		if !strings.Contains(valueCol, "\x1b[") {
			t.Errorf("Expected ANSI color codes in value column for key %q, but none found: %q",
				key, valueCol)
		}
	}

	// Make sure all keys were included
	for key := range data {
		if !foundKeys[key] {
			t.Errorf("Key %q was not found in the formatted output", key)
		}
	}
}

// TestFormatKeyValueDataWithoutColor tests that color is disabled in non-TTY mode
func TestFormatKeyValueDataWithoutColor(t *testing.T) {
	// Disable colors for this test
	disableColorOutput()
	defer enableColorOutput() // Restore colors for other tests

	data := map[string]interface{}{
		"string":    "text",
		"number":    42,
		"boolean":   true,
		"nil_value": nil,
	}

	scheme := DefaultColorScheme()
	rows := FormatKeyValueDataWithColor(data, scheme)

	// Check all rows have 2 columns and the value does NOT contain color codes
	for _, row := range rows {
		if len(row) != 2 {
			t.Errorf("Expected 2 columns per row, got %d", len(row))
			continue
		}

		key := row[0]
		valueCol := row[1]

		// Value column should NOT contain ANSI color codes
		if strings.Contains(valueCol, "\x1b[") {
			t.Errorf("Expected no ANSI color codes in non-TTY mode for key %q, but found some: %q",
				key, valueCol)
		}
	}
}

func TestColorTableIntegration(t *testing.T) {
	// Enable colors for this test
	enableColorOutput()

	// Test that colored values can be used in tables
	data := map[string]interface{}{
		"string":    "text",
		"number":    42,
		"boolean":   true,
		"nil_value": nil,
		"map":       map[string]interface{}{"key": "value"},
		"array":     []interface{}{1, 2, 3},
	}

	scheme := DefaultColorScheme()
	rows := FormatKeyValueDataWithColor(data, scheme)

	style := DefaultTableStyle()
	style.Title = "COLORED TABLE TEST"

	result := RenderTable([]string{"KEY", "VALUE"}, rows, style)
	compareWithSnapshot(t, "colored_table", result)
}
