package scenario

import "time"

// Scenario defines a complete scenario with phases and signal configurations
type Scenario struct {
	Name        string                    `yaml:"name"`
	Description string                    `yaml:"description"`
	Duration    string                    `yaml:"duration"` // e.g., "8m", "unlimited"
	DefaultRate string                    `yaml:"default_rate"`
	Signals     map[string]*SignalConfig  `yaml:"signals"`
	Phases      []Phase                   `yaml:"phases"`
}

// Phase represents a time-bounded stage of a scenario with specific overrides
type Phase struct {
	Name      string                   `yaml:"name"`
	Duration  string                   `yaml:"duration"`
	Overrides map[string]*SignalConfig `yaml:"overrides,omitempty"`
}

// SignalConfig defines the configuration for a signal
type SignalConfig struct {
	Baseline interface{} `yaml:"baseline,omitempty"` // Can be number or array
	Noise    interface{} `yaml:"noise,omitempty"`    // Can be number or array
	Rate     string      `yaml:"rate,omitempty"`     // e.g., "1hz", "50hz"
	Unit     string      `yaml:"unit,omitempty"`     // e.g., "bpm", "ms"

	// Override modifiers
	Add              float64 `yaml:"add,omitempty"`
	Multiply         float64 `yaml:"multiply,omitempty"`
	Value            string  `yaml:"value,omitempty"` // For discrete values like "on"/"off"
	Ramp             string  `yaml:"ramp,omitempty"`  // Ramp duration
	RampToBaseline   string  `yaml:"ramp_to_baseline,omitempty"`
}

// ParseDuration parses duration strings like "8m", "30s", "unlimited"
func ParseDuration(s string) (time.Duration, bool) {
	if s == "unlimited" || s == "" {
		return 0, true // 0 means unlimited
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, false
	}
	return d, false
}

// GetEffectiveConfig returns the signal config for a given signal name at a specific time
func (s *Scenario) GetEffectiveConfig(signalName string, elapsed time.Duration) *SignalConfig {
	// Start with base signal config
	baseConfig := s.Signals[signalName]
	if baseConfig == nil {
		return nil
	}

	// Find current phase
	currentPhase := s.getCurrentPhase(elapsed)
	if currentPhase == nil {
		return baseConfig
	}

	// Apply phase overrides if they exist
	if override, ok := currentPhase.Overrides[signalName]; ok {
		// Merge override with base config
		merged := *baseConfig
		if override.Add != 0 {
			merged.Add = override.Add
		}
		if override.Multiply != 0 {
			merged.Multiply = override.Multiply
		}
		if override.Value != "" {
			merged.Value = override.Value
		}
		if override.Ramp != "" {
			merged.Ramp = override.Ramp
		}
		if override.RampToBaseline != "" {
			merged.RampToBaseline = override.RampToBaseline
		}
		if override.Baseline != nil {
			merged.Baseline = override.Baseline
		}
		if override.Noise != nil {
			merged.Noise = override.Noise
		}
		return &merged
	}

	return baseConfig
}

func (s *Scenario) getCurrentPhase(elapsed time.Duration) *Phase {
	if len(s.Phases) == 0 {
		return nil
	}

	var currentTime time.Duration
	for i := range s.Phases {
		phaseDuration, unlimited := ParseDuration(s.Phases[i].Duration)
		if unlimited {
			return &s.Phases[i]
		}

		if elapsed < currentTime+phaseDuration {
			return &s.Phases[i]
		}
		currentTime += phaseDuration
	}

	// Return last phase if we've exceeded total duration
	return &s.Phases[len(s.Phases)-1]
}
