package main

import (
	"fmt"
	"log"
	"path/filepath"

	stackfinder "github.com/mcalhoun/skunk/internal/stack-finder"
)

func main() {
	// Example 1: Using FindStacks with a glob pattern
	stacksGlob := "fixtures/stacks/plat-prod-east-1.yaml"
	stacks, err := stackfinder.FindStacks(stacksGlob)
	if err != nil {
		log.Fatalf("Error finding stacks with glob pattern: %v", err)
	}

	fmt.Println("Found stacks using glob pattern:")
	printStacks(stacks)

	// Example 2: Using FindStacksRecursive to search the directory recursively
	stacksDir := "fixtures/stacks"
	recursiveStacks, err := stackfinder.FindStacksRecursive(stacksDir)
	if err != nil {
		log.Fatalf("Error finding stacks recursively: %v", err)
	}

	fmt.Println("\nFound stacks recursively:")
	printStacks(recursiveStacks)

	// Example 3: Getting stacks with specific labels
	fmt.Println("\nStacks in prod environment:")
	for _, stack := range recursiveStacks {
		if stack.Labels["environment"] == "prod" {
			fmt.Printf("- %s (in %s)\n", stack.Name, filepath.Base(stack.FilePath))
		}
	}
}

func printStacks(stacks []stackfinder.StackMetadata) {
	if len(stacks) == 0 {
		fmt.Println("No stacks found.")
		return
	}

	for i, stack := range stacks {
		fmt.Printf("%d. Stack: %s\n", i+1, stack.Name)
		fmt.Printf("   File: %s\n", stack.FilePath)

		if len(stack.Labels) > 0 {
			fmt.Println("   Labels:")
			for k, v := range stack.Labels {
				fmt.Printf("     %s: %s\n", k, v)
			}
		} else {
			fmt.Println("   Labels: none")
		}
		fmt.Println()
	}
}
