package transport

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/synheart/synheart-cli/internal/encoding"
	"github.com/synheart/synheart-cli/internal/models"
)

func TestSSE_Stress(t *testing.T) {
	server := NewSSEServer("127.0.0.1", 18888, encoding.NewJSONEncoder())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	var totalReceived int64

	// Connect 5 clients
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req, _ := http.NewRequest("GET", "http://127.0.0.1:18888/hsi/sse", nil)
			reqCtx, reqCancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer reqCancel()
			req = req.WithContext(reqCtx)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			buf := make([]byte, 8192)
			for {
				n, err := resp.Body.Read(buf)
				if err == io.EOF || err != nil {
					break
				}
				count := strings.Count(string(buf[:n]), "data:")
				atomic.AddInt64(&totalReceived, int64(count))
			}
		}()
	}

	time.Sleep(200 * time.Millisecond)
	t.Logf("Connected clients: %d", server.GetClientCount())

	// Broadcast 50 events
	for i := 0; i < 50; i++ {
		event := models.Event{
			SchemaVersion: "hsi.input.v1",
			EventID:       fmt.Sprintf("stress-%d", i),
			Signal:        models.Signal{Name: "stress.test", Value: float64(i)},
		}
		server.Broadcast(event)
		time.Sleep(20 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)
	cancel()
	wg.Wait()

	t.Logf("Total received: %d (expected ~250)", totalReceived)
	if totalReceived < 200 {
		t.Errorf("Too many dropped: got %d, want >= 200", totalReceived)
	}
}

func TestUDP_Stress(t *testing.T) {
	server := NewUDPServer("127.0.0.1", 18889, encoding.NewJSONEncoder())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	var totalReceived int64

	// Connect 5 UDP clients
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			clientAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
			client, _ := net.ListenUDP("udp", clientAddr)
			defer client.Close()

			serverAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:18889")
			client.WriteToUDP([]byte("subscribe"), serverAddr)

			buf := make([]byte, 4096)
			client.SetReadDeadline(time.Now().Add(3 * time.Second))

			for {
				n, err := client.Read(buf)
				if err != nil {
					break
				}
				if strings.Contains(string(buf[:n]), "stress.test") {
					atomic.AddInt64(&totalReceived, 1)
				}
			}
		}()
	}

	time.Sleep(200 * time.Millisecond)
	t.Logf("Subscribed clients: %d", server.GetClientCount())

	// Broadcast 50 events
	for i := 0; i < 50; i++ {
		event := models.Event{
			SchemaVersion: "hsi.input.v1",
			EventID:       fmt.Sprintf("udp-stress-%d", i),
			Signal:        models.Signal{Name: "stress.test", Value: float64(i)},
		}
		server.Broadcast(event)
		time.Sleep(20 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)
	cancel()
	wg.Wait()

	t.Logf("Total received: %d (expected ~250)", totalReceived)
	if totalReceived < 200 {
		t.Errorf("Too many dropped: got %d, want >= 200", totalReceived)
	}
}
