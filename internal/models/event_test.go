package models

import (
	"encoding/json"
	"testing"
)

func TestNewEvent(t *testing.T) {
	source := Source{
		Type: "wearable",
		ID:   "test-watch",
	}

	session := Session{
		RunID:    "test-run",
		Scenario: "baseline",
		Seed:     42,
	}

	signal := Signal{
		Name:    "ppg.hr_bpm",
		Unit:    "bpm",
		Value:   72.5,
		Quality: 0.95,
	}

	event := NewEvent("test-event-id", source, session, signal, 123)

	if event.SchemaVersion != "hsi.input.v1" {
		t.Errorf("Expected schema version 'hsi.input.v1', got %s", event.SchemaVersion)
	}

	if event.EventID != "test-event-id" {
		t.Errorf("Expected event ID 'test-event-id', got %s", event.EventID)
	}

	if event.Meta.Sequence != 123 {
		t.Errorf("Expected sequence 123, got %d", event.Meta.Sequence)
	}

	if event.Source.Type != "wearable" {
		t.Errorf("Expected source type 'wearable', got %s", event.Source.Type)
	}
}

func TestEventJSONMarshaling(t *testing.T) {
	left := "left"
	source := Source{
		Type: "wearable",
		ID:   "test-watch",
		Side: &left,
	}

	session := Session{
		RunID:    "test-run",
		Scenario: "baseline",
		Seed:     42,
	}

	signal := Signal{
		Name:    "ppg.hr_bpm",
		Unit:    "bpm",
		Value:   72.5,
		Quality: 0.95,
	}

	event := NewEvent("test-event-id", source, session, signal, 123)

	// Marshal to JSON
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Unmarshal back
	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	// Verify fields
	if decoded.EventID != event.EventID {
		t.Errorf("Event ID mismatch after marshal/unmarshal")
	}

	if decoded.Signal.Name != "ppg.hr_bpm" {
		t.Errorf("Signal name mismatch after marshal/unmarshal")
	}

	if decoded.Source.Side == nil || *decoded.Source.Side != "left" {
		t.Errorf("Source side mismatch after marshal/unmarshal")
	}
}
