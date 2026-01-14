package transport

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/synheart/synheart-cli/internal/encoding"
	"github.com/synheart/synheart-cli/internal/models"
)

// UDPServer broadcasts events via UDP
type UDPServer struct {
	host    string
	port    int
	encoder encoding.Encoder
	conn    *net.UDPConn
	clients map[string]*net.UDPAddr
	mu      sync.RWMutex
}

// NewUDPServer creates a new UDP server
func NewUDPServer(host string, port int, encoder encoding.Encoder) *UDPServer {
	return &UDPServer{
		host:    host,
		port:    port,
		encoder: encoder,
		clients: make(map[string]*net.UDPAddr),
	}
}

// Start starts the UDP server
func (s *UDPServer) Start(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", s.host, s.port))
	if err != nil {
		return fmt.Errorf("failed to resolve address: %w", err)
	}

	s.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	log.Printf("UDP server listening on %s:%d", s.host, s.port)

	go s.readLoop(ctx)

	<-ctx.Done()
	return s.Shutdown()
}

// readLoop listens for client registration packets
func (s *UDPServer) readLoop(ctx context.Context) {
	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			s.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, addr, err := s.conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}

			msg := string(buf[:n])
			s.handleMessage(msg, addr)
		}
	}
}

func (s *UDPServer) handleMessage(msg string, addr *net.UDPAddr) {
	key := addr.String()

	s.mu.Lock()
	defer s.mu.Unlock()

	switch msg {
	case "subscribe":
		s.clients[key] = addr
		log.Printf("UDP client subscribed: %s (total: %d)", key, len(s.clients))
	case "unsubscribe":
		delete(s.clients, key)
		log.Printf("UDP client unsubscribed: %s (total: %d)", key, len(s.clients))
	default:
		// Any message registers client
		if _, exists := s.clients[key]; !exists {
			s.clients[key] = addr
			log.Printf("UDP client registered: %s (total: %d)", key, len(s.clients))
		}
	}
}

// Broadcast sends an event to all registered clients
func (s *UDPServer) Broadcast(event models.Event) error {
	if s.GetClientCount() == 0 {
		return nil
	}

	data, err := s.encoder.Encode(event)
	if err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, addr := range s.clients {
		s.conn.WriteToUDP(data, addr)
	}
	return nil
}

// BroadcastFromChannel reads events and broadcasts them
func (s *UDPServer) BroadcastFromChannel(ctx context.Context, events <-chan models.Event) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil
			}
			s.Broadcast(event)
		}
	}
}

// GetClientCount returns registered client count
func (s *UDPServer) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// Shutdown closes the UDP connection
func (s *UDPServer) Shutdown() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// GetAddress returns the server address
func (s *UDPServer) GetAddress() string {
	return fmt.Sprintf("udp://%s:%d", s.host, s.port)
}
