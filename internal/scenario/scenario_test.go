package scenario

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		unlimited bool
	}{
		{"unlimited", 0, true},
		{"", 0, true},
		{"5m", 5 * time.Minute, false},
		{"30s", 30 * time.Second, false},
		{"1h", time.Hour, false},
	}

	for _, test := range tests {
		duration, unlimited := ParseDuration(test.input)
		if unlimited != test.unlimited {
			t.Errorf("ParseDuration(%s): expected unlimited=%v, got %v", test.input, test.unlimited, unlimited)
		}
		if !unlimited && duration != test.expected {
			t.Errorf("ParseDuration(%s): expected %v, got %v", test.input, test.expected, duration)
		}
	}
}

func TestGetEffectiveConfig(t *testing.T) {
	scenario := &Scenario{
		Name: "test",
		Signals: map[string]*SignalConfig{
			"ppg.hr_bpm": {
				Baseline: 72.0,
				Noise:    3.0,
			},
		},
		Phases: []Phase{
			{
				Name:     "baseline",
				Duration: "2m",
			},
			{
				Name:     "spike",
				Duration: "30s",
				Overrides: map[string]*SignalConfig{
					"ppg.hr_bpm": {
						Add: 35.0,
					},
				},
			},
		},
	}

	// Test baseline phase (first 2 minutes)
	config := scenario.GetEffectiveConfig("ppg.hr_bpm", 1*time.Minute)
	if config.Add != 0 {
		t.Errorf("Expected no override in baseline phase, got add=%v", config.Add)
	}

	// Test spike phase (after 2 minutes)
	config = scenario.GetEffectiveConfig("ppg.hr_bpm", 2*time.Minute+15*time.Second)
	if config.Add != 35.0 {
		t.Errorf("Expected add=35.0 in spike phase, got %v", config.Add)
	}
}

func TestScenarioEngine(t *testing.T) {
	scenario := &Scenario{
		Name:     "test",
		Duration: "5m",
		Signals: map[string]*SignalConfig{
			"ppg.hr_bpm": {
				Baseline: 72.0,
			},
		},
		Phases: []Phase{
			{
				Name:     "phase1",
				Duration: "2m",
			},
		},
	}

	engine := NewEngine(scenario)

	// Check initial state
	if engine.IsComplete() {
		t.Error("Engine should not be complete immediately after creation")
	}

	// Get signal config
	config := engine.GetSignalConfig("ppg.hr_bpm")
	if config == nil {
		t.Error("Expected signal config, got nil")
	}

	if config.Baseline != 72.0 {
		t.Errorf("Expected baseline 72.0, got %v", config.Baseline)
	}
}
