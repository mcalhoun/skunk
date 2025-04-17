package cmd

import (
	stackfinder "github.com/mcalhoun/skunk/internal/stack-finder"
)

// StackFinder interface allows for easily mocking stack finder functionality in tests
type StackFinder interface {
	FindStacks(pattern string) ([]stackfinder.StackMetadata, error)
}

// DefaultStackFinder is the default implementation that uses the stackfinder package
type DefaultStackFinder struct{}

// FindStacks implements the StackFinder interface using the actual stackfinder package
func (f *DefaultStackFinder) FindStacks(pattern string) ([]stackfinder.StackMetadata, error) {
	return stackfinder.FindStacks(pattern)
}

// Creates a new default stack finder
func NewDefaultStackFinder() StackFinder {
	return &DefaultStackFinder{}
}
