package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/mcalhoun/skunk/internal/logger"
	stackfinder "github.com/mcalhoun/skunk/internal/stack-finder"
	tablerender "github.com/mcalhoun/skunk/internal/table-render"
	"github.com/mcalhoun/skunk/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
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
		filters, err := cmd.Flags().GetStringArray("filter")
		if err != nil {
			logger.Log.Fatalf("Error getting filters: %v", err)
		}

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

		// Otherwise, print pretty table output with the tablerender package
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

// printBubblesTable prints the stacks using the tablerender package
func printBubblesTable(stacks []stackfinder.StackMetadata) {
	// Convert stacks to rows
	rows := make([][]string, 0, len(stacks))
	for _, stack := range stacks {
		// Get the relative path for display
		relPath := stack.FilePath
		if absPath, err := filepath.Abs(stack.FilePath); err == nil {
			if rel, err := filepath.Rel(".", absPath); err == nil {
				relPath = rel
			}
		}

		rows = append(rows, []string{stack.Name, relPath})
	}

	// Setup table style
	style := tablerender.DefaultTableStyle()
	style.Title = "STACKS"

	// Render and print the table
	table := tablerender.RenderTable([]string{"NAME", "PATH"}, rows, style)
	fmt.Println(table)
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
