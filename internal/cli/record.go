package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/synheart/synheart-cli/internal/flux"
	"github.com/synheart/synheart-cli/internal/generator"
	"github.com/synheart/synheart-cli/internal/models"
	"github.com/synheart/synheart-cli/internal/recorder"
	"github.com/synheart/synheart-cli/internal/scenario"
)

var (
	recordScenario string
	recordDuration string
	recordOut      string
	recordSeed     int64
	recordRate     string
	recordVendor   string
	recordFlux     bool
)

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record mock data to a file",
	Long: `Generate and record HSI records or raw wearable sensor signals to an NDJSON file.`,
	RunE: runRecord,
}

func init() {
	recordCmd.Flags().StringVar(&recordScenario, "scenario", "baseline", "Scenario to run")
	recordCmd.Flags().StringVar(&recordDuration, "duration", "5m", "Duration to record")
	recordCmd.Flags().StringVar(&recordOut, "out", "", "Output file (required)")
	recordCmd.Flags().Int64Var(&recordSeed, "seed", time.Now().UnixNano(), "Random seed")
	recordCmd.Flags().StringVar(&recordRate, "rate", "50hz", "Global tick rate")
	recordCmd.Flags().StringVar(&recordVendor, "vendor", "whoop", "Vendor data format: whoop|garmin")
	recordCmd.Flags().BoolVar(&recordFlux, "flux", false, "Enable Synheart Flux Wasm transformation (defaults to raw vendor JSON)")
	recordCmd.MarkFlagRequired("out")
}

func runRecord(cmd *cobra.Command, args []string) error {
	// Load scenarios
	registry := scenario.NewRegistry()
	if err := registry.LoadFromDir(getScenarioDir()); err != nil {
		return fmt.Errorf("failed to load scenarios: %w", err)
	}

	// Get scenario
	scen, err := registry.Get(recordScenario)
	if err != nil {
		return fmt.Errorf("failed to load scenario '%s': %w", recordScenario, err)
	}

	// Override duration
	scen.Duration = recordDuration

	// Create scenario engine
	engine := scenario.NewEngine(scen)

	// Parse rate
	tickRate, err := parseTickRate(recordRate)
	if err != nil {
		return fmt.Errorf("invalid rate: %w", err)
	}

	// Create generator
	genConfig := generator.Config{
		Seed:        recordSeed,
		DefaultRate: tickRate,
		SourceType:  "wearable",
		SourceID:    "mock-watch-01",
	}
	gen := generator.NewGenerator(engine, genConfig)

	// Setup Flux Engine (Optional HSI Engine)
	var fluxEngine *flux.Engine
	if recordFlux {
		wasmPath := filepath.Join("bin", "synheart_flux.wasm")
		if _, err := os.Stat(wasmPath); err != nil {
			return fmt.Errorf("flux wasm not found (run 'make build' first): %w", err)
		}

		var err error
		fluxEngine, err = flux.NewEngine(context.Background(), wasmPath)
		if err != nil {
			return fmt.Errorf("failed to initialize flux engine: %w", err)
		}
		defer fluxEngine.Close(context.Background())
	}

	aggregator := flux.NewAggregator()

	// Create recorder
	rec, err := recorder.NewRecorder(recordOut)
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}
	defer rec.Close()

	// Create channels
	events := make(chan models.Event, 100)
	records := make(chan []byte, 10)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	fmt.Printf("📼 Recording Session Started\n\n")
	fmt.Printf("Scenario:   %s\n", scen.Name)
	fmt.Printf("Duration:   %s\n", recordDuration)
	fmt.Printf("Output:     %s\n", recordOut)
	fmt.Printf("Seed:       %d\n", recordSeed)
	fmt.Printf("Vendor:     %s\n", recordVendor)
	fmt.Printf("Flux:       %v\n", recordFlux)
	fmt.Printf("Run ID:     %s\n\n", gen.GetRunID())

	eventCount := 0
	progressCallback := func() {
		eventCount++
		if eventCount%1000 == 0 {
			fmt.Printf("\rRecorded %d entries...", eventCount)
		}
	}
	// Start recording
	go func() {
		if err := rec.RecordFromChannel(ctx, records, progressCallback); err != nil && err != context.Canceled {
			log.Printf("Recording error: %v", err)
		}
	}()

	// Start transformation loop
	go func() {
		defer close(records)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				aggregator.Add(event)
				if aggregator.Count() >= 20 {
					var payload string
					var err error

					if recordVendor == "garmin" {
						payload, err = aggregator.ToGarminJSON()
					} else {
						payload, err = aggregator.ToWhoopJSON()
					}

					if err == nil {
						if recordFlux {
							var hsi string
							if recordVendor == "garmin" {
								hsi, err = fluxEngine.GarminToHSI(ctx, payload, "UTC", "mock-watch-01")
							} else {
								hsi, err = fluxEngine.WhoopToHSI(ctx, payload, "UTC", "mock-watch-01")
							}
							if err == nil {
								records <- []byte(hsi)
							}
						} else {
							records <- []byte(payload)
						}
					}
					aggregator.Clear()
				}
			}
		}
	}()

	fmt.Println("Press Ctrl+C to stop early")
	fmt.Println("\nRecording events...")

	// Start generating
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	if err := gen.Generate(ctx, ticker, events); err != nil && err != context.Canceled {
		return fmt.Errorf("generator error: %w", err)
	}

	close(events)
	time.Sleep(100 * time.Millisecond) // Let recording finish

	fmt.Printf("\n\n✅ Recording complete: %s\n", recordOut)
	return nil
}
