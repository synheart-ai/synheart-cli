package receiver

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/synheart/synheart-cli/internal/models"
)

// Config holds the receiver server configuration
type Config struct {
	Host       string
	Port       int
	Token      string
	OutDir     string
	Format     string // "json" or "ndjson"
	AcceptGzip bool
}

// Server is the HTTP receiver server
type Server struct {
	config     Config
	writer     Writer
	idempotent *IdempotencyStore
	server     *http.Server
	mu         sync.RWMutex
	stats      Stats
}

// Stats holds server statistics
type Stats struct {
	TotalReceived   int
	TotalDuplicates int
	TotalErrors     int
}

// NewServer creates a new receiver server
func NewServer(config Config, writer Writer) *Server {
	return &Server{
		config:     config,
		writer:     writer,
		idempotent: NewIdempotencyStore(),
	}
}

// Start starts the receiver server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/hsi/import", s.handleImport)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleRoot)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
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
		return err
	}
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// GetAddress returns the server address
func (s *Server) GetAddress() string {
	return fmt.Sprintf("http://%s:%d", s.config.Host, s.config.Port)
}

// GetStats returns current server statistics
func (s *Server) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"service": "synheart-receiver",
		"version": "1.0.0",
		"endpoint": "/v1/hsi/import",
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	// Only accept POST
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Validate Authorization
	if !s.validateAuth(r) {
		s.mu.Lock()
		s.stats.TotalErrors++
		s.mu.Unlock()
		s.writeError(w, http.StatusUnauthorized, "invalid or missing authorization token")
		return
	}

	// Validate required headers
	if err := s.validateHeaders(r); err != nil {
		s.mu.Lock()
		s.stats.TotalErrors++
		s.mu.Unlock()
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get idempotency key
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		idempotencyKey = r.Header.Get("X-Synheart-Export-Id")
	}

	// Check for duplicate
	isDuplicate := s.idempotent.Exists(idempotencyKey)

	// Read body (with gzip support)
	body, err := s.readBody(r)
	if err != nil {
		s.mu.Lock()
		s.stats.TotalErrors++
		s.mu.Unlock()
		s.writeError(w, http.StatusBadRequest, "failed to read request body: "+err.Error())
		return
	}

	// Parse and validate payload
	var export models.HSIExport
	if err := json.Unmarshal(body, &export); err != nil {
		s.mu.Lock()
		s.stats.TotalErrors++
		s.mu.Unlock()
		s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Validate schema
	if err := export.Validate(); err != nil {
		s.mu.Lock()
		s.stats.TotalErrors++
		s.mu.Unlock()
		s.writeError(w, http.StatusBadRequest, "schema validation failed: "+err.Error())
		return
	}

	// Mark as seen for idempotency
	s.idempotent.Mark(idempotencyKey)

	// Write output
	if err := s.writer.Write(&export); err != nil {
		s.mu.Lock()
		s.stats.TotalErrors++
		s.mu.Unlock()
		s.writeError(w, http.StatusInternalServerError, "failed to write export: "+err.Error())
		return
	}

	// Update stats
	s.mu.Lock()
	s.stats.TotalReceived++
	if isDuplicate {
		s.stats.TotalDuplicates++
	}
	s.mu.Unlock()

	// Create receipt
	receipt := models.NewExportReceipt(&export, isDuplicate)

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"receipt": receipt,
	})
}

func (s *Server) validateAuth(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return false
	}

	return parts[1] == s.config.Token
}

func (s *Server) validateHeaders(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return fmt.Errorf("Content-Type must be application/json")
	}

	schema := r.Header.Get("X-Synheart-Schema")
	if schema != "" && schema != "synheart.hsi.export.v1" {
		return fmt.Errorf("unsupported schema version: %s", schema)
	}

	exportID := r.Header.Get("X-Synheart-Export-Id")
	if exportID == "" {
		return fmt.Errorf("X-Synheart-Export-Id header is required")
	}

	return nil
}

func (s *Server) readBody(r *http.Request) ([]byte, error) {
	var reader io.Reader = r.Body

	// Handle gzip if enabled and content is compressed
	if s.config.AcceptGzip && r.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress gzip: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Limit body size to 10MB
	limitReader := io.LimitReader(reader, 10*1024*1024)
	return io.ReadAll(limitReader)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// IdempotencyStore tracks processed export IDs
type IdempotencyStore struct {
	seen map[string]time.Time
	mu   sync.RWMutex
}

// NewIdempotencyStore creates a new idempotency store
func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{
		seen: make(map[string]time.Time),
	}
}

// Exists checks if an ID has been processed
func (s *IdempotencyStore) Exists(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.seen[id]
	return exists
}

// Mark records an ID as processed
func (s *IdempotencyStore) Mark(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[id] = time.Now()
}
