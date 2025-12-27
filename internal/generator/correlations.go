package generator

import (
	"math"
)

// CorrelationContext holds generated signal values for correlation
type CorrelationContext struct {
	values map[string]interface{}
}

// NewCorrelationContext creates a new correlation context
func NewCorrelationContext() *CorrelationContext {
	return &CorrelationContext{
		values: make(map[string]interface{}),
	}
}

// Set stores a signal value
func (c *CorrelationContext) Set(name string, value interface{}) {
	c.values[name] = value
}

// Get retrieves a signal value
func (c *CorrelationContext) Get(name string) (interface{}, bool) {
	val, ok := c.values[name]
	return val, ok
}

// ApplyCorrelations applies correlation rules between signals
func (c *CorrelationContext) ApplyCorrelations() {
	// HR ↔ Accel: Higher acceleration should correlate with higher HR
	if accel, ok := c.Get("accel.xyz_mps2"); ok {
		if hr, ok := c.Get("ppg.hr_bpm"); ok {
			accelVec := accel.([]float64)
			magnitude := math.Sqrt(accelVec[0]*accelVec[0] + accelVec[1]*accelVec[1] + accelVec[2]*accelVec[2])

			// If high acceleration (>11 m/s²), nudge HR up slightly
			if magnitude > 11.0 {
				hrVal := hr.(float64)
				hrVal += (magnitude - 11.0) * 2.0 // Small correlation factor
				c.Set("ppg.hr_bpm", clamp(hrVal, 40, 200))
			}
		}
	}

	// HRV ↔ EDA: Higher stress (EDA) should reduce HRV
	if eda, ok := c.Get("eda.us"); ok {
		if hrv, ok := c.Get("ppg.hrv_rmssd_ms"); ok {
			edaVal := eda.(float64)
			hrvVal := hrv.(float64)

			// If EDA is elevated (>4.0), reduce HRV
			if edaVal > 4.0 {
				factor := 1.0 - (edaVal-4.0)*0.05
				if factor < 0.6 {
					factor = 0.6
				}
				c.Set("ppg.hrv_rmssd_ms", clamp(hrvVal*factor, 10, 150))
			}
		}
	}

	// Motion activity ↔ Accel: Ensure consistency
	if motion, ok := c.Get("motion.activity"); ok {
		if accel, ok := c.Get("accel.xyz_mps2"); ok {
			motionStr := motion.(string)
			accelVec := accel.([]float64)
			magnitude := math.Sqrt(accelVec[0]*accelVec[0] + accelVec[1]*accelVec[1] + accelVec[2]*accelVec[2])

			// Adjust magnitude based on motion state for consistency
			switch motionStr {
			case "still":
				if magnitude > 10.5 {
					// Reduce to near-gravity
					factor := 9.85 / magnitude
					c.Set("accel.xyz_mps2", []float64{
						accelVec[0] * factor,
						accelVec[1] * factor,
						accelVec[2] * factor,
					})
				}
			case "walk":
				if magnitude < 10.0 || magnitude > 15.0 {
					// Keep in walking range
					target := 11.0 + math.Abs(accelVec[0])*0.5
					factor := target / magnitude
					c.Set("accel.xyz_mps2", []float64{
						accelVec[0] * factor,
						accelVec[1] * factor,
						accelVec[2] * factor,
					})
				}
			case "run":
				if magnitude < 12.0 {
					// Boost to running range
					factor := 13.0 / magnitude
					c.Set("accel.xyz_mps2", []float64{
						accelVec[0] * factor,
						accelVec[1] * factor,
						accelVec[2] * factor,
					})
				}
			}
		}
	}
}
