# Stack Finder

A Go package for finding Stack YAML files and extracting metadata from them.

## Features

- Find Stack YAML files using glob patterns
- Recursively search directories for Stack files
- Extract metadata such as name and labels
- Handle YAML files with unresolved anchors
- Robust parsing with fallback to regex-based extraction

## Usage

```go
package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/mcalhoun/skunk/internal/stack-finder"
)

func main() {
	// Example 1: Using FindStacks with a glob pattern
	stacksGlob := "fixtures/stacks/*.yaml"
	stacks, err := stackfinder.FindStacks(stacksGlob)
	if err != nil {
		log.Fatalf("Error finding stacks with glob pattern: %v", err)
	}

	fmt.Println("Found stacks using glob pattern:")
	for _, stack := range stacks {
		fmt.Printf("Stack: %s, File: %s\n", stack.Name, stack.FilePath)
		fmt.Println("Labels:", stack.Labels)
	}

	// Example 2: Using FindStacksRecursive to search the directory recursively
	stacksDir := "fixtures/stacks"
	recursiveStacks, err := stackfinder.FindStacksRecursive(stacksDir)
	if err != nil {
		log.Fatalf("Error finding stacks recursively: %v", err)
	}

	fmt.Println("\nFound stacks recursively:")
	for _, stack := range recursiveStacks {
		fmt.Printf("Stack: %s, File: %s\n", stack.Name, stack.FilePath)
		fmt.Println("Labels:", stack.Labels)
	}

	// Example 3: Getting stacks with specific labels
	fmt.Println("\nStacks in prod env:")
	for _, stack := range recursiveStacks {
		if stack.Labels["environment"] == "prod" {
			fmt.Printf("- %s (in %s)\n", stack.Name, filepath.Base(stack.FilePath))
		}
	}
}
```

## API Reference

### Types

#### `type StackMetadata`

```go
type StackMetadata struct {
	Name     string            // metadata.name from the Stack
	Labels   map[string]string // metadata.labels from the Stack
	FilePath string            // path to the Stack file
}
```

### Functions

#### `func FindStacks(globPattern string) ([]StackMetadata, error)`

Finds all YAML files matching the glob pattern, parses them, and returns metadata for those that are of kind: Stack.

#### `func FindStacksRecursive(root string) ([]StackMetadata, error)`

Finds all Stack files in a directory and its subdirectories.
