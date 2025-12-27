package cli

import (
	"fmt"
	"net"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/synheart/synheart-cli/internal/scenario"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment and print connection info",
	Long:  `Validates the local environment, checks port availability, and provides connection examples.`,
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("🏥 Synheart Environment Check")

	// Check Go version
	fmt.Printf("Go Version:        %s\n", runtime.Version())
	fmt.Printf("OS/Arch:           %s/%s\n\n", runtime.GOOS, runtime.GOARCH)

	// Check scenarios directory
	scenariosDir := getScenarioDir()
	if _, err := os.Stat(scenariosDir); err == nil {
		fmt.Printf("✅ Scenarios directory found: %s\n", scenariosDir)

		// Count scenarios
		registry := scenario.NewRegistry()
		if err := registry.LoadFromDir(scenariosDir); err == nil {
			scenarios := registry.List()
			fmt.Printf("   Found %d scenarios: %v\n\n", len(scenarios), scenarios)
		}
	} else {
		fmt.Printf("❌ Scenarios directory not found: %s\n\n", scenariosDir)
	}

	// Check default port availability
	defaultPort := 8787
	if isPortAvailable(defaultPort) {
		fmt.Printf("✅ Default port %d is available\n\n", defaultPort)
	} else {
		fmt.Printf("⚠️  Default port %d is in use\n", defaultPort)
		fmt.Printf("   Use --port flag to specify a different port\n\n")
	}

	// Print connection examples
	fmt.Println("📡 Connection Examples:")
	fmt.Println()

	fmt.Println("JavaScript/Node.js:")
	fmt.Println("  const ws = new WebSocket('ws://localhost:8787/hsi');")
	fmt.Println("  ws.onmessage = (event) => {")
	fmt.Println("    const data = JSON.parse(event.data);")
	fmt.Println("    console.log(data);")
	fmt.Println("  };")
	fmt.Println()

	fmt.Println("Python:")
	fmt.Println("  import websocket")
	fmt.Println("  import json")
	fmt.Println("  ws = websocket.WebSocket()")
	fmt.Println("  ws.connect('ws://localhost:8787/hsi')")
	fmt.Println("  while True:")
	fmt.Println("    data = json.loads(ws.recv())")
	fmt.Println("    print(data)")
	fmt.Println()

	fmt.Println("Go:")
	fmt.Println("  conn, _, err := websocket.DefaultDialer.Dial(\"ws://localhost:8787/hsi\", nil)")
	fmt.Println("  for {")
	fmt.Println("    _, message, err := conn.ReadMessage()")
	fmt.Println("    var event Event")
	fmt.Println("    json.Unmarshal(message, &event)")
	fmt.Println("  }")
	fmt.Println()

	fmt.Println("Rust:")
	fmt.Println("  let (mut socket, _) = connect(\"ws://localhost:8787/hsi\").await?;")
	fmt.Println("  while let Some(msg) = socket.next().await {")
	fmt.Println("    let event: Event = serde_json::from_str(&msg?.to_string())?;")
	fmt.Println("  }")
	fmt.Println()

	fmt.Println("✅ Environment check complete")
	return nil
}

func isPortAvailable(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}
