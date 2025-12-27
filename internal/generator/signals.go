package generator

import (
	"math"
	"math/rand"

	"github.com/synheart/synheart-cli/internal/scenario"
)

// SignalGenerator generates a specific signal value
type SignalGenerator func(rng *rand.Rand, config *scenario.SignalConfig, elapsed float64) interface{}

// GetAllSignals returns all available signal generators
func GetAllSignals() map[string]SignalGenerator {
	return map[string]SignalGenerator{
		"ppg.hr_bpm":          generateHeartRate,
		"ppg.hrv_rmssd_ms":    generateHRV,
		"accel.xyz_mps2":      generateAccel,
		"gyro.xyz_rps":        generateGyro,
		"temp.skin_c":         generateSkinTemp,
		"eda.us":              generateEDA,
		"screen.state":        generateScreenState,
		"app.activity":        generateAppActivity,
		"motion.activity":     generateMotionActivity,
	}
}

// generateHeartRate generates heart rate in BPM
func generateHeartRate(rng *rand.Rand, config *scenario.SignalConfig, elapsed float64) interface{} {
	baseline := getFloat(config.Baseline, 72.0)
	noise := getFloat(config.Noise, 3.0)

	// Apply modifiers
	value := baseline
	if config.Add != 0 {
		value += config.Add
	}
	if config.Multiply != 0 {
		value *= config.Multiply
	}

	// Add random noise
	value += rng.NormFloat64() * noise

	// Clamp to realistic range
	return clamp(value, 40, 200)
}

// generateHRV generates heart rate variability in milliseconds
func generateHRV(rng *rand.Rand, config *scenario.SignalConfig, elapsed float64) interface{} {
	baseline := getFloat(config.Baseline, 50.0)
	noise := getFloat(config.Noise, 8.0)

	value := baseline
	if config.Multiply != 0 {
		value *= config.Multiply
	}

	value += rng.NormFloat64() * noise

	return clamp(value, 10, 150)
}

// generateAccel generates 3D acceleration vector in m/s²
func generateAccel(rng *rand.Rand, config *scenario.SignalConfig, elapsed float64) interface{} {
	baseline := getVector3(config.Baseline, []float64{0, 0, 9.81})
	noise := getFloat(config.Noise, 0.05)

	return []float64{
		baseline[0] + rng.NormFloat64()*noise,
		baseline[1] + rng.NormFloat64()*noise,
		baseline[2] + rng.NormFloat64()*noise,
	}
}

// generateGyro generates 3D gyroscope vector in rad/s
func generateGyro(rng *rand.Rand, config *scenario.SignalConfig, elapsed float64) interface{} {
	baseline := getVector3(config.Baseline, []float64{0, 0, 0})
	noise := getFloat(config.Noise, 0.02)

	return []float64{
		baseline[0] + rng.NormFloat64()*noise,
		baseline[1] + rng.NormFloat64()*noise,
		baseline[2] + rng.NormFloat64()*noise,
	}
}

// generateSkinTemp generates skin temperature in Celsius
func generateSkinTemp(rng *rand.Rand, config *scenario.SignalConfig, elapsed float64) interface{} {
	baseline := getFloat(config.Baseline, 33.0)
	noise := getFloat(config.Noise, 0.1)

	// Slow sinusoidal drift
	drift := math.Sin(elapsed/600.0) * 0.3

	value := baseline + drift + rng.NormFloat64()*noise

	return clamp(value, 30, 37)
}

// generateEDA generates electrodermal activity in microsiemens
func generateEDA(rng *rand.Rand, config *scenario.SignalConfig, elapsed float64) interface{} {
	baseline := getFloat(config.Baseline, 2.0)
	noise := getFloat(config.Noise, 0.2)

	value := baseline
	if config.Add != 0 {
		value += config.Add
	}

	value += rng.NormFloat64() * noise

	return clamp(value, 0.1, 20)
}

// generateScreenState generates screen on/off state
func generateScreenState(rng *rand.Rand, config *scenario.SignalConfig, elapsed float64) interface{} {
	if config.Value != "" {
		return config.Value
	}

	// Random on/off with some persistence
	if rng.Float64() > 0.95 {
		return "off"
	}
	return "on"
}

// generateAppActivity generates app foreground/background state
func generateAppActivity(rng *rand.Rand, config *scenario.SignalConfig, elapsed float64) interface{} {
	if config.Value != "" {
		return config.Value
	}

	activities := []string{"foreground", "background", "typing", "scrolling"}
	return activities[rng.Intn(len(activities))]
}

// generateMotionActivity generates motion activity state
func generateMotionActivity(rng *rand.Rand, config *scenario.SignalConfig, elapsed float64) interface{} {
	if config.Value != "" {
		return config.Value
	}

	activities := []string{"still", "walk", "run"}
	weights := []float64{0.7, 0.25, 0.05}

	r := rng.Float64()
	cumulative := 0.0
	for i, weight := range weights {
		cumulative += weight
		if r < cumulative {
			return activities[i]
		}
	}
	return "still"
}

// Helper functions

func getFloat(val interface{}, defaultVal float64) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return defaultVal
	}
}

func getVector3(val interface{}, defaultVal []float64) []float64 {
	switch v := val.(type) {
	case []interface{}:
		if len(v) >= 3 {
			return []float64{
				getFloat(v[0], defaultVal[0]),
				getFloat(v[1], defaultVal[1]),
				getFloat(v[2], defaultVal[2]),
			}
		}
	case []float64:
		if len(v) >= 3 {
			return v
		}
	}
	return defaultVal
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
