package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/synheart/synheart-cli/internal/flux"
	"github.com/synheart/synheart-cli/internal/generator"
	"github.com/synheart/synheart-cli/internal/recorder"
	"github.com/synheart/synheart-cli/internal/scenario"
	"github.com/synheart/synheart-cli/internal/transport"
)

var (
	startHost     string
	startPort     int
	startScenario string
	startDuration string
	startRate     string
	startSeed     int64
	startOut      string
	startFlux     bool
	startFluxVerbose bool
	startVendor   string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start generating and broadcasting sensor data",
	Long: `Starts generating raw sensor events, aggregates them into vendor-specific payloads, and optionally transforms them into HSI using the Flux engine.`,
	RunE: runStart,
}

func init() {
	startCmd.Flags().StringVar(&startHost, "host", "127.0.0.1", "Host to bind to")
	startCmd.Flags().IntVar(&startPort, "port", 8787, "Port to listen on")
	startCmd.Flags().StringVar(&startScenario, "scenario", "baseline", "Scenario to run")
	startCmd.Flags().StringVar(&startDuration, "duration", "", "Duration to run (e.g., 5m, 1h)")
	startCmd.Flags().StringVar(&startRate, "rate", "50hz", "Global tick rate")
	startCmd.Flags().Int64Var(&startSeed, "seed", time.Now().UnixNano(), "Random seed for deterministic output")
	startCmd.Flags().StringVar(&startOut, "out", "", "Record events to file")
	startCmd.Flags().BoolVar(&startFlux, "flux", false, "Enable Synheart Flux Wasm transformation (defaults to raw vendor JSON)")
	startCmd.Flags().BoolVar(&startFluxVerbose, "flux-verbose", false, "Log raw vendor data before Flux transformation")
	startCmd.Flags().StringVar(&startVendor, "vendor", "whoop", "Vendor data format: whoop|garmin")
}

func runStart(cmd *cobra.Command, args []string) error {
	// Load scenarios
	registry := scenario.NewRegistry()
	if err := registry.LoadFromDir(getScenarioDir()); err != nil {
		return fmt.Errorf("failed to load scenarios: %w", err)
	}

	// Get scenario
	scen, err := registry.Get(startScenario)
	if err != nil {
		return fmt.Errorf("failed to load scenario '%s': %w", startScenario, err)
	}

	if startDuration != "" {
		scen.Duration = startDuration
	}

	// Create scenario engine
	scenarioEngine := scenario.NewEngine(scen)

	// Parse rate
	tickRate, err := parseTickRate(startRate)
	if err != nil {
		return fmt.Errorf("invalid rate: %w", err)
	}

	// Create generator
	genConfig := generator.Config{
		Seed:        startSeed,
		DefaultRate: tickRate,
		SourceType:  "wearable",
		SourceID:    "mock-watch-01",
		Vendor:      startVendor,
	}
	gen := generator.NewGenerator(scenarioEngine, genConfig)

	// Setup Flux Engine (Optional HSI Engine)
	var fluxEngine *flux.Engine
	if startFlux {
		wasmPath := filepath.Join("bin", "synheart_flux.wasm")
		if _, err := os.Stat(wasmPath); err != nil {
			return fmt.Errorf("flux wasm not found (run 'make build' first): %w", err)
		}

		fluxEngine, err = flux.NewEngine(context.Background(), wasmPath)
		if err != nil {
			return fmt.Errorf("failed to initialize flux engine: %w", err)
		}
		defer fluxEngine.Close(context.Background())
		fmt.Printf("✨ Flux Engine initialized (Wasm: %s)\n", wasmPath)
	}

	// Create channels
	vendorPayloads := make(chan []byte, 100)
	broadcastRecords := make(chan []byte, 100)

	// Create dispatcher for final output
	dispatcher := transport.NewDispatcher(broadcastRecords, 100)

	// Create network servers
	wsServer := transport.NewWebSocketServer(startHost, startPort)
	sse := transport.NewSSEServer(startHost, startPort+1)
	udp := transport.NewUDPServer(startHost, startPort+2)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("\nReceived interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Start servers
	go func() { if err := wsServer.Start(ctx); err != nil && err != context.Canceled { log.Printf("WS error: %v", err) } }()
	go func() { if err := sse.Start(ctx); err != nil && err != context.Canceled { log.Printf("SSE error: %v", err) } }()
	go func() { if err := udp.Start(ctx); err != nil && err != context.Canceled { log.Printf("UDP error: %v", err) } }()

	time.Sleep(200 * time.Millisecond)

	fmt.Printf("🚀 Synheart Mock Server Started\n\n")
	fmt.Printf("Scenario:     %s\n", scen.Name)
	fmt.Printf("WebSocket:    %s\n", wsServer.GetAddress())
	fmt.Printf("SSE:          %s\n", sse.GetAddress())
	fmt.Printf("UDP:          %s\n", udp.GetAddress())
	fmt.Printf("Vendor:       %s\n", startVendor)
	fmt.Printf("Flux Enabled: %v\n\n", startFlux)

	// Wire up transport broadcasting
	go func() { wsServer.BroadcastFromChannel(ctx, dispatcher.Subscribe()) }()
	go func() { sse.BroadcastFromChannel(ctx, dispatcher.Subscribe()) }()
	go func() { udp.BroadcastFromChannel(ctx, dispatcher.Subscribe()) }()

	if startOut != "" {
		if rec, err := recorder.NewRecorder(startOut); err == nil {
			defer rec.Close()
			go rec.RecordFromChannel(ctx, dispatcher.Subscribe(), nil)
			fmt.Printf("Recording:    %s\n\n", startOut)
		}
	}

	go dispatcher.Run(ctx)

	// Transformation Pipeline: Generator -> Vendor Payloads -> (Flux) -> Final Records
	go func() {
		defer close(broadcastRecords)
		for {
			select {
			case <-ctx.Done():
				return
			case payload, ok := <-vendorPayloads:
				if !ok { return }

				if startFluxVerbose {
					ui := NewUI(os.Stdout, os.Stderr, false, false, false)
					ui.Printf("\n%s\n", ui.bold(fmt.Sprintf("--- Raw %s JSON ---", strings.ToUpper(startVendor))))
					ui.Printf("%s\n\n", string(payload))
				}

				if startFlux {
					var hsi string
					var err error
					if startVendor == "garmin" {
						hsi, err = fluxEngine.GarminToHSI(ctx, string(payload), "UTC", "mock-watch-01")
					} else {
						hsi, err = fluxEngine.WhoopToHSI(ctx, string(payload), "UTC", "mock-watch-01")
					}
					if err == nil {
						broadcastRecords <- []byte(hsi)
					} else {
						log.Printf("Flux error: %v", err)
					}
				} else {
					broadcastRecords <- payload
				}
			}
		}
	}()

	// Start Generating
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()
	if err := gen.Generate(ctx, ticker, nil, vendorPayloads); err != nil && err != context.Canceled {
		return fmt.Errorf("generator error: %w", err)
	}

	close(vendorPayloads)
	
	fmt.Println("\nShutdown complete")
	return nil
}
