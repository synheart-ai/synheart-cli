package transport

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// SSEServer broadcasts events via Server-Sent Events
type SSEServer struct {
	host    string
	port    int
	clients map[chan []byte]bool
	mu      sync.RWMutex
	server  *http.Server
}

// NewSSEServer creates a new SSE server
func NewSSEServer(host string, port int) *SSEServer {
	return &SSEServer{
		host:    host,
		port:    port,
		clients: make(map[chan []byte]bool),
	}
}

// Start starts the SSE server
func (s *SSEServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/hsi/sse", s.handleSSE)
	mux.HandleFunc("/", s.handleRoot)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		return s.Shutdown()
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("SSE server failed: %w", err)
		}
		return nil
	}
}

func (s *SSEServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Synheart SSE Server\n\nEndpoint: http://%s:%d/hsi/sse\n", s.host, s.port)
}

func (s *SSEServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan []byte, 100)
	s.addClient(clientChan)
	defer s.removeClient(clientChan)

	log.Printf("SSE client connected (total: %d)", s.GetClientCount())

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-clientChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *SSEServer) addClient(ch chan []byte) {
	s.mu.Lock()
	s.clients[ch] = true
	s.mu.Unlock()
}

func (s *SSEServer) removeClient(ch chan []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.clients[ch]; exists {
		delete(s.clients, ch)
		close(ch)
		log.Printf("SSE client disconnected (total: %d)", len(s.clients))
	}
}

// Broadcast sends data to all connected clients
func (s *SSEServer) Broadcast(data []byte) error {
	if s.GetClientCount() == 0 {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for ch := range s.clients {
		select {
		case ch <- data:
		default:
		}
	}
	return nil
}

// BroadcastFromChannel reads data from a channel and broadcasts it
func (s *SSEServer) BroadcastFromChannel(ctx context.Context, dataStream <-chan []byte) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-dataStream:
			if !ok {
				return nil
			}
			if err := s.Broadcast(data); err != nil {
				log.Printf("Broadcast error: %v", err)
			}
		}
	}
}

// GetClientCount returns connected client count
func (s *SSEServer) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// Shutdown gracefully stops the server
func (s *SSEServer) Shutdown() error {
	s.mu.Lock()
	for ch := range s.clients {
		close(ch)
	}
	s.clients = make(map[chan []byte]bool)
	s.mu.Unlock()

	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// GetAddress returns the server address
func (s *SSEServer) GetAddress() string {
	return fmt.Sprintf("http://%s:%d/hsi/sse", s.host, s.port)
}
