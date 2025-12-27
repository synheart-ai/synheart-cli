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
	"github.com/synheart/synheart-cli/internal/models"
	"github.com/synheart/synheart-cli/internal/recorder"
	"github.com/synheart/synheart-cli/internal/transport"
)

var (
	replayIn    string
	replaySpeed float64
	replayLoop  bool
	replayHost  string
	replayPort  int
)

var replayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Replay recorded events",
	Long: `Replay events from a previously recorded NDJSON file.

Examples:
  synheart mock replay --in workout.ndjson
  synheart mock replay --in test.ndjson --speed 2.0 --loop`,
	RunE: runReplay,
}

func init() {
	replayCmd.Flags().StringVar(&replayIn, "in", "", "Input file to replay (required)")
	replayCmd.Flags().Float64Var(&replaySpeed, "speed", 1.0, "Playback speed multiplier")
	replayCmd.Flags().BoolVar(&replayLoop, "loop", false, "Loop playback continuously")
	replayCmd.Flags().StringVar(&replayHost, "host", "127.0.0.1", "Host to bind to")
	replayCmd.Flags().IntVar(&replayPort, "port", 8787, "Port to listen on")
	replayCmd.MarkFlagRequired("in")
}

func runReplay(cmd *cobra.Command, args []string) error {
	// Create replayer
	rep := recorder.NewReplayer(replayIn, replaySpeed, replayLoop)

	// Get info about the recording
	count, err := rep.CountEvents()
	if err != nil {
		return fmt.Errorf("failed to read recording: %w", err)
	}

	firstEvent, err := rep.GetFirstEvent()
	if err != nil {
		return fmt.Errorf("failed to read first event: %w", err)
	}

	// Create event channel
	events := make(chan models.Event, 100)

	// Create WebSocket server
	wsServer := transport.NewWebSocketServer(replayHost, replayPort)

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

	// Start WebSocket server
	go func() {
		if err := wsServer.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	fmt.Printf("▶️  Replay Session Started\n\n")
	fmt.Printf("File:         %s\n", replayIn)
	fmt.Printf("Events:       %d\n", count)
	fmt.Printf("Scenario:     %s\n", firstEvent.Session.Scenario)
	fmt.Printf("Speed:        %.1fx\n", replaySpeed)
	fmt.Printf("Loop:         %v\n", replayLoop)
	fmt.Printf("WebSocket:    %s\n\n", wsServer.GetAddress())

	// Start broadcasting
	go func() {
		if err := wsServer.BroadcastFromChannel(ctx, events); err != nil && err != context.Canceled {
			log.Printf("Broadcast error: %v", err)
		}
	}()

	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("\nReplaying events...")

	// Start replay
	if err := rep.Replay(ctx, events); err != nil && err != context.Canceled {
		return fmt.Errorf("replay error: %w", err)
	}

	close(events)

	fmt.Println("\nReplay complete")
	return nil
}
