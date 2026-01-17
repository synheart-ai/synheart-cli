package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/synheart/synheart-cli/internal/receiver"
)

var (
	receiverHost   string
	receiverPort   int
	receiverToken  string
	receiverOut    string
	receiverFormat string
	receiverGzip   bool
)

var receiverCmd = &cobra.Command{
	Use:   "receiver",
	Short: "Start a local HTTP server to receive HSI exports",
	Long: `Starts a blocking HTTP server that listens for incoming HSI export payloads
from Synheart Life app over a local network.

The server validates incoming payloads against the HSI Export Schema v1,
handles idempotency, and outputs received data to stdout or files.

Examples:
  synheart receiver
  synheart receiver --port 9000 --token mysecrettoken
  synheart receiver --out ./exports --format ndjson
  synheart receiver --host 0.0.0.0 --gzip`,
	RunE: runReceiver,
}

func init() {
	receiverCmd.Flags().StringVar(&receiverHost, "host", "0.0.0.0", "Host address to bind to")
	receiverCmd.Flags().IntVar(&receiverPort, "port", 8787, "Port to listen on")
	receiverCmd.Flags().StringVar(&receiverToken, "token", "", "Static bearer token (auto-generated if not provided)")
	receiverCmd.Flags().StringVar(&receiverOut, "out", "", "Directory to write received payloads (stdout if not set)")
	receiverCmd.Flags().StringVar(&receiverFormat, "format", "json", "Output format: json|ndjson")
	receiverCmd.Flags().BoolVar(&receiverGzip, "gzip", false, "Accept gzip-compressed payloads")
}

func runReceiver(cmd *cobra.Command, args []string) error {
	// Validate format
	receiverFormat = strings.ToLower(strings.TrimSpace(receiverFormat))
	if receiverFormat != "json" && receiverFormat != "ndjson" {
		return fmt.Errorf("invalid --format %q (expected: json|ndjson)", receiverFormat)
	}

	// Generate token if not provided
	token := receiverToken
	if token == "" {
		generated, err := generateToken()
		if err != nil {
			return fmt.Errorf("failed to generate token: %w", err)
		}
		token = generated
	}

	// Create writer
	var writer receiver.Writer
	if receiverOut != "" {
		fw, err := receiver.NewFileWriter(receiverOut, receiverFormat)
		if err != nil {
			return fmt.Errorf("failed to create file writer: %w", err)
		}
		writer = fw
	} else {
		writer = receiver.NewStdoutWriter(cmd.OutOrStdout(), receiverFormat)
	}
	defer writer.Close()

	// Create server config
	config := receiver.Config{
		Host:       receiverHost,
		Port:       receiverPort,
		Token:      token,
		OutDir:     receiverOut,
		Format:     receiverFormat,
		AcceptGzip: receiverGzip,
	}

	// Create server
	server := receiver.NewServer(config, writer)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintln(cmd.ErrOrStderr(), "\n⏹  Received interrupt signal, shutting down...")
		cancel()
	}()

	// Print startup banner
	printReceiverBanner(cmd, server.GetAddress(), token, receiverOut, receiverFormat, receiverGzip)

	// Start server (blocks until context is cancelled)
	if err := server.Start(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("server error: %w", err)
	}

	// Print final stats
	stats := server.GetStats()
	fmt.Fprintf(cmd.ErrOrStderr(), "\n📊 Session Stats:\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "   Received:   %d\n", stats.TotalReceived)
	fmt.Fprintf(cmd.ErrOrStderr(), "   Duplicates: %d\n", stats.TotalDuplicates)
	fmt.Fprintf(cmd.ErrOrStderr(), "   Errors:     %d\n", stats.TotalErrors)
	fmt.Fprintln(cmd.ErrOrStderr(), "\n✓ Shutdown complete")

	return nil
}

func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "sh_" + hex.EncodeToString(bytes), nil
}

func printReceiverBanner(cmd *cobra.Command, address, token, outDir, format string, gzip bool) {
	out := cmd.ErrOrStderr()

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "╔═══════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(out, "║                 🫀 Synheart Receiver Started                   ║")
	fmt.Fprintln(out, "╚═══════════════════════════════════════════════════════════════╝")
	fmt.Fprintln(out, "")
	fmt.Fprintf(out, "  Endpoint:  %s/v1/hsi/import\n", address)
	fmt.Fprintf(out, "  Token:     %s\n", token)
	fmt.Fprintln(out, "")

	if outDir != "" {
		fmt.Fprintf(out, "  Output:    %s/\n", outDir)
	} else {
		fmt.Fprintln(out, "  Output:    stdout")
	}
	fmt.Fprintf(out, "  Format:    %s\n", format)
	if gzip {
		fmt.Fprintln(out, "  Gzip:      enabled")
	}

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "───────────────────────────────────────────────────────────────────")
	fmt.Fprintln(out, "  Configure in Synheart Life:")
	fmt.Fprintln(out, "    Account → Data → Exports → Add Destination")
	fmt.Fprintln(out, "")
	fmt.Fprintf(out, "    Endpoint: %s/v1/hsi/import\n", address)
	fmt.Fprintf(out, "    Token:    %s\n", token)
	fmt.Fprintln(out, "───────────────────────────────────────────────────────────────────")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Waiting for exports... (Press Ctrl+C to stop)")
	fmt.Fprintln(out, "")
}
