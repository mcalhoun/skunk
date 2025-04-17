package tablerender

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

// Style configuration for consistent table rendering
type TableStyle struct {
	TotalWidth    int
	FirstColWidth int
	HeaderColor   lipgloss.Color
	TextColor     lipgloss.Color
	Title         string
}

// Default style settings
func DefaultTableStyle() TableStyle {
	// Get maxTableWidth from config, default to 80 if not set
	maxWidth := viper.GetInt("maxTableWidth")
	if maxWidth <= 0 {
		maxWidth = 80
	}

	// Calculate first column width proportionally (40% of total)
	firstColWidth := maxWidth * 2 / 5

	return TableStyle{
		TotalWidth:    maxWidth,
		FirstColWidth: firstColWidth,
		HeaderColor:   lipgloss.Color("99"),  // Purple
		TextColor:     lipgloss.Color("245"), // Light gray
		Title:         "",
	}
}

// RenderTable renders a table with the provided data and styling
func RenderTable(headers []string, rows [][]string, style TableStyle) string {
	// Get maxTableWidth from config
	maxWidth := viper.GetInt("maxTableWidth")
	if maxWidth <= 0 {
		maxWidth = 80
	}

	// Ensure table doesn't exceed the max width
	if style.TotalWidth > maxWidth {
		style.TotalWidth = maxWidth
		// Recalculate first column width proportionally
		style.FirstColWidth = maxWidth * 2 / 5
	}

	// Set default style if not provided
	if style.TotalWidth == 0 {
		style.TotalWidth = maxWidth
	}
	if style.FirstColWidth == 0 {
		style.FirstColWidth = maxWidth * 2 / 5
	}
	if style.HeaderColor == "" {
		style.HeaderColor = lipgloss.Color("99") // Purple
	}
	if style.TextColor == "" {
		style.TextColor = lipgloss.Color("245") // Light gray
	}

	// Calculate second column width
	secondColWidth := style.TotalWidth - style.FirstColWidth - 3 // Account for borders and padding

	// Define table columns
	columns := []table.Column{
		{Title: headers[0], Width: style.FirstColWidth},
		{Title: headers[1], Width: secondColWidth},
	}

	// Prepare table rows
	tableRows := []table.Row{}
	for _, row := range rows {
		tableRows = append(tableRows, table.Row{row[0], row[1]})
	}

	// Create and configure the table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(tableRows),
		table.WithFocused(false),
		table.WithHeight(len(tableRows)),
		table.WithWidth(style.TotalWidth),
	)

	// Create title style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(style.HeaderColor).
		MarginBottom(1).
		MarginTop(1)

	// Style the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(style.HeaderColor).
		BorderBottom(true).
		Bold(true).
		Foreground(style.HeaderColor).
		Align(lipgloss.Center).
		Padding(0, 1)

	s.Selected = s.Selected.
		Foreground(lipgloss.NoColor{}).
		Background(lipgloss.NoColor{}).
		Bold(false)

	s.Cell = s.Cell.
		Foreground(style.TextColor)

	// Apply styles
	t.SetStyles(s)

	// Create a consistent border style
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(style.HeaderColor).
		BorderTop(true).
		BorderRight(true).
		BorderBottom(true).
		BorderLeft(true).
		Padding(0, 0)

	// Render table with border
	renderedTable := t.View()
	finalTable := borderStyle.Render(renderedTable)

	// Add title if provided
	if style.Title != "" {
		title := titleStyle.Render(style.Title)
		return title + "\n" + finalTable
	}

	return finalTable
}

// Format a key-value data structure into rows for table rendering
func FormatKeyValueData(data map[string]interface{}) [][]string {
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
		valueStr := formatValue(value)
		rows = append(rows, []string{key, valueStr})
	}

	return rows
}

// formatValue formats a value for display, handling complex types
func formatValue(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case bool:
		return fmt.Sprintf("%v", v)
	case string:
		return v
	case map[string]interface{}:
		return formatMap(v)
	case []interface{}:
		return formatSlice(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Format a map for display
func formatMap(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}

	// Sort keys for deterministic order
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := "{ "
	for i, k := range keys {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s: %s", k, formatValue(m[k]))
	}
	result += " }"

	return result
}

// Format a slice for display
func formatSlice(s []interface{}) string {
	if len(s) == 0 {
		return "[]"
	}

	result := "[ "
	for i, v := range s {
		if i > 0 {
			result += ", "
		}
		result += formatValue(v)
	}
	result += " ]"

	return result
}
