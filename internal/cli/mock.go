package cli

import (
	"github.com/spf13/cobra"
)

var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "Mock data generation commands",
	Long:  `Commands for generating, recording, and replaying mock HSI sensor data.`,
}

func init() {
	mockCmd.AddCommand(startCmd)
	mockCmd.AddCommand(recordCmd)
	mockCmd.AddCommand(replayCmd)
	mockCmd.AddCommand(listScenariosCmd)
	mockCmd.AddCommand(describeCmd)
}
