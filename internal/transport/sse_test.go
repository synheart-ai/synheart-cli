package transport

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/synheart/synheart-cli/internal/encoding"
	"github.com/synheart/synheart-cli/internal/models"
)

func TestSSEServer_Broadcast(t *testing.T) {
	server := NewSSEServer("127.0.0.1", 19876, encoding.NewJSONEncoder())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", "http://127.0.0.1:19876/hsi/sse", nil)
	reqCtx, reqCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer reqCancel()
	req = req.WithContext(reqCtx)

	go func() {
		time.Sleep(200 * time.Millisecond)
		event := models.Event{
			SchemaVersion: "hsi.input.v1",
			EventID:       "test-1",
			Signal:        models.Signal{Name: "test.signal", Value: 42.0},
		}
		server.Broadcast(event)
	}()

	resp, err := client.Do(req)
	if err != nil && !strings.Contains(err.Error(), "context") {
		t.Fatalf("failed to connect: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()

		if resp.Header.Get("Content-Type") != "text/event-stream" {
			t.Errorf("wrong content type: %s", resp.Header.Get("Content-Type"))
		}

		buf := make([]byte, 1024)
		n, _ := resp.Body.Read(buf)
		if n > 0 && !strings.Contains(string(buf[:n]), "test.signal") {
			t.Errorf("expected event data, got: %s", string(buf[:n]))
		}
	}
}

func TestSSEServer_ClientCount(t *testing.T) {
	server := NewSSEServer("127.0.0.1", 19877, encoding.NewJSONEncoder())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	if server.GetClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", server.GetClientCount())
	}

	reqCtx, reqCancel := context.WithCancel(context.Background())
	req, _ := http.NewRequest("GET", "http://127.0.0.1:19877/hsi/sse", nil)
	req = req.WithContext(reqCtx)

	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	time.Sleep(200 * time.Millisecond)
	if server.GetClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", server.GetClientCount())
	}

	reqCancel()
	time.Sleep(100 * time.Millisecond)
}

func TestSSEServer_Address(t *testing.T) {
	server := NewSSEServer("127.0.0.1", 8080, encoding.NewJSONEncoder())
	addr := server.GetAddress()
	if addr != "http://127.0.0.1:8080/hsi/sse" {
		t.Errorf("wrong address: %s", addr)
	}
}
func TestSSEServer_ShutdownWithClients(t *testing.T) {
	server := NewSSEServer("127.0.0.1", 19878, encoding.NewJSONEncoder())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	reqCtx, reqCancel := context.WithCancel(context.Background())
	defer reqCancel()
	req, _ := http.NewRequest("GET", "http://127.0.0.1:19878/hsi/sse", nil)
	req = req.WithContext(reqCtx)

	clientDone := make(chan struct{})
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		close(clientDone)
	}()

	time.Sleep(100 * time.Millisecond)
	if server.GetClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", server.GetClientCount())
	}

	err := server.Shutdown()
	if err != nil {
		t.Errorf("shutdown failed: %v", err)
	}

	select {
	case <-clientDone:
	case <-time.After(500 * time.Millisecond):
	}

	reqCancel()
}

func TestSSEServer_PortConflict(t *testing.T) {
	server1 := NewSSEServer("127.0.0.1", 19879, encoding.NewJSONEncoder())
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	go server1.Start(ctx1)
	time.Sleep(100 * time.Millisecond)

	server2 := NewSSEServer("127.0.0.1", 19879, encoding.NewJSONEncoder())
	ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel2()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server2.Start(ctx2)
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error for port conflict")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("should fail fast")
	}
}
