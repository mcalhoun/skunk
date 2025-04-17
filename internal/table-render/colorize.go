package tablerender

import (
	"fmt"
	"os"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// ColorScheme defines colors for different data types
type ColorScheme struct {
	StringColor    lipgloss.Color
	NumberColor    lipgloss.Color
	BoolTrueColor  lipgloss.Color
	BoolFalseColor lipgloss.Color
	NullColor      lipgloss.Color
	MapColor       lipgloss.Color
	ArrayColor     lipgloss.Color
}

// DefaultColorScheme returns a default set of colors
func DefaultColorScheme() ColorScheme {
	return ColorScheme{
		StringColor:    lipgloss.Color("149"), // Light green
		NumberColor:    lipgloss.Color("170"), // Orange
		BoolTrueColor:  lipgloss.Color("76"),  // Green
		BoolFalseColor: lipgloss.Color("203"), // Red
		NullColor:      lipgloss.Color("245"), // Gray
		MapColor:       lipgloss.Color("105"), // Purple
		ArrayColor:     lipgloss.Color("39"),  // Blue
	}
}

// shouldUseColor determines whether to render with color based on environment variables and TTY status
func shouldUseColor() bool {
	// Environment variables take precedence over TTY detection

	// Check if NO_COLOR is set (standard way to disable color)
	if noColor := os.Getenv("NO_COLOR"); noColor != "" {
		return false
	}

	// Check if CLICOLOR_FORCE is set (forces color even in non-TTY)
	if forceColor := os.Getenv("CLICOLOR_FORCE"); forceColor != "" {
		return true
	}

	// Check if output is going to a terminal
	fd := int(os.Stdout.Fd())
	return term.IsTerminal(fd)
}

// FormatValueWithColor formats a value with appropriate coloring based on type
func FormatValueWithColor(value interface{}, scheme ColorScheme) string {
	// Check if we should use color
	useColor := shouldUseColor()

	if !useColor {
		if value == nil {
			return "null"
		}

		switch v := value.(type) {
		case bool:
			return fmt.Sprintf("%v", v)
		case string:
			return v
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return fmt.Sprintf("%v", v)
		case map[string]interface{}:
			return formatMap(v)
		case []interface{}:
			return formatSlice(v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}

	// Use colored output
	if value == nil {
		// Ensure we're explicitly rendering with noColor set to false
		return lipgloss.NewStyle().Foreground(scheme.NullColor).Render("null")
	}

	switch v := value.(type) {
	case bool:
		if v {
			// Explicitly force color rendering
			return lipgloss.NewStyle().Foreground(scheme.BoolTrueColor).Bold(false).Render(fmt.Sprintf("%v", v))
		}
		return lipgloss.NewStyle().Foreground(scheme.BoolFalseColor).Bold(false).Render(fmt.Sprintf("%v", v))

	case string:
		return lipgloss.NewStyle().Foreground(scheme.StringColor).Bold(false).Render(v)

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return lipgloss.NewStyle().Foreground(scheme.NumberColor).Bold(false).Render(fmt.Sprintf("%v", v))

	case map[string]interface{}:
		formattedMap := formatMap(v)
		return lipgloss.NewStyle().Foreground(scheme.MapColor).Bold(false).Render(formattedMap)

	case []interface{}:
		formattedSlice := formatSlice(v)
		return lipgloss.NewStyle().Foreground(scheme.ArrayColor).Bold(false).Render(formattedSlice)

	default:
		// For any other type, just convert to string without special coloring
		return fmt.Sprintf("%v", v)
	}
}

// FormatKeyValueDataWithColor formats a map to table rows with colored values
func FormatKeyValueDataWithColor(data map[string]interface{}, scheme ColorScheme) [][]string {
	rows := make([][]string, 0, len(data))

	// Sort keys for consistent output
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	// Sort keys for deterministic order
	sort.Strings(keys)

	// Format each entry as a row
	for _, key := range keys {
		value := data[key]
		valueStr := FormatValueWithColor(value, scheme)
		rows = append(rows, []string{key, valueStr})
	}

	return rows
}
