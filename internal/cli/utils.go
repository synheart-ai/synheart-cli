package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func getScenarioDir() string {
	// Try current directory first
	if _, err := os.Stat("scenarios"); err == nil {
		return "scenarios"
	}

	// Try relative to executable
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "scenarios")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	// Default to scenarios in current directory
	return "scenarios"
}

func parseTickRate(rate string) (time.Duration, error) {
	var hz float64
	_, err := fmt.Sscanf(rate, "%fhz", &hz)
	if err != nil {
		return 0, err
	}
	if hz <= 0 {
		return 0, fmt.Errorf("rate must be positive")
	}
	return time.Duration(float64(time.Second) / hz), nil
}
