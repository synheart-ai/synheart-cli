package recorder

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"
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

// Record writes a raw byte payload to the file followed by a newline
func (r *Recorder) Record(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := r.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	if _, err := r.writer.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// RecordFromChannel reads data from a channel and records it
func (r *Recorder) RecordFromChannel(ctx context.Context, dataStream <-chan []byte, onEntry func()) error {
	for {
		select {
		case <-ctx.Done():
			return r.Close()
		case data, ok := <-dataStream:
			if !ok {
				return r.Close() // Channel closed
			}
			if err := r.Record(data); err != nil {
				return err
			}
			if onEntry != nil {
				onEntry()
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
