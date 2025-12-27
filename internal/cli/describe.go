package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/synheart/synheart-cli/internal/scenario"
)

var describeCmd = &cobra.Command{
	Use:   "describe <scenario>",
	Short: "Describe a scenario in detail",
	Long:  `Shows detailed information about a scenario including signals, phases, and typical ranges.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDescribe,
}

func runDescribe(cmd *cobra.Command, args []string) error {
	scenarioName := args[0]

	// Load scenarios
	registry := scenario.NewRegistry()
	if err := registry.LoadFromDir(getScenarioDir()); err != nil {
		return fmt.Errorf("failed to load scenarios: %w", err)
	}

	// Get scenario
	scen, err := registry.Get(scenarioName)
	if err != nil {
		return fmt.Errorf("scenario not found: %w", err)
	}

	// Print details
	fmt.Printf("Scenario: %s\n", scen.Name)
	fmt.Printf("Description: %s\n", scen.Description)
	fmt.Printf("Duration: %s\n", scen.Duration)
	fmt.Printf("Default Rate: %s\n\n", scen.DefaultRate)

	// Print signals
	fmt.Println("Signals:")
	for name, config := range scen.Signals {
		fmt.Printf("  %s\n", name)
		if config.Baseline != nil {
			fmt.Printf("    Baseline: %v\n", config.Baseline)
		}
		if config.Noise != nil {
			fmt.Printf("    Noise: %v\n", config.Noise)
		}
		if config.Rate != "" {
			fmt.Printf("    Rate: %s\n", config.Rate)
		}
		if config.Unit != "" {
			fmt.Printf("    Unit: %s\n", config.Unit)
		}
	}

	// Print phases
	if len(scen.Phases) > 0 {
		fmt.Println("\nPhases:")
		for i, phase := range scen.Phases {
			fmt.Printf("  %d. %s (duration: %s)\n", i+1, phase.Name, phase.Duration)
			if len(phase.Overrides) > 0 {
				fmt.Println("     Overrides:")
				for signal, override := range phase.Overrides {
					fmt.Printf("       %s:", signal)
					if override.Add != 0 {
						fmt.Printf(" add=%.1f", override.Add)
					}
					if override.Multiply != 0 {
						fmt.Printf(" multiply=%.1f", override.Multiply)
					}
					if override.Value != "" {
						fmt.Printf(" value=%s", override.Value)
					}
					if override.Baseline != nil {
						fmt.Printf(" baseline=%v", override.Baseline)
					}
					if override.Noise != nil {
						fmt.Printf(" noise=%v", override.Noise)
					}
					fmt.Println()
				}
			}
		}
	}

	fmt.Println()
	return nil
}
