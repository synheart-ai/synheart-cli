package receiver

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/synheart/synheart-cli/internal/models"
)

func TestHandleImport_ValidPayload(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf, "json")

	config := Config{
		Host:   "127.0.0.1",
		Port:   8787,
		Token:  "test-token",
		Format: "json",
	}

	server := NewServer(config, writer)

	// Create valid payload
	export := models.HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "test-export-123",
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

	body, _ := json.Marshal(export)

	req := httptest.NewRequest(http.MethodPost, "/v1/hsi/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Synheart-Schema", "synheart.hsi.export.v1")
	req.Header.Set("X-Synheart-Export-Id", "test-export-123")

	rr := httptest.NewRecorder()
	server.handleImport(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Check response body
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
}

func TestHandleImport_InvalidToken(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf, "json")

	config := Config{
		Host:   "127.0.0.1",
		Port:   8787,
		Token:  "correct-token",
		Format: "json",
	}

	server := NewServer(config, writer)

	req := httptest.NewRequest(http.MethodPost, "/v1/hsi/import", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")
	req.Header.Set("X-Synheart-Export-Id", "test-123")

	rr := httptest.NewRecorder()
	server.handleImport(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestHandleImport_MissingToken(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf, "json")

	config := Config{
		Host:   "127.0.0.1",
		Port:   8787,
		Token:  "test-token",
		Format: "json",
	}

	server := NewServer(config, writer)

	req := httptest.NewRequest(http.MethodPost, "/v1/hsi/import", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Synheart-Export-Id", "test-123")

	rr := httptest.NewRecorder()
	server.handleImport(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestHandleImport_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf, "json")

	config := Config{
		Host:   "127.0.0.1",
		Port:   8787,
		Token:  "test-token",
		Format: "json",
	}

	server := NewServer(config, writer)

	req := httptest.NewRequest(http.MethodPost, "/v1/hsi/import", bytes.NewReader([]byte("not valid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Synheart-Export-Id", "test-123")

	rr := httptest.NewRecorder()
	server.handleImport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleImport_InvalidSchema(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf, "json")

	config := Config{
		Host:   "127.0.0.1",
		Port:   8787,
		Token:  "test-token",
		Format: "json",
	}

	server := NewServer(config, writer)

	// Create payload with wrong schema
	export := models.HSIExport{
		Schema:       "wrong.schema.v1",
		ExportID:     "test-export-123",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: models.ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: models.ExportDevice{
			Platform:   "ios",
			AppVersion: "1.0.0",
		},
	}

	body, _ := json.Marshal(export)

	req := httptest.NewRequest(http.MethodPost, "/v1/hsi/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Synheart-Export-Id", "test-123")

	rr := httptest.NewRecorder()
	server.handleImport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleImport_MissingExportIDHeader(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf, "json")

	config := Config{
		Host:   "127.0.0.1",
		Port:   8787,
		Token:  "test-token",
		Format: "json",
	}

	server := NewServer(config, writer)

	req := httptest.NewRequest(http.MethodPost, "/v1/hsi/import", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	// Missing X-Synheart-Export-Id header

	rr := httptest.NewRecorder()
	server.handleImport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleImport_MethodNotAllowed(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf, "json")

	config := Config{
		Host:   "127.0.0.1",
		Port:   8787,
		Token:  "test-token",
		Format: "json",
	}

	server := NewServer(config, writer)

	req := httptest.NewRequest(http.MethodGet, "/v1/hsi/import", nil)

	rr := httptest.NewRecorder()
	server.handleImport(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleImport_Idempotency(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf, "json")

	config := Config{
		Host:   "127.0.0.1",
		Port:   8787,
		Token:  "test-token",
		Format: "json",
	}

	server := NewServer(config, writer)

	// Create valid payload
	export := models.HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "idempotent-test-123",
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

	body, _ := json.Marshal(export)

	// First request
	req1 := httptest.NewRequest(http.MethodPost, "/v1/hsi/import", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer test-token")
	req1.Header.Set("X-Synheart-Export-Id", "idempotent-test-123")
	req1.Header.Set("Idempotency-Key", "idempotent-test-123")

	rr1 := httptest.NewRecorder()
	server.handleImport(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("first request: expected status 200, got %d", rr1.Code)
	}

	// Check not marked as duplicate
	var resp1 map[string]any
	json.Unmarshal(rr1.Body.Bytes(), &resp1)
	receipt1 := resp1["receipt"].(map[string]any)
	if receipt1["duplicate"] == true {
		t.Error("first request should not be marked as duplicate")
	}

	// Second request with same idempotency key
	req2 := httptest.NewRequest(http.MethodPost, "/v1/hsi/import", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer test-token")
	req2.Header.Set("X-Synheart-Export-Id", "idempotent-test-123")
	req2.Header.Set("Idempotency-Key", "idempotent-test-123")

	rr2 := httptest.NewRecorder()
	server.handleImport(rr2, req2)

	// Should still succeed (idempotent)
	if rr2.Code != http.StatusOK {
		t.Errorf("second request: expected status 200, got %d", rr2.Code)
	}

	// Check marked as duplicate
	var resp2 map[string]any
	json.Unmarshal(rr2.Body.Bytes(), &resp2)
	receipt2 := resp2["receipt"].(map[string]any)
	if receipt2["duplicate"] != true {
		t.Error("second request should be marked as duplicate")
	}

	// Verify stats
	stats := server.GetStats()
	if stats.TotalReceived != 2 {
		t.Errorf("expected 2 total received, got %d", stats.TotalReceived)
	}
	if stats.TotalDuplicates != 1 {
		t.Errorf("expected 1 duplicate, got %d", stats.TotalDuplicates)
	}
}

func TestHandleImport_GzipPayload(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf, "json")

	config := Config{
		Host:       "127.0.0.1",
		Port:       8787,
		Token:      "test-token",
		Format:     "json",
		AcceptGzip: true,
	}

	server := NewServer(config, writer)

	// Create valid payload
	export := models.HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "gzip-test-123",
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

	body, _ := json.Marshal(export)

	// Compress body
	var compressed bytes.Buffer
	gzWriter := gzip.NewWriter(&compressed)
	gzWriter.Write(body)
	gzWriter.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/hsi/import", &compressed)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Synheart-Export-Id", "gzip-test-123")

	rr := httptest.NewRecorder()
	server.handleImport(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestIdempotencyStore(t *testing.T) {
	store := NewIdempotencyStore()

	// Initially not exists
	if store.Exists("key1") {
		t.Error("key1 should not exist initially")
	}

	// Mark and check
	store.Mark("key1")
	if !store.Exists("key1") {
		t.Error("key1 should exist after marking")
	}

	// Other keys still don't exist
	if store.Exists("key2") {
		t.Error("key2 should not exist")
	}
}
