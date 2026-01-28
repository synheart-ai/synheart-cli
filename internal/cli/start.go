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
	"github.com/synheart/synheart-cli/internal/models"
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
	startFluxVerbose bool
	startVendor   string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start generating and broadcasting HSI events",
	Long: `Starts generating raw sensor events, transforms them into HSI using the Flux engine, and broadcasts HSI records over network protocols.`,
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
	startCmd.Flags().BoolVar(&startFluxVerbose, "flux-verbose", false, "Log raw vendor data before Flux transformation")
	startCmd.Flags().StringVar(&startVendor, "vendor", "whoop", "Vendor data format for Flux: whoop|garmin")
}

func runStart(cmd *cobra.Command, args []string) error {
	// Load scenarios
	registry := scenario.NewRegistry()
	scenariosDir := getScenarioDir()
	if err := registry.LoadFromDir(scenariosDir); err != nil {
		return fmt.Errorf("failed to load scenarios: %w", err)
	}

	// Get scenario
	scen, err := registry.Get(startScenario)
	if err != nil {
		return fmt.Errorf("failed to load scenario '%s': %w", startScenario, err)
	}

	// Override duration if specified
	if startDuration != "" {
		scen.Duration = startDuration
	}

	// Create scenario engine
	engine := scenario.NewEngine(scen)

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
	}
	gen := generator.NewGenerator(engine, genConfig)

	// Create event channel (raw sensor data)
	events := make(chan models.Event, 100)

	// Create WebSocket server
	wsServer := transport.NewWebSocketServer(startHost, startPort)

	// Create Server-Sent Events server
	sse := transport.NewSSEServer(startHost, startPort+1)

	// Create UDP server
	udp := transport.NewUDPServer(startHost, startPort+2)

	// Setup Flux Engine (Primary HSI Engine)
	wasmPath := filepath.Join("bin", "synheart_flux.wasm")
	if _, err := os.Stat(wasmPath); err != nil {
		return fmt.Errorf("flux wasm not found (run 'make build' first): %w", err)
	}

	fluxEngine, err := flux.NewEngine(context.Background(), wasmPath)
	if err != nil {
		return fmt.Errorf("failed to initialize flux engine: %w", err)
	}
	defer fluxEngine.Close(context.Background())
	aggregator := flux.NewAggregator()
	fmt.Printf("✨ Flux Engine initialized (Wasm: %s)\n", wasmPath)

	// Create HSI record channel (Dispatcher source)
	hsiRecords := make(chan []byte, 10)
	dispatcher := transport.NewDispatcher(hsiRecords, 100)

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

	// Start WebSocket server
	go func() {
		if err := wsServer.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	// Start Server-Sent Events server
	go func() {
		if err := sse.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("Server-Sent Events server error: %v", err)
		}
	}()

	// Start UDP server
	go func() {
		if err := udp.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("UDP server error: %v", err)
		}
	}()

	// Give servers time to start
	time.Sleep(200 * time.Millisecond)

	fmt.Printf("🚀 Synheart Mock Server Started\n\n")
	fmt.Printf("Scenario:     %s\n", scen.Name)
	fmt.Printf("Description:  %s\n", scen.Description)
	fmt.Printf("WebSocket:    %s\n", wsServer.GetAddress())
	fmt.Printf("SSE:          %s\n", sse.GetAddress())
	fmt.Printf("UDP:          %s\n", udp.GetAddress())
	fmt.Printf("Vendor:       %s\n", startVendor)
	fmt.Printf("Seed:         %d\n", startSeed)
	fmt.Printf("Run ID:       %s\n\n", gen.GetRunID())

	// dispatch HSI records to network servers
	wsEvents := dispatcher.Subscribe()
	go func() {
		if err := wsServer.BroadcastFromChannel(ctx, wsEvents); err != nil && err != context.Canceled {
			log.Printf("Broadcast error: %v", err)
		}
	}()

	sseEvents := dispatcher.Subscribe()
	go func() {
		if err := sse.BroadcastFromChannel(ctx, sseEvents); err != nil && err != context.Canceled {
			log.Printf("Broadcast error: %v", err)
		}
	}()

	udpEvents := dispatcher.Subscribe()
	go func() {
		if err := udp.BroadcastFromChannel(ctx, udpEvents); err != nil && err != context.Canceled {
			log.Printf("Broadcast error: %v", err)
		}
	}()

	var rec *recorder.Recorder
	if startOut != "" {
		rec, err = recorder.NewRecorder(startOut)
		if err != nil {
			return fmt.Errorf("failed to create recorder: %w", err)
		}
		defer rec.Close()

		recEvents := dispatcher.Subscribe()
		go func() {
			if err := rec.RecordFromChannel(ctx, recEvents, nil); err != nil && err != context.Canceled {
				log.Printf("Recording error: %v", err)
			}
		}()

		fmt.Printf("Recording:    %s\n\n", startOut)
	}

	go dispatcher.Run(ctx)

	// Internal processing loop: Sensors -> Aggregator -> Flux -> Dispatcher
	go func() {
		defer close(hsiRecords)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				aggregator.Add(event)

				// Process every 20 events (approx 1s at 20Hz effective)
				if aggregator.Count() >= 20 {
					var payload string
					var err error
					var hsi string

					switch startVendor {
					case "garmin":
						payload, err = aggregator.ToGarminJSON()
						if err == nil {
							hsi, err = fluxEngine.GarminToHSI(ctx, payload, "UTC", "mock-watch-01")
						}
					default: // whoop
						payload, err = aggregator.ToWhoopJSON()
						if err == nil {
							hsi, err = fluxEngine.WhoopToHSI(ctx, payload, "UTC", "mock-watch-01")
						}
					}

					if err != nil {
						log.Printf("Flux transformation error: %v", err)
					} else {
						if startFluxVerbose {
							ui.Printf("\n%s\n", ui.bold(fmt.Sprintf("--- Raw %s JSON ---", strings.ToUpper(startVendor))))
							ui.Printf("%s\n\n", payload)
						}

						// Send to all transports
						hsiRecords <- []byte(hsi)
					}
					aggregator.Clear()
				}
			}
		}
	}()

	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("\nGenerating events...")

	// Start generating
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	if err := gen.Generate(ctx, ticker, events); err != nil && err != context.Canceled {
		return fmt.Errorf("generator error: %w", err)
	}

	close(events)

	fmt.Println("\nShutdown complete")
	return nil
}

func parseTickRate(rate string) (time.Duration, error) {
	var hz float64
	_, err := fmt.Sscanf(rate, "%fhz", &hz)
	if err != nil {
		return 0, err
	}
	if hz <= 0 {
		return 0, fmt.Errorf("rate must be positive")
	}
	return time.Duration(float64(time.Second) / hz), nil
}

func getScenarioDir() string {
	// Try current directory first
	if _, err := os.Stat("scenarios"); err == nil {
		return "scenarios"
	}

	// Try relative to executable
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "scenarios")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	// Default to scenarios in current directory
	return "scenarios"
}
