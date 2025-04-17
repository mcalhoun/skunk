package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/mcalhoun/skunk/internal/logger"
	stackfinder "github.com/mcalhoun/skunk/internal/stack-finder"
	"github.com/mcalhoun/skunk/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Color definitions
	purple   = lipgloss.Color("99")  // Bright purple for headers and borders
	gray     = lipgloss.Color("245") // Light gray for text
	darkGray = lipgloss.Color("236") // Dark background

	// Style definitions for table
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(purple).
			MarginBottom(1).
			MarginTop(1)

	headerStyle = lipgloss.NewStyle().
			Foreground(purple).
			Bold(true).
			Align(lipgloss.Center)

	cellStyle    = lipgloss.NewStyle().Padding(0, 1).Width(14)
	oddRowStyle  = cellStyle.Copy().Foreground(gray)
	evenRowStyle = cellStyle.Copy().Foreground(gray)

	// Command flags
	jsonOutput bool
	noColor    bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources",
	Long:  `Lists various resources such as stacks.`,
}

// listStacksCmd represents the list stacks command
var listStacksCmd = &cobra.Command{
	Use:   "stacks",
	Short: "List all stacks",
	Long:  `List all stacks that match the configured stacksPath glob pattern.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get filters from flags
		filters, _ := cmd.Flags().GetStringArray("filter")

		// Get stacksPath from config
		stacksPath := viper.GetString("stacksPath")
		if stacksPath == "" {
			logger.Log.Fatalf("Error: stacksPath not defined in config")
		}

		// Find stacks
		stacks, err := stackfinder.FindStacks(stacksPath)
		if err != nil {
			logger.Log.Fatalf("Error finding stacks: %v", err)
		}

		if len(stacks) == 0 {
			logger.Log.Infof("No stacks found matching pattern: %s", stacksPath)
			return
		}

		// find duplicate stacks
		duplicates := utils.FindDuplicateStacks(stacks)
		if len(duplicates) > 0 {
			for stackName, stackFiles := range duplicates {
				filesWithBrackets := "[" + strings.Join(stackFiles, ", ") + "]"
				logger.Log.Error("duplicate stack detected",
					"stack", stackName,
					"error", "Stacks must have unique names",
					"files_count", len(stackFiles),
					"files", filesWithBrackets)
			}
			os.Exit(1)
		}

		// Apply filters if any are specified
		if len(filters) > 0 {
			stacks = utils.FilterStacks(stacks, filters)
			if len(stacks) == 0 {
				logger.Log.Info("No stacks match the specified filters")
				return
			}
		}

		// If JSON output is requested, print as JSON and exit
		if jsonOutput {
			outputJSON(stacks)
			return
		}

		// If no-color is specified, use the plain table format
		if noColor {
			printStandardTable(stacks)
			return
		}

		// Otherwise, print pretty table output with the bubbles table component
		printBubblesTable(stacks)
	},
}

// outputJSON prints the stacks as JSON
func outputJSON(stacks []stackfinder.StackMetadata) {
	type jsonOutput struct {
		Name     string            `json:"name"`
		Path     string            `json:"path"`
		Labels   map[string]string `json:"labels,omitempty"`
		FilePath string            `json:"filePath"` // Original full path
	}

	var output []jsonOutput
	for _, stack := range stacks {
		// Convert to relative path from current directory if possible
		relPath := stack.FilePath
		if absPath, err := filepath.Abs(stack.FilePath); err == nil {
			if rel, err := filepath.Rel(".", absPath); err == nil {
				relPath = rel
			}
		}

		output = append(output, jsonOutput{
			Name:     stack.Name,
			Path:     relPath,
			Labels:   stack.Labels,
			FilePath: stack.FilePath,
		})
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		logger.Log.Fatalf("Error marshaling to JSON: %v", err)
	}

	fmt.Println(string(jsonData))
}

// printBubblesTable prints the stacks using the Bubbles table component
func printBubblesTable(stacks []stackfinder.StackMetadata) {
	// Print the title
	title := titleStyle.Render("STACKS")
	fmt.Println(title)

	// Define table columns
	columns := []table.Column{
		{Title: "NAME", Width: 30},
		{Title: "PATH", Width: 50},
	}

	// Prepare rows data
	rows := []table.Row{}
	for _, stack := range stacks {
		// Get the relative path for display
		relPath := stack.FilePath
		if absPath, err := filepath.Abs(stack.FilePath); err == nil {
			if rel, err := filepath.Rel(".", absPath); err == nil {
				relPath = rel
			}
		}

		rows = append(rows, table.Row{stack.Name, relPath})
	}

	// Create and configure the table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)),
	)

	// Create a custom border style that matches the image
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(purple)

	// Style the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(purple).
		BorderBottom(true).
		Bold(true).
		Foreground(purple).
		Align(lipgloss.Center).
		Padding(0, 1)

	s.Selected = s.Selected.
		Foreground(lipgloss.NoColor{}).
		Background(lipgloss.NoColor{}).
		Bold(false)

	s.Cell = s.Cell.
		Foreground(gray)

	// Apply styles
	t.SetStyles(s)

	// Wrap the table in a border
	renderedTable := t.View()

	// Apply outer border to match the image
	finalTable := borderStyle.Render(renderedTable)

	// Render the table
	fmt.Println(finalTable)
}

// printStandardTable prints the stacks as a plain text table without any styling
func printStandardTable(stacks []stackfinder.StackMetadata) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Println("Stack Resources")
	fmt.Println()
	fmt.Fprintln(w, "Name\tPath")
	fmt.Fprintln(w, "----\t----")

	for _, stack := range stacks {
		// Get the relative path for display
		relPath := stack.FilePath
		if absPath, err := filepath.Abs(stack.FilePath); err == nil {
			if rel, err := filepath.Rel(".", absPath); err == nil {
				relPath = rel
			}
		}

		fmt.Fprintf(w, "%s\t%s\n", stack.Name, relPath)
	}

	w.Flush()
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listStacksCmd)

	// Add flags
	listStacksCmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON instead of a table")
	listStacksCmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
	listStacksCmd.Flags().StringArray("filter", []string{}, "filter stacks by label (format: key=value or key!=value), can be specified multiple times")
}
