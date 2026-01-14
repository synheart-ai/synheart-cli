package transport

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/synheart/synheart-cli/internal/encoding"
	"github.com/synheart/synheart-cli/internal/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development
	},
}

type client struct {
	conn *websocket.Conn
	send chan []byte
}

// WebSocketServer broadcasts events to WebSocket clients
type WebSocketServer struct {
	host    string
	port    int
	clients map[*client]bool
	mu      sync.RWMutex
	server  *http.Server
	encoder encoding.Encoder
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(host string, port int, encoder encoding.Encoder) *WebSocketServer {
	return &WebSocketServer{
		host:    host,
		port:    port,
		clients: make(map[*client]bool),
		encoder: encoder,
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

func (s *WebSocketServer) writePump(c *client) {
	defer func() {
		c.conn.Close()
	}()

	for msg := range c.send {
		msgType := websocket.TextMessage
		if s.encoder.ContentType() == "application/x-protobuf" {
			msgType = websocket.BinaryMessage
		}

		// set a deadline If the network is too slow this will time out
		// and clean up the connection instead of hanging forever
		c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

		if err := c.conn.WriteMessage(msgType, msg); err != nil {
			return
		}
	}
}

// handleWebSocket handles WebSocket connections
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	c := &client{
		conn: conn,
		send: make(chan []byte, 256),
	}

	s.mu.Lock()
	s.clients[c] = true
	clientCount := len(s.clients)
	s.mu.Unlock()

	log.Printf("Client connected from %s (total: %d)", r.RemoteAddr, clientCount)

	go s.writePump(c)

	// Handle client disconnection
	defer func() {
		s.mu.Lock()
		// only close if Shutdown hasnt already done so
		if _, ok := s.clients[c]; ok {
			delete(s.clients, c)
			close(c.send)
		}
		currentCount := len(s.clients)
		s.mu.Unlock()
		log.Printf("Client disconnected (total: %d)", currentCount)
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
	if s.GetClientCount() == 0 {
		return nil
	}

	data, err := s.encoder.Encode(event)
	if err != nil {
		return fmt.Errorf("failed to encode event: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		select {
		case client.send <- data:
		default:
			log.Printf("Buffer overflow for client! Dropping real-time packet.")
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
	s.mu.Lock()
	// Close all client channels This triggers writePumps to finish and exit.
	for c := range s.clients {
		close(c.send)
		delete(s.clients, c)
	}
	s.mu.Unlock()

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// GetAddress returns the server address
func (s *WebSocketServer) GetAddress() string {
	return fmt.Sprintf("ws://%s:%d/hsi", s.host, s.port)
}
