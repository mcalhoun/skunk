package logger

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// Log is the global logger instance
var Log = log.NewWithOptions(os.Stderr, log.Options{
	ReportTimestamp: true,
	TimeFormat:      "15:04:05",
	Level:           log.InfoLevel,
	Prefix:          "",
	ReportCaller:    false,
})

// Colors
var (
	errorColor = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	warnColor  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	infoColor  = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	debugColor = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// Valid log levels
const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
	FatalLevel = "fatal"
	PanicLevel = "panic"
)

// SetLevel sets the logging level based on a string
func SetLevel(level string) {
	switch level {
	case DebugLevel:
		Log.SetLevel(log.DebugLevel)
	case InfoLevel:
		Log.SetLevel(log.InfoLevel)
	case WarnLevel:
		Log.SetLevel(log.WarnLevel)
	case ErrorLevel:
		Log.SetLevel(log.ErrorLevel)
	case FatalLevel:
		Log.SetLevel(log.FatalLevel)
	case PanicLevel:
		// Charm Bracelet logger doesn't have a panic level, use fatal instead
		Log.SetLevel(log.FatalLevel)
	default:
		// Default to info
		Log.SetLevel(log.InfoLevel)
	}
}

// Init initializes the logger with the specified settings and prints a blank line
// to separate logger output from previous content
func Init() {
	// Default settings are already configured in the Log variable declaration
	// Print a blank line to separate logger output
	fmt.Fprintln(os.Stderr)
}

// ErrorStyled logs an error message with custom styling
func ErrorStyled(msg string) {
	Log.Error(errorColor.Render(msg))
}

// WarnStyled logs a warning message with custom styling
func WarnStyled(msg string) {
	Log.Warn(warnColor.Render(msg))
}

// InfoStyled logs an info message with custom styling
func InfoStyled(msg string) {
	Log.Info(infoColor.Render(msg))
}

// DebugStyled logs a debug message with custom styling
func DebugStyled(msg string) {
	Log.Debug(debugColor.Render(msg))
}
