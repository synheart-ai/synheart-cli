package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
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
)

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record mock data to a file",
	Long: `Generate and record HSI events to an NDJSON file.

Examples:
  synheart mock record --scenario workout --duration 15m --out workout.ndjson
  synheart mock record --scenario stress_spike --seed 42 --out test.ndjson`,
	RunE: runRecord,
}

func init() {
	recordCmd.Flags().StringVar(&recordScenario, "scenario", "baseline", "Scenario to run")
	recordCmd.Flags().StringVar(&recordDuration, "duration", "5m", "Duration to record")
	recordCmd.Flags().StringVar(&recordOut, "out", "", "Output file (required)")
	recordCmd.Flags().Int64Var(&recordSeed, "seed", time.Now().UnixNano(), "Random seed")
	recordCmd.Flags().StringVar(&recordRate, "rate", "50hz", "Global tick rate")
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

	// Create recorder
	rec, err := recorder.NewRecorder(recordOut)
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}
	defer rec.Close()

	// Create event channel
	events := make(chan models.Event, 100)

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
	fmt.Printf("Run ID:     %s\n\n", gen.GetRunID())

	// Start recording
	recordingDone := make(chan error, 1)
	go func() {
		recordingDone <- rec.RecordFromChannel(ctx, events)
	}()

	fmt.Println("Press Ctrl+C to stop early")
	fmt.Println("\nRecording events...")

	// Start progress display goroutine
	progressTicker := time.NewTicker(500 * time.Millisecond)
	defer progressTicker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-progressTicker.C:
				count := rec.GetCount()
				if count > 0 && count%1000 == 0 {
					fmt.Printf("\rRecorded %d events...", count)
				}
			}
		}
	}()

	// Start generating
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	if err := gen.Generate(ctx, ticker, events); err != nil && err != context.Canceled {
		return fmt.Errorf("generator error: %w", err)
	}

	close(events)

	// Wait for recording to finish
	select {
	case err := <-recordingDone:
		if err != nil && err != context.Canceled {
			return fmt.Errorf("recording error: %w", err)
		}
	case <-time.After(5 * time.Second):
		return fmt.Errorf("recording did not complete in time")
	}

	eventCount := rec.GetCount()
	fmt.Printf("\n\n✅ Recording complete: %d events saved to %s\n", eventCount, recordOut)
	return nil
}
