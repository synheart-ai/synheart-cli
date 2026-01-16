package recorder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/synheart/synheart-cli/internal/models"
)

// Recorder writes events to an NDJSON file
type Recorder struct {
	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex
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

	return nil
}

// RecordFromChannel reads events from a channel and records them
func (r *Recorder) RecordFromChannel(ctx context.Context, events <-chan models.Event, onEvent func()) error {
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
			if onEvent != nil { // <--- The Guard Logic
				onEvent()
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
