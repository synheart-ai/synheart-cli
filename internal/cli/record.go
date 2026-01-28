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
	Long: `Generate and record HSI records or raw wearable sensor signals in vendor-specific formats (Whoop/Garmin) to an NDJSON file.`,
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
	registry := scenario.NewRegistry()
	if err := registry.LoadFromDir(getScenarioDir()); err != nil {
		return fmt.Errorf("failed to load scenarios: %w", err)
	}

	scen, err := registry.Get(recordScenario)
	if err != nil {
		return fmt.Errorf("failed to load scenario '%s': %w", recordScenario, err)
	}
	scen.Duration = recordDuration

	scenarioEngine := scenario.NewEngine(scen)

	tickRate, err := parseTickRate(recordRate)
	if err != nil {
		return fmt.Errorf("invalid rate: %w", err)
	}

	genConfig := generator.Config{
		Seed:        recordSeed,
		DefaultRate: tickRate,
		SourceType:  "wearable",
		SourceID:    "mock-watch-01",
		Vendor:      recordVendor,
	}
	gen := generator.NewGenerator(scenarioEngine, genConfig)

	var fluxEngine *flux.Engine
	if recordFlux {
		wasmPath := filepath.Join("bin", "synheart_flux.wasm")
		if _, err := os.Stat(wasmPath); err != nil {
			return fmt.Errorf("flux wasm not found (run 'make build' first): %w", err)
		}
		fluxEngine, err = flux.NewEngine(context.Background(), wasmPath)
		if err != nil {
			return fmt.Errorf("failed to initialize flux engine: %w", err)
		}
		defer fluxEngine.Close(context.Background())
	}

	rec, err := recorder.NewRecorder(recordOut)
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}
	defer rec.Close()

	// Channels
	vendorPayloads := make(chan []byte, 100)
	finalRecords := make(chan []byte, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	fmt.Printf("📼 Recording Session Started\n\n")
	fmt.Printf("Scenario:   %s\n", scen.Name)
	fmt.Printf("Output:     %s\n", recordOut)
	fmt.Printf("Vendor:     %s\n", recordVendor)
	fmt.Printf("Flux:       %v\n\n", recordFlux)

	eventCount := 0
	progressCallback := func() {
		eventCount++
		if eventCount%1000 == 0 {
			fmt.Printf("\rRecorded ~%d sensor events...", eventCount)
		}
	}

	// Recording thread
	go func() {
		if err := rec.RecordFromChannel(ctx, finalRecords, progressCallback); err != nil && err != context.Canceled {
			log.Printf("Recording error: %v", err)
		}
	}()

	// Transformation Pipeline
	go func() {
		defer close(finalRecords)
		for {
			select {
			case <-ctx.Done():
				return
			case payload, ok := <-vendorPayloads:
				if !ok { return }
				if recordFlux {
					var hsi string
					var err error
					if recordVendor == "garmin" {
						hsi, err = fluxEngine.GarminToHSI(ctx, string(payload), "UTC", "mock-watch-01")
					} else {
						hsi, err = fluxEngine.WhoopToHSI(ctx, string(payload), "UTC", "mock-watch-01")
					}
					if err == nil {
						finalRecords <- []byte(hsi)
					}
				} else {
					finalRecords <- payload
				}
			}
		}
	}()

	// Generation
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()
	if err := gen.Generate(ctx, ticker, nil, vendorPayloads); err != nil && err != context.Canceled {
		return fmt.Errorf("generator error: %w", err)
	}

	close(vendorPayloads)
	time.Sleep(100 * time.Millisecond) // Let recording finish

	fmt.Printf("\n\n✅ Recording complete: %s\n", recordOut)
	return nil
}
