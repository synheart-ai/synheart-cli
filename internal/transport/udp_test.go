package transport

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/synheart/synheart-cli/internal/encoding"
	"github.com/synheart/synheart-cli/internal/models"
)

func TestUDPServer_Broadcast(t *testing.T) {
	server := NewUDPServer("127.0.0.1", 19878, encoding.NewJSONEncoder())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Create client
	clientAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	client, err := net.ListenUDP("udp", clientAddr)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Subscribe
	serverAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:19878")
	client.WriteToUDP([]byte("subscribe"), serverAddr)
	time.Sleep(100 * time.Millisecond)

	// Broadcast
	event := models.Event{
		SchemaVersion: "hsi.input.v1",
		EventID:       "udp-test-1",
		Signal:        models.Signal{Name: "udp.signal", Value: 99.0},
	}
	server.Broadcast(event)

	// Receive
	buf := make([]byte, 2048)
	client.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, err := client.Read(buf)
	if err != nil {
		t.Fatalf("failed to receive: %v", err)
	}

	if !strings.Contains(string(buf[:n]), "udp.signal") {
		t.Errorf("expected event data, got: %s", string(buf[:n]))
	}
}

func TestUDPServer_ClientCount(t *testing.T) {
	server := NewUDPServer("127.0.0.1", 19879, encoding.NewJSONEncoder())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	if server.GetClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", server.GetClientCount())
	}

	// Subscribe
	client, _ := net.Dial("udp", "127.0.0.1:19879")
	client.Write([]byte("subscribe"))
	time.Sleep(100 * time.Millisecond)

	if server.GetClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", server.GetClientCount())
	}

	// Unsubscribe
	client.Write([]byte("unsubscribe"))
	time.Sleep(100 * time.Millisecond)

	if server.GetClientCount() != 0 {
		t.Errorf("expected 0 clients after unsubscribe, got %d", server.GetClientCount())
	}

	client.Close()
}

func TestUDPServer_Address(t *testing.T) {
	server := NewUDPServer("127.0.0.1", 9999, encoding.NewJSONEncoder())
	addr := server.GetAddress()
	if addr != "udp://127.0.0.1:9999" {
		t.Errorf("wrong address: %s", addr)
	}
}
