package tablerender

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

func init() {
	// Set a fixed maxTableWidth for tests
	viper.Set("maxTableWidth", 80)
}

// ANSI color code pattern
var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// Strip ANSI color codes from a string
func stripAnsiCodes(s string) string {
	return ansiEscapePattern.ReplaceAllString(s, "")
}

// Helper function to save/load snapshots
func compareWithSnapshot(t *testing.T, name string, actual string) {
	t.Helper()

	// Create snapshots dir if it doesn't exist
	snapshotDir := "testdata/snapshots"
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		t.Fatalf("Failed to create snapshot directory: %v", err)
	}

	snapshotFile := filepath.Join(snapshotDir, fmt.Sprintf("%s.snapshot", name))

	// Strip ANSI color codes for visual comparison
	strippedActual := stripAnsiCodes(actual)

	// Update mode: Save current output as the new expected output
	if os.Getenv("UPDATE_SNAPSHOTS") == "1" {
		if err := os.WriteFile(snapshotFile, []byte(strippedActual), 0644); err != nil {
			t.Fatalf("Failed to write snapshot file: %v", err)
		}
		t.Logf("Updated snapshot: %s", name)
		return
	}

	// Normal mode: Compare with existing snapshot
	expected, err := os.ReadFile(snapshotFile)
	if os.IsNotExist(err) {
		t.Fatalf("Snapshot file doesn't exist. Run with UPDATE_SNAPSHOTS=1 to create it: %s", snapshotFile)
	} else if err != nil {
		t.Fatalf("Failed to read snapshot file: %v", err)
	}

	// Compare the stripped actual with the snapshot
	if string(expected) != strippedActual {
		t.Errorf("Output doesn't match snapshot.\nExpected:\n%s\n\nActual:\n%s", string(expected), strippedActual)
	}
}

func TestRenderTable_Basic(t *testing.T) {
	headers := []string{"NAME", "VALUE"}
	rows := [][]string{
		{"item1", "value1"},
		{"item2", "value2"},
		{"item3", "value3"},
	}

	style := DefaultTableStyle()
	style.Title = "BASIC TABLE"

	result := RenderTable(headers, rows, style)
	compareWithSnapshot(t, "basic_table", result)
}

func TestRenderTable_LongValues(t *testing.T) {
	headers := []string{"NAME", "VALUE"}
	rows := [][]string{
		{"short_name", "short value"},
		{"long_name_that_might_get_truncated", "This is a very long value that might get truncated or wrapped depending on the table rendering logic"},
		{"another_name", "Another normal value"},
	}

	style := DefaultTableStyle()
	style.Title = "LONG VALUES TABLE"

	result := RenderTable(headers, rows, style)
	compareWithSnapshot(t, "long_values_table", result)
}

func TestRenderTable_EmptyTable(t *testing.T) {
	headers := []string{"NAME", "VALUE"}
	var rows [][]string // Empty rows

	style := DefaultTableStyle()
	style.Title = "EMPTY TABLE"

	result := RenderTable(headers, rows, style)
	compareWithSnapshot(t, "empty_table", result)
}

func TestRenderTable_CustomStyle(t *testing.T) {
	headers := []string{"CUSTOM", "STYLING"}
	rows := [][]string{
		{"item1", "value1"},
		{"item2", "value2"},
	}

	style := TableStyle{
		TotalWidth:    70, // Narrower
		FirstColWidth: 25,
		HeaderColor:   lipgloss.Color("170"), // Different color
		TextColor:     lipgloss.Color("252"), // Different text color
		Title:         "CUSTOM STYLED TABLE",
	}

	result := RenderTable(headers, rows, style)
	compareWithSnapshot(t, "custom_style_table", result)
}

func TestFormatKeyValueData(t *testing.T) {
	// Create test data with various types
	data := map[string]interface{}{
		"string_value": "text",
		"int_value":    42,
		"bool_value":   true,
		"nil_value":    nil,
	}

	rows := FormatKeyValueData(data)

	// Convert to JSON for snapshot comparison (ordering issues)
	jsonData, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	compareWithSnapshot(t, "basic_key_value_format", string(jsonData))
}

func TestFormatKeyValueData_ComplexTypes(t *testing.T) {
	// Create complex nested data structure
	data := map[string]interface{}{
		"nested_map": map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
		"array_value": []interface{}{"item1", "item2", 3},
		"nested_array": []interface{}{
			map[string]interface{}{"name": "obj1"},
			map[string]interface{}{"name": "obj2"},
		},
	}

	rows := FormatKeyValueData(data)

	// Convert to JSON for snapshot comparison
	jsonData, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	compareWithSnapshot(t, "complex_key_value_format", string(jsonData))
}

func TestComplexTable(t *testing.T) {
	// Test rendering a table with complex values
	headers := []string{"PROPERTY", "VALUE"}

	// Create complex data
	data := map[string]interface{}{
		"simple_string": "text",
		"long_array": []interface{}{
			"item1", "item2", "item3", "item4", "item5",
			"item6", "item7", "item8", "item9", "item10",
		},
		"nested_map": map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"nested": map[string]interface{}{
				"inner": "value",
			},
		},
		"deeply_nested": map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": map[string]interface{}{
						"final": "very deep value",
					},
				},
			},
		},
	}

	// Format the data and render the table
	rows := FormatKeyValueData(data)

	style := DefaultTableStyle()
	style.Title = "COMPLEX DATA TABLE"

	result := RenderTable(headers, rows, style)
	compareWithSnapshot(t, "complex_data_table", result)
}

// Test extremely wide tables (regression test for truncation)
func TestWideTable(t *testing.T) {
	headers := []string{"NAME", "VERY WIDE VALUE"}

	// Create a row with extremely wide content
	wideValue := ""
	for i := 0; i < 150; i++ {
		wideValue += fmt.Sprintf("word%d ", i)
	}

	rows := [][]string{
		{"wide_row", wideValue},
	}

	style := DefaultTableStyle()
	style.Title = "WIDE TABLE TEST"

	result := RenderTable(headers, rows, style)
	compareWithSnapshot(t, "wide_table", result)
}

// Test extremely tall tables
func TestTallTable(t *testing.T) {
	headers := []string{"INDEX", "VALUE"}

	// Create many rows
	var rows [][]string
	for i := 0; i < 50; i++ {
		rows = append(rows, []string{
			fmt.Sprintf("row_%d", i),
			fmt.Sprintf("value for row %d", i),
		})
	}

	style := DefaultTableStyle()
	style.Title = "TALL TABLE TEST"

	result := RenderTable(headers, rows, style)
	compareWithSnapshot(t, "tall_table", result)
}
