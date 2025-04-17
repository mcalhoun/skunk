package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/goccy/go-yaml"
	"github.com/mcalhoun/skunk/internal/logger"
	stackfinder "github.com/mcalhoun/skunk/internal/stack-finder"
	tablerender "github.com/mcalhoun/skunk/internal/table-render"
	"github.com/mcalhoun/skunk/internal/utils"
	yamlparser "github.com/mcalhoun/skunk/internal/yaml-parser"
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

// Global finder for use in production code
var defaultStackFinder StackFinder = NewDefaultStackFinder()

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
		runShowStackCmd(cmd, args, defaultStackFinder)
	},
}

// runShowStackCmd is the implementation of the show stack command logic
// extracted to a separate function to make it testable with a mock stack finder
func runShowStackCmd(cmd *cobra.Command, args []string, finder StackFinder) {
	// Get filters from flags
	filters, err := cmd.Flags().GetStringArray("filter")
	if err != nil {
		logger.Log.Fatalf("Error getting filters: %v", err)
	}

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
	stacks, err := finder.FindStacks(stacksPath)
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
		// Update stackName with the selected stack's name
		stackName = targetStack.Name
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
		return
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
			return
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
			printComponentVarsStandardTable(stackName, vars, foundComponent)
			return
		}

		// Otherwise, print pretty table output
		printComponentVarsBubblesTable(targetStack.Name, vars, foundComponent)
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
		printComponentsStandardTable(stackName, components)
		return
	}

	// Otherwise, print pretty table output with the bubbles table component
	printComponentsBubblesTable(stackName, components)
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

// printComponentsBubblesTable prints components using the tablerender package
func printComponentsBubblesTable(stackName string, components []Component) {
	// Convert components to rows
	rows := make([][]string, 0, len(components))
	for _, component := range components {
		rows = append(rows, []string{component.Type, component.Name})
	}

	// Setup table style
	style := tablerender.DefaultTableStyle()
	style.Title = fmt.Sprintf("STACK: %s\nCOMPONENTS", stackName)

	// Render and print the table
	table := tablerender.RenderTable([]string{"TYPE", "NAME"}, rows, style)
	fmt.Println(table)
}

// printComponentsStandardTable prints a plain table of components
func printComponentsStandardTable(stackName string, components []Component) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "STACK: %s\n", stackName)
	fmt.Fprintf(w, "COMPONENT TYPE\tCOMPONENT NAME\n")
	for _, c := range components {
		fmt.Fprintf(w, "%s\t%s\n", c.Type, c.Name)
	}
	w.Flush()
}

// printComponentVarsBubblesTable prints component variables using the tablerender package
func printComponentVarsBubblesTable(stackName string, vars []ComponentVar, component *Component) {
	// Create a title based on component
	title := fmt.Sprintf("STACK: %s\nCOMPONENT: %s/%s", stackName, component.Type, component.Name)

	// Convert vars to a map for the table renderer
	data := make(map[string]interface{}, len(vars))
	for _, v := range vars {
		data[v.Name] = v.Value
	}

	// Convert map to rows with colored values
	colorScheme := tablerender.DefaultColorScheme()
	rows := tablerender.FormatKeyValueDataWithColor(data, colorScheme)

	// Sort rows by name for consistent output
	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})

	// Setup table style
	style := tablerender.DefaultTableStyle()
	style.Title = title

	// Render and print the table
	table := tablerender.RenderTable([]string{"VARIABLE", "VALUE"}, rows, style)
	fmt.Println(table)
}

// printComponentVarsStandardTable prints component variables as a plain text table
func printComponentVarsStandardTable(stackName string, vars []ComponentVar, component *Component) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Printf("Stack: %s\n", stackName)
	fmt.Printf("Component: %s/%s\n", component.Type, component.Name)
	fmt.Println()
	fmt.Fprintln(w, "Variable\tValue")
	fmt.Fprintln(w, "--------\t-----")

	for _, v := range vars {
		// Convert value to string representation
		valueStr := formatVariableValue(v.Value)
		fmt.Fprintf(w, "%s\t%s\n", v.Name, valueStr)
	}

	w.Flush()
}

// formatVariableValue formats a value as a string for plain text output
func formatVariableValue(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string:
		return fmt.Sprintf("%v", v)
	default:
		// For complex types, use JSON representation
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value) // Fallback to default formatting
		}
		return string(jsonBytes)
	}
}

// outputTerraformVars prints component variables in Terraform .tfvars format
func outputTerraformVars(vars []ComponentVar, stackFile string, componentName string) {
	// Add a header comment with stack and component info
	fmt.Printf("# Terraform variables for component '%s' from stack '%s'\n", componentName, stackFile)
	fmt.Printf("# Generated by skunk\n\n")

	// Use the tablerender package to format values
	for _, v := range vars {
		// Convert to appropriate Terraform syntax
		valueStr := ""
		switch v.Value.(type) {
		case string:
			valueStr = fmt.Sprintf("\"%v\"", v.Value)
		case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			valueStr = fmt.Sprintf("%v", v.Value)
		default:
			// Use JSON marshaling for complex types
			jsonData, err := json.Marshal(v.Value)
			if err != nil {
				valueStr = fmt.Sprintf("%v", v.Value)
			} else {
				valueStr = string(jsonData)
			}
		}
		fmt.Printf("%s = %s\n", v.Name, valueStr)
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
	showStackCmd.Flags().StringArray("filter", []string{}, "filter stacks by label (format: key=value or key!=value), by name prefix (format: name=pattern or name!=pattern), by regex (format: name~=regex or name!~=regex), or directly by name using wildcard pattern '*' or regex '/pattern/'")
}
