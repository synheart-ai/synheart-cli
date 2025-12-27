package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "synheart",
	Short: "Synheart CLI - Mock HSI data generator for local development",
	Long: `Synheart CLI generates HSI-compatible sensor data streams
that mimic phone + wearable sources for local SDK development.

It eliminates dependency on physical devices during development,
providing repeatable scenarios for QA and demos.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(mockCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(versionCmd)
}
