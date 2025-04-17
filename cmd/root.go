package cmd

import (
	"os"

	"github.com/mcalhoun/skunk/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// PersistentPreRun runs before each command and configures logging
func configureLogging(cmd *cobra.Command, args []string) {
	// Configure logging based on precedence:
	// 1. Command-line flag (highest)
	// 2. Config value (middle)
	// 3. Default (lowest)

	logLevelToUse := logger.InfoLevel // Default

	// Check config file
	configLogLevel := viper.GetString("logLevel")
	if configLogLevel != "" {
		logLevelToUse = configLogLevel
	}

	// Command line overrides config
	if logLevel != "" {
		logLevelToUse = logLevel
	}

	// Set log level
	logger.SetLevel(logLevelToUse)

	// Output debug info about the chosen log level
	logger.Log.Debug("Log level configured: " + logLevelToUse)
}

var (
	cfgFile  string
	logLevel string
	rootCmd  = &cobra.Command{
		Use:   "skunk",
		Short: "Skunk is a YAML stack management tool",
		Long: `Skunk helps you manage YAML stacks with anchors and references.
It provides functionality to find stacks, merge YAML with anchors,
and manage your infrastructure configuration.`,
		PersistentPreRun: configureLogging,
	}
)

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Log.Errorf("%v", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Initialize logger
	logger.Init()

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./skunk.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level (debug, info, warn, error)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory with name "skunk.yaml" (without extension).
		viper.AddConfigPath(".")
		viper.SetConfigName("skunk")
		viper.SetConfigType("yaml")
	}

	// Set default values
	viper.SetDefault("stacksPath", "fixtures/stacks/*.yaml")
	viper.SetDefault("catalogDir", "fixtures/catalog")
	viper.SetDefault("logLevel", "info")
	viper.SetDefault("maxTableWidth", 80)

	// Read environment variables
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// This message should only appear at debug level
		logger.Log.Debug("Using config file: " + viper.ConfigFileUsed())
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			logger.Log.Error("Error reading config file: " + err.Error())
		}
	}
}
