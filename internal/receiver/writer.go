package receiver

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/synheart/synheart-cli/internal/models"
)

// Writer defines the interface for export output writers
type Writer interface {
	Write(export *models.HSIExport) error
	Close() error
}

// StdoutWriter writes exports to stdout
type StdoutWriter struct {
	out    io.Writer
	format string // "json" or "ndjson"
	mu     sync.Mutex
}

// NewStdoutWriter creates a new stdout writer
func NewStdoutWriter(out io.Writer, format string) *StdoutWriter {
	return &StdoutWriter{
		out:    out,
		format: format,
	}
}

// Write writes an export to stdout
func (w *StdoutWriter) Write(export *models.HSIExport) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var data []byte
	var err error

	if w.format == "ndjson" {
		data, err = json.Marshal(export)
		if err != nil {
			return fmt.Errorf("failed to marshal export: %w", err)
		}
		data = append(data, '\n')
	} else {
		// Pretty print JSON
		data, err = json.MarshalIndent(export, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal export: %w", err)
		}
		data = append(data, '\n')
	}

	_, err = w.out.Write(data)
	return err
}

// Close is a no-op for stdout writer
func (w *StdoutWriter) Close() error {
	return nil
}

// FileWriter writes exports to individual files in a directory
type FileWriter struct {
	dir    string
	format string
	mu     sync.Mutex
}

// NewFileWriter creates a new file writer
func NewFileWriter(dir string, format string) (*FileWriter, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &FileWriter{
		dir:    dir,
		format: format,
	}, nil
}

// Write writes an export to a file
func (w *FileWriter) Write(export *models.HSIExport) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	filename := fmt.Sprintf("synheart_export_%s.json", export.ExportID)
	filepath := filepath.Join(w.dir, filename)

	var data []byte
	var err error

	if w.format == "ndjson" {
		data, err = json.Marshal(export)
	} else {
		data, err = json.MarshalIndent(export, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("failed to marshal export: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Close is a no-op for file writer
func (w *FileWriter) Close() error {
	return nil
}

// MultiWriter writes to multiple destinations
type MultiWriter struct {
	writers []Writer
}

// NewMultiWriter creates a writer that writes to multiple destinations
func NewMultiWriter(writers ...Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

// Write writes to all underlying writers
func (w *MultiWriter) Write(export *models.HSIExport) error {
	for _, writer := range w.writers {
		if err := writer.Write(export); err != nil {
			return err
		}
	}
	return nil
}

// Close closes all underlying writers
func (w *MultiWriter) Close() error {
	for _, writer := range w.writers {
		if err := writer.Close(); err != nil {
			return err
		}
	}
	return nil
}
