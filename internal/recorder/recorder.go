package recorder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/synheart/synheart-cli/internal/models"
)

// Recorder writes events to an NDJSON file
type Recorder struct {
	file       *os.File
	writer     *bufio.Writer
	mu         sync.Mutex
	eventCount int64 // atomic counter for events recorded
}

// NewRecorder creates a new recorder
func NewRecorder(filename string) (*Recorder, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create recording file: %w", err)
	}

	return &Recorder{
		file:   file,
		writer: bufio.NewWriter(file),
	}, nil
}

// Record writes a single event to the file
func (r *Recorder) Record(event models.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if _, err := r.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	if _, err := r.writer.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	// Increment event counter atomically
	count := atomic.AddInt64(&r.eventCount, 1)

	// Flush every 100 events to prevent data loss on crash
	if count%100 == 0 {
		if err := r.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush buffer: %w", err)
		}
	}

	return nil
}

// GetCount returns the number of events that have been recorded.
// This is thread-safe and can be called concurrently.
func (r *Recorder) GetCount() int64 {
	return atomic.LoadInt64(&r.eventCount)
}

// RecordFromChannel reads events from a channel and records them
func (r *Recorder) RecordFromChannel(ctx context.Context, events <-chan models.Event) error {
	for {
		select {
		case <-ctx.Done():
			return r.Close()
		case event, ok := <-events:
			if !ok {
				return r.Close() // Channel closed
			}
			if err := r.Record(event); err != nil {
				return err
			}
		}
	}
}

// Flush flushes the buffer to disk
func (r *Recorder) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.writer.Flush()
}

// Close flushes and closes the recorder
func (r *Recorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.writer.Flush(); err != nil {
		r.file.Close()
		return fmt.Errorf("failed to flush buffer: %w", err)
	}

	if err := r.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return nil
}
