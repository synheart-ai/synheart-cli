package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/synheart/synheart-cli/internal/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development
	},
}

// WebSocketServer broadcasts events to WebSocket clients
type WebSocketServer struct {
	host    string
	port    int
	clients map[*websocket.Conn]bool
	mu      sync.RWMutex
	server  *http.Server
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(host string, port int) *WebSocketServer {
	return &WebSocketServer{
		host:    host,
		port:    port,
		clients: make(map[*websocket.Conn]bool),
	}
}

// Start starts the WebSocket server
func (s *WebSocketServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/hsi", s.handleWebSocket)
	mux.HandleFunc("/", s.handleRoot)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		log.Printf("WebSocket server listening on ws://%s:%d/hsi", s.host, s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return s.Shutdown()
}

// handleRoot provides info at the root endpoint
func (s *WebSocketServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Synheart Mock Data Server\n\n")
	fmt.Fprintf(w, "WebSocket endpoint: ws://%s:%d/hsi\n", s.host, s.port)
	fmt.Fprintf(w, "Connected clients: %d\n", s.GetClientCount())
}

// handleWebSocket handles WebSocket connections
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	s.mu.Lock()
	s.clients[conn] = true
	clientCount := len(s.clients)
	s.mu.Unlock()

	log.Printf("Client connected from %s (total: %d)", r.RemoteAddr, clientCount)

	// Handle client disconnection
	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		clientCount := len(s.clients)
		s.mu.Unlock()

		conn.Close()
		log.Printf("Client disconnected (total: %d)", clientCount)
	}()

	// Keep connection alive and handle client messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// Broadcast sends an event to all connected clients
func (s *WebSocketServer) Broadcast(event models.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Failed to send to client: %v", err)
			// Client will be cleaned up by the connection handler
		}
	}

	return nil
}

// BroadcastFromChannel reads events from a channel and broadcasts them
func (s *WebSocketServer) BroadcastFromChannel(ctx context.Context, events <-chan models.Event) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil // Channel closed
			}
			if err := s.Broadcast(event); err != nil {
				log.Printf("Broadcast error: %v", err)
			}
		}
	}
}

// GetClientCount returns the number of connected clients
func (s *WebSocketServer) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// Shutdown gracefully shuts down the server
func (s *WebSocketServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close all client connections
	s.mu.Lock()
	for client := range s.clients {
		client.Close()
	}
	s.clients = make(map[*websocket.Conn]bool)
	s.mu.Unlock()

	// Shutdown HTTP server
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// GetAddress returns the server address
func (s *WebSocketServer) GetAddress() string {
	return fmt.Sprintf("ws://%s:%d/hsi", s.host, s.port)
}
