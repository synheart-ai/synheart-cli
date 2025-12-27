package scenario

import (
	"sync"
	"time"
)

// Engine executes a scenario and tracks progression through phases
type Engine struct {
	scenario  *Scenario
	startTime time.Time
	mu        sync.RWMutex
}

// NewEngine creates a new scenario engine
func NewEngine(scenario *Scenario) *Engine {
	return &Engine{
		scenario:  scenario,
		startTime: time.Now(),
	}
}

// GetElapsed returns the time elapsed since scenario start
func (e *Engine) GetElapsed() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return time.Since(e.startTime)
}

// GetCurrentPhase returns the current phase based on elapsed time
func (e *Engine) GetCurrentPhase() *Phase {
	elapsed := e.GetElapsed()
	return e.scenario.getCurrentPhase(elapsed)
}

// GetSignalConfig returns the effective signal configuration at current time
func (e *Engine) GetSignalConfig(signalName string) *SignalConfig {
	elapsed := e.GetElapsed()
	return e.scenario.GetEffectiveConfig(signalName, elapsed)
}

// IsComplete returns true if the scenario has finished
func (e *Engine) IsComplete() bool {
	duration, unlimited := ParseDuration(e.scenario.Duration)
	if unlimited {
		return false
	}
	return e.GetElapsed() >= duration
}

// GetScenario returns the underlying scenario
func (e *Engine) GetScenario() *Scenario {
	return e.scenario
}

// Reset resets the scenario to the beginning
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.startTime = time.Now()
}
