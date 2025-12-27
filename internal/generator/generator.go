package generator

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/synheart/synheart-cli/internal/models"
	"github.com/synheart/synheart-cli/internal/scenario"
)

// Generator orchestrates signal generation based on scenario
type Generator struct {
	engine      *scenario.Engine
	rng         *rand.Rand
	runID       string
	source      models.Source
	sequence    int64
	signals     map[string]SignalGenerator
	signalRates map[string]time.Duration
	lastEmit    map[string]time.Time
}

// Config holds generator configuration
type Config struct {
	Seed         int64
	DefaultRate  time.Duration
	SourceType   string
	SourceID     string
	SourceSide   *string
}

// NewGenerator creates a new event generator
func NewGenerator(engine *scenario.Engine, config Config) *Generator {
	source := rand.NewSource(config.Seed)
	rng := rand.New(source)

	side := config.SourceSide
	if side == nil && config.SourceType == "wearable" {
		left := "left"
		side = &left
	}

	return &Generator{
		engine:      engine,
		rng:         rng,
		runID:       uuid.New().String(),
		source: models.Source{
			Type: config.SourceType,
			ID:   config.SourceID,
			Side: side,
		},
		sequence:    0,
		signals:     GetAllSignals(),
		signalRates: make(map[string]time.Duration),
		lastEmit:    make(map[string]time.Time),
	}
}

// Generate produces events at the specified tick rate
func (g *Generator) Generate(ctx context.Context, ticker *time.Ticker, output chan<- models.Event) error {
	now := time.Now()
	for signalName := range g.signals {
		g.lastEmit[signalName] = now
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if g.engine.IsComplete() {
				return nil
			}

			events := g.generateTick()
			for _, event := range events {
				select {
				case output <- event:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}

// generateTick generates all events for the current tick
func (g *Generator) generateTick() []models.Event {
	elapsed := g.engine.GetElapsed()
	now := time.Now()
	events := make([]models.Event, 0)

	// Build correlation context
	ctx := NewCorrelationContext()

	// Generate all signals first
	for signalName, generator := range g.signals {
		config := g.engine.GetSignalConfig(signalName)
		if config == nil {
			continue
		}

		// Check if it's time to emit this signal
		signalRate := g.getSignalRate(config)
		if now.Sub(g.lastEmit[signalName]) < signalRate {
			continue
		}

		value := generator(g.rng, config, elapsed.Seconds())
		ctx.Set(signalName, value)
		g.lastEmit[signalName] = now
	}

	// Apply correlations
	ctx.ApplyCorrelations()

	// Create events from correlated values
	for signalName := range g.signals {
		value, ok := ctx.Get(signalName)
		if !ok {
			continue
		}

		config := g.engine.GetSignalConfig(signalName)
		if config == nil {
			continue
		}

		event := g.createEvent(signalName, value, config)
		events = append(events, event)
	}

	return events
}

// createEvent creates a single event
func (g *Generator) createEvent(signalName string, value interface{}, config *scenario.SignalConfig) models.Event {
	g.sequence++

	signal := models.Signal{
		Name:    signalName,
		Value:   value,
		Quality: 0.9 + g.rng.Float64()*0.1, // 0.9-1.0 quality
	}

	// Set unit from config or use default
	if config.Unit != "" {
		signal.Unit = config.Unit
	} else {
		signal.Unit = getDefaultUnit(signalName)
	}

	session := models.Session{
		RunID:    g.runID,
		Scenario: g.engine.GetScenario().Name,
		Seed:     g.rng.Int63(),
	}

	return models.NewEvent(
		uuid.New().String(),
		g.source,
		session,
		signal,
		g.sequence,
	)
}

// getSignalRate returns the rate for a signal (how often to emit)
func (g *Generator) getSignalRate(config *scenario.SignalConfig) time.Duration {
	if config.Rate != "" {
		if duration, err := parseRate(config.Rate); err == nil {
			return duration
		}
	}

	// Default rates based on signal type
	return time.Second // 1Hz default
}

// parseRate converts rate string like "50hz" to duration
func parseRate(rate string) (time.Duration, error) {
	var hz float64
	_, err := fmt.Sscanf(rate, "%fhz", &hz)
	if err != nil {
		return 0, err
	}
	if hz <= 0 {
		return 0, fmt.Errorf("invalid rate: %s", rate)
	}
	return time.Duration(float64(time.Second) / hz), nil
}

// getDefaultUnit returns the default unit for a signal
func getDefaultUnit(signalName string) string {
	units := map[string]string{
		"ppg.hr_bpm":       "bpm",
		"ppg.hrv_rmssd_ms": "ms",
		"accel.xyz_mps2":   "m/s²",
		"gyro.xyz_rps":     "rad/s",
		"temp.skin_c":      "°C",
		"eda.us":           "μS",
		"screen.state":     "",
		"app.activity":     "",
		"motion.activity":  "",
	}
	return units[signalName]
}

// GetRunID returns the current run ID
func (g *Generator) GetRunID() string {
	return g.runID
}
