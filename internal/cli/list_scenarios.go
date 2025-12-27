package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/synheart/synheart-cli/internal/scenario"
)

var listScenariosCmd = &cobra.Command{
	Use:   "list-scenarios",
	Short: "List available scenarios",
	Long:  `Lists all built-in scenarios with their descriptions.`,
	RunE:  runListScenarios,
}

func runListScenarios(cmd *cobra.Command, args []string) error {
	// Load scenarios
	registry := scenario.NewRegistry()
	if err := registry.LoadFromDir(getScenarioDir()); err != nil {
		return fmt.Errorf("failed to load scenarios: %w", err)
	}

	scenarios := registry.ListWithDescriptions()
	if len(scenarios) == 0 {
		fmt.Println("No scenarios found")
		return nil
	}

	// Sort by name
	names := make([]string, 0, len(scenarios))
	for name := range scenarios {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Println("Available scenarios:")
	fmt.Println()
	for _, name := range names {
		fmt.Printf("  %-20s %s\n", name, scenarios[name])
	}
	fmt.Println()

	return nil
}
