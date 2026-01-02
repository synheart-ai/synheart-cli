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
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start generating and broadcasting HSI events",
	Long: `Starts generating HSI-compatible events and broadcasting them over WebSocket.

Examples:
  synheart mock start
  synheart mock start --scenario stress_spike --seed 42
  synheart mock start --port 9000 --duration 5m`,
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

	// Create event channel
	events := make(chan models.Event, 100)

	// Create WebSocket server
	wsServer := transport.NewWebSocketServer(startHost, startPort)

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

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	fmt.Printf("🚀 Synheart Mock Server Started\n\n")
	fmt.Printf("Scenario:     %s\n", scen.Name)
	fmt.Printf("Description:  %s\n", scen.Description)
	fmt.Printf("WebSocket:    %s\n", wsServer.GetAddress())
	fmt.Printf("Seed:         %d\n", startSeed)
	fmt.Printf("Run ID:       %s\n\n", gen.GetRunID())

	// dispatch events to both websocket and recorder
	dispatcher := transport.NewDispatcher(events, 100)

	wsEvents := dispatcher.Subscribe()
	go func() {
		if err := wsServer.BroadcastFromChannel(ctx, wsEvents); err != nil && err != context.Canceled {
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
			if err := rec.RecordFromChannel(ctx, recEvents); err != nil && err != context.Canceled {
				log.Printf("Recording error: %v", err)
			}
		}()

		fmt.Printf("Recording:    %s\n\n", startOut)
	}

	go dispatcher.Run(ctx)

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
