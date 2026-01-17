package receiver

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/synheart/synheart-cli/internal/models"
)

func TestStdoutWriter_JSON(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf, "json")

	export := &models.HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "test-123",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: models.ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: models.ExportDevice{
			Platform:   "ios",
			AppVersion: "1.0.0",
		},
		Summaries: []models.Summary{},
		Insights:  []models.Insight{},
	}

	if err := writer.Write(export); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Verify output is valid JSON
	var parsed models.HSIExport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if parsed.ExportID != "test-123" {
		t.Errorf("expected export_id 'test-123', got '%s'", parsed.ExportID)
	}
}

func TestStdoutWriter_NDJSON(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf, "ndjson")

	export := &models.HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "test-456",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: models.ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: models.ExportDevice{
			Platform:   "android",
			AppVersion: "2.0.0",
		},
		Summaries: []models.Summary{},
		Insights:  []models.Insight{},
	}

	if err := writer.Write(export); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Output should be a single line
	output := buf.String()
	if output[len(output)-1] != '\n' {
		t.Error("NDJSON output should end with newline")
	}

	// Verify it's valid JSON on a single line
	var parsed models.HSIExport
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed.ExportID != "test-456" {
		t.Errorf("expected export_id 'test-456', got '%s'", parsed.ExportID)
	}
}

func TestFileWriter(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "synheart-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewFileWriter(tmpDir, "json")
	if err != nil {
		t.Fatalf("failed to create file writer: %v", err)
	}

	export := &models.HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "file-test-789",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: models.ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: models.ExportDevice{
			Platform:   "ios",
			AppVersion: "1.0.0",
		},
		Summaries: []models.Summary{
			{ID: "sum-1", Type: "activity", Timestamp: "2026-01-15T12:00:00Z"},
		},
		Insights: []models.Insight{
			{ID: "ins-1", Type: "stress", Timestamp: "2026-01-15T14:00:00Z"},
		},
	}

	if err := writer.Write(export); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, "synheart_export_file-test-789.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("expected file does not exist: %s", expectedPath)
	}

	// Read and verify content
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var parsed models.HSIExport
	if err := json.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("file content is not valid JSON: %v", err)
	}

	if parsed.ExportID != "file-test-789" {
		t.Errorf("expected export_id 'file-test-789', got '%s'", parsed.ExportID)
	}

	if len(parsed.Summaries) != 1 {
		t.Errorf("expected 1 summary, got %d", len(parsed.Summaries))
	}

	if len(parsed.Insights) != 1 {
		t.Errorf("expected 1 insight, got %d", len(parsed.Insights))
	}
}

func TestFileWriter_CreateDirectory(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "synheart-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Nested directory that doesn't exist
	nestedDir := filepath.Join(tmpDir, "nested", "exports")

	writer, err := NewFileWriter(nestedDir, "json")
	if err != nil {
		t.Fatalf("failed to create file writer: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("nested directory should have been created")
	}

	_ = writer.Close()
}

func TestMultiWriter(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	writer1 := NewStdoutWriter(&buf1, "json")
	writer2 := NewStdoutWriter(&buf2, "json")

	multi := NewMultiWriter(writer1, writer2)

	export := &models.HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "multi-test",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: models.ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: models.ExportDevice{
			Platform:   "ios",
			AppVersion: "1.0.0",
		},
		Summaries: []models.Summary{},
		Insights:  []models.Insight{},
	}

	if err := multi.Write(export); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Both buffers should have content
	if buf1.Len() == 0 {
		t.Error("buffer 1 should have content")
	}
	if buf2.Len() == 0 {
		t.Error("buffer 2 should have content")
	}

	// Content should be identical
	if buf1.String() != buf2.String() {
		t.Error("both buffers should have identical content")
	}
}
