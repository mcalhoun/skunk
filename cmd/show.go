package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
	"github.com/mcalhoun/skunk/internal/logger"
	stackfinder "github.com/mcalhoun/skunk/internal/stack-finder"
	"github.com/mcalhoun/skunk/internal/utils"
	"github.com/mcalhoun/skunk/pkg/yamlparser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Only declare variables that are specific to this file
var (
	stackName     string
	componentName string
	tfVars        bool
)

// ComponentVar represents a component variable
type ComponentVar struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// Component represents a component in a stack
type Component struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show detailed information",
	Long:  `Shows detailed information about various resources.`,
}

// showStackCmd represents the show stack command
var showStackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Show stack components",
	Long:  `Show detailed information about components in a specific stack.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get filters from flags
		filters, _ := cmd.Flags().GetStringArray("filter")

		if stackName == "" && len(filters) == 0 {
			logger.Log.Fatalf("Error: either stack name or filter is required. Use --stackName/-s or --filter")
		}

		// Validate that --tfvars is only used with --component
		if tfVars && componentName == "" {
			logger.Log.Fatalf("Error: --tfvars can only be used with --component")
		}

		// Get stacksPath from config
		stacksPath := viper.GetString("stacksPath")
		if stacksPath == "" {
			logger.Log.Fatalf("Error: stacksPath not defined in config")
		}

		// Find all stacks
		stacks, err := stackfinder.FindStacks(stacksPath)
		if err != nil {
			logger.Log.Fatalf("Error finding stacks: %v", err)
		}

		// Check for duplicate stack names
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

		var targetStack *stackfinder.StackMetadata

		// If filters are used without stackName, use the first matching stack
		if stackName == "" && len(stacks) > 0 {
			targetStack = &stacks[0]
			// Inform the user which stack was selected
			logger.Log.Infof("Selected stack '%s' based on filter criteria", targetStack.Name)
		} else {
			// Find the stack by name
			for i, stack := range stacks {
				if stack.Name == stackName {
					targetStack = &stacks[i]
					break
				}
			}
		}

		if targetStack == nil {
			if stackName != "" {
				logger.Log.Fatalf("Error: stack with name '%s' not found", stackName)
			} else {
				logger.Log.Fatal("Error: no matching stack found")
			}
		}

		// Parse the YAML file to extract components
		components, err := extractComponents(targetStack.FilePath)
		if err != nil {
			logger.Log.Fatalf("Error extracting components: %v", err)
		}

		if len(components) == 0 {
			logger.Log.Infof("No components found in stack '%s'", targetStack.Name)
			return
		}

		// If a specific component is requested, show its variables
		if componentName != "" {
			// Find the component
			var foundComponent *Component
			for i, comp := range components {
				if comp.Name == componentName {
					foundComponent = &components[i]
					break
				}
			}

			if foundComponent == nil {
				logger.Log.Fatalf("Error: component with name '%s' not found in stack '%s'", componentName, stackName)
			}

			// Extract component variables
			vars, err := extractComponentVars(targetStack.FilePath, foundComponent.Type, foundComponent.Name)
			if err != nil {
				logger.Log.Fatalf("Error extracting component variables: %v", err)
			}

			if len(vars) == 0 {
				logger.Log.Infof("No variables found for component '%s' in stack '%s'", componentName, stackName)
				return
			}

			// If tfvars output is requested, output in Terraform format
			if tfVars {
				outputTerraformVars(vars, filepath.Base(targetStack.FilePath), componentName)
				return
			}

			// If JSON output is requested, print as JSON and exit
			if jsonOutput {
				outputComponentVarsJSON(vars)
				return
			}

			// If no-color is specified, use the plain table format
			if noColor {
				printComponentVarsStandardTable(vars, foundComponent)
				return
			}

			// Otherwise, print pretty table output
			printComponentVarsBubblesTable(vars, foundComponent)
			return
		}

		// If no specific component is requested, show all components
		// If JSON output is requested, print as JSON and exit
		if jsonOutput {
			outputComponentsJSON(components)
			return
		}

		// If no-color is specified, use the plain table format
		if noColor {
			printComponentsStandardTable(components)
			return
		}

		// Otherwise, print pretty table output with the bubbles table component
		printComponentsBubblesTable(components)
	},
}

// extractComponents extracts components from a stack YAML file
func extractComponents(filePath string) ([]Component, error) {
	// Get catalogDir from config
	catalogDir := viper.GetString("catalogDir")
	if catalogDir == "" {
		// Default to fixtures/catalog if not specified
		catalogDir = "fixtures/catalog"
	}

	// Use our YAML parser that can handle anchors
	mergedYAML, err := yamlparser.MergeYAML(filePath, catalogDir)
	if err != nil {
		return nil, fmt.Errorf("failed to merge YAML: %w", err)
	}

	// Parse the merged YAML to get the components structure
	var stack struct {
		Spec struct {
			Components map[string]map[string]interface{} `yaml:"components"`
		} `yaml:"spec"`
	}

	if err := yaml.Unmarshal(mergedYAML, &stack); err != nil {
		return nil, fmt.Errorf("failed to parse merged YAML: %w", err)
	}

	// Extract component types and names
	var components []Component

	// Iterate through component types (e.g., terraform)
	for typeName, typeComponents := range stack.Spec.Components {
		// Iterate through component names (e.g., vpc)
		for componentName := range typeComponents {
			components = append(components, Component{
				Type: typeName,
				Name: componentName,
			})
		}
	}

	return components, nil
}

// extractComponentVars extracts variables from a specific component in a stack
func extractComponentVars(filePath, componentType, componentName string) ([]ComponentVar, error) {
	// Get catalogDir from config
	catalogDir := viper.GetString("catalogDir")
	if catalogDir == "" {
		// Default to fixtures/catalog if not specified
		catalogDir = "fixtures/catalog"
	}

	// Use our YAML parser that can handle anchors
	mergedYAML, err := yamlparser.MergeYAML(filePath, catalogDir)
	if err != nil {
		return nil, fmt.Errorf("failed to merge YAML: %w", err)
	}

	// Parse the merged YAML to get the component structure
	var stack map[string]interface{}
	if err := yaml.Unmarshal(mergedYAML, &stack); err != nil {
		return nil, fmt.Errorf("failed to parse merged YAML: %w", err)
	}

	// Navigate to the component vars
	spec, ok := stack["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("spec section not found in YAML")
	}

	components, ok := spec["components"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("components section not found in YAML")
	}

	typeComponents, ok := components[componentType].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("component type '%s' not found", componentType)
	}

	component, ok := typeComponents[componentName].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("component '%s' not found", componentName)
	}

	varsMap, ok := component["vars"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("vars section not found for component '%s'", componentName)
	}

	// Convert vars map to slice of ComponentVar
	var vars []ComponentVar
	for name, value := range varsMap {
		vars = append(vars, ComponentVar{
			Name:  name,
			Value: value,
		})
	}

	// Sort vars by name for consistent output
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Name < vars[j].Name
	})

	return vars, nil
}

// outputComponentsJSON prints components as JSON
func outputComponentsJSON(components []Component) {
	jsonData, err := json.MarshalIndent(components, "", "  ")
	if err != nil {
		logger.Log.Fatalf("Error marshaling to JSON: %v", err)
	}

	fmt.Println(string(jsonData))
}

// outputComponentVarsJSON prints component variables as JSON
func outputComponentVarsJSON(vars []ComponentVar) {
	jsonData, err := json.MarshalIndent(vars, "", "  ")
	if err != nil {
		logger.Log.Fatalf("Error marshaling to JSON: %v", err)
	}

	fmt.Println(string(jsonData))
}

// printComponentsBubblesTable prints components using the Bubbles table component
func printComponentsBubblesTable(components []Component) {
	// Print the title
	title := titleStyle.Render("COMPONENTS")
	fmt.Println(title)

	// Define table columns
	columns := []table.Column{
		{Title: "TYPE", Width: 30},
		{Title: "NAME", Width: 50},
	}

	// Prepare rows data
	rows := []table.Row{}
	for _, component := range components {
		rows = append(rows, table.Row{component.Type, component.Name})
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

// printComponentsStandardTable prints components as a plain text table
func printComponentsStandardTable(components []Component) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Println("Stack Components")
	fmt.Println()
	fmt.Fprintln(w, "Type\tName")
	fmt.Fprintln(w, "----\t----")

	for _, component := range components {
		fmt.Fprintf(w, "%s\t%s\n", component.Type, component.Name)
	}

	w.Flush()
}

// printComponentVarsBubblesTable prints component variables using the Bubbles table component
func printComponentVarsBubblesTable(vars []ComponentVar, component *Component) {
	// Print the title
	title := titleStyle.Render(fmt.Sprintf("COMPONENT: %s/%s", component.Type, component.Name))
	fmt.Println(title)

	// Define table columns
	columns := []table.Column{
		{Title: "VARIABLE", Width: 40},
		{Title: "VALUE", Width: 50},
	}

	// Prepare rows data
	rows := []table.Row{}
	for _, v := range vars {
		// Convert value to string representation with styling for complex types
		valueStr := formatValueWithStyle(v.Value)
		rows = append(rows, table.Row{v.Name, valueStr})
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

// formatValueWithStyle formats values with color for the styled table
func formatValueWithStyle(value interface{}) string {
	str := formatValue(value)

	// For colored output, we can add specific styling for different types
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		// Use a subtle color for complex structures
		return lipgloss.NewStyle().Foreground(lipgloss.Color("105")).Render(str)
	case reflect.Bool:
		// Highlight booleans
		if v.Bool() {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("76")).Render(str) // Green for true
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(str) // Red for false
	case reflect.String:
		// Style strings
		return lipgloss.NewStyle().Foreground(lipgloss.Color("149")).Render(str)
	default:
		return str
	}
}

// formatValue formats a value for display, handling maps and slices nicely
func formatValue(value interface{}) string {
	if value == nil {
		return "null"
	}

	// Check the type using reflection
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map:
		// Format maps as key-value pairs
		if v.Len() == 0 {
			return "{}"
		}

		var pairs []string
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			val := formatValue(iter.Value().Interface()) // Recursively format values
			pairs = append(pairs, fmt.Sprintf("%s: %s", key, val))
		}
		// Sort for consistent output
		sort.Strings(pairs)
		return "{ " + strings.Join(pairs, ", ") + " }"

	case reflect.Slice, reflect.Array:
		// Format slices as comma-separated values
		if v.Len() == 0 {
			return "[]"
		}

		var items []string
		for i := 0; i < v.Len(); i++ {
			item := formatValue(v.Index(i).Interface()) // Recursively format values
			items = append(items, item)
		}
		return "[ " + strings.Join(items, ", ") + " ]"

	case reflect.String:
		// Return strings without quotes for nicer display
		return fmt.Sprintf("%s", value)

	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// Format simple values directly
		return fmt.Sprintf("%v", value)

	default:
		// For complex types, use JSON representation
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value) // Fallback to default formatting
		}
		return string(jsonBytes)
	}
}

// printComponentVarsStandardTable prints component variables as a plain text table
func printComponentVarsStandardTable(vars []ComponentVar, component *Component) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Printf("Component: %s/%s\n", component.Type, component.Name)
	fmt.Println()
	fmt.Fprintln(w, "Variable\tValue")
	fmt.Fprintln(w, "--------\t-----")

	for _, v := range vars {
		// Convert value to string representation
		valueStr := formatValue(v.Value)
		fmt.Fprintf(w, "%s\t%s\n", v.Name, valueStr)
	}

	w.Flush()
}

// outputTerraformVars prints component variables in Terraform .tfvars format
func outputTerraformVars(vars []ComponentVar, stackFile string, componentName string) {
	// Add a header comment with stack and component info
	fmt.Printf("# Terraform variables for component '%s' from stack '%s'\n", componentName, stackFile)
	fmt.Printf("# Generated by skunk\n\n")

	// Print each variable in terraform format
	for _, v := range vars {
		fmt.Printf("%s = %s\n", v.Name, formatTerraformValue(v.Value))
	}
}

// formatTerraformValue formats a value for Terraform .tfvars format
func formatTerraformValue(value interface{}) string {
	if value == nil {
		return "null"
	}

	// Check the type using reflection
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map:
		// Format maps as Terraform objects
		if v.Len() == 0 {
			return "{}"
		}

		var pairs []string
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			val := formatTerraformValue(iter.Value().Interface()) // Recursively format values
			pairs = append(pairs, fmt.Sprintf("%s = %s", key, val))
		}
		// Sort for consistent output
		sort.Strings(pairs)
		return "{\n  " + strings.Join(pairs, "\n  ") + "\n}"

	case reflect.Slice, reflect.Array:
		// Format slices as Terraform lists
		if v.Len() == 0 {
			return "[]"
		}

		var items []string
		for i := 0; i < v.Len(); i++ {
			item := formatTerraformValue(v.Index(i).Interface()) // Recursively format values
			items = append(items, item)
		}
		return "[" + strings.Join(items, ", ") + "]"

	case reflect.String:
		// Strings must be quoted in Terraform
		return fmt.Sprintf("\"%s\"", value)

	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// Format simple values directly
		return fmt.Sprintf("%v", value)

	default:
		// For complex types, use JSON representation
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value) // Fallback to default formatting
		}
		return string(jsonBytes)
	}
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.AddCommand(showStackCmd)

	// Add flags
	showStackCmd.Flags().StringVarP(&stackName, "stackName", "s", "", "stack name (required if --filter not specified)")
	showStackCmd.Flags().StringVarP(&componentName, "component", "c", "", "component name")
	showStackCmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON instead of a table")
	showStackCmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
	showStackCmd.Flags().BoolVar(&tfVars, "tfvars", false, "output component variables in Terraform format (only valid with --component)")
	showStackCmd.Flags().StringArray("filter", []string{}, "filter stacks by label (format: key=value or key!=value), can be specified multiple times")
}
