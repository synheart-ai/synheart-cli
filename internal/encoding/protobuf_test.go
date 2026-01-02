package encoding

import (
	"testing"

	"github.com/synheart/synheart-cli/internal/models"
	"github.com/synheart/synheart-cli/internal/proto/hsi"
	"google.golang.org/protobuf/proto"
)

func TestProtobufEncoder_ScalarValue(t *testing.T) {
	enc := NewProtobufEncoder()
	side := "left"

	event := models.Event{
		SchemaVersion: "hsi.input.v1",
		EventID:       "test-123",
		Timestamp:     "2025-01-02T10:00:00Z",
		Source:        models.Source{Type: "wearable", ID: "watch-1", Side: &side},
		Session:       models.Session{RunID: "run-1", Scenario: "baseline", Seed: 42},
		Signal:        models.Signal{Name: "ppg.hr_bpm", Unit: "bpm", Value: 72.5, Quality: 0.95},
		Meta:          models.Meta{Sequence: 1},
	}

	data, err := enc.Encode(event)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var pb hsi.Event
	if err := proto.Unmarshal(data, &pb); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if pb.SchemaVersion != "hsi.input.v1" {
		t.Errorf("schema version = %q, want hsi.input.v1", pb.SchemaVersion)
	}
	if pb.EventId != "test-123" {
		t.Errorf("event_id = %q, want test-123", pb.EventId)
	}
	if pb.Source.GetSide() != "left" {
		t.Errorf("source.side = %q, want left", pb.Source.GetSide())
	}
	if pb.Signal.Value.GetScalar() != 72.5 {
		t.Errorf("signal.value.scalar = %v, want 72.5", pb.Signal.Value.GetScalar())
	}
}

func TestProtobufEncoder_VectorValue(t *testing.T) {
	enc := NewProtobufEncoder()

	event := models.Event{
		SchemaVersion: "hsi.input.v1",
		EventID:       "accel-456",
		Timestamp:     "2025-01-02T10:00:00Z",
		Source:        models.Source{Type: "phone", ID: "phone-1"},
		Session:       models.Session{RunID: "run-1", Scenario: "workout", Seed: 100},
		Signal:        models.Signal{Name: "accel.xyz", Unit: "m/s^2", Value: []float64{0.1, -9.8, 0.3}, Quality: 1.0},
		Meta:          models.Meta{Sequence: 2},
	}

	data, err := enc.Encode(event)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var pb hsi.Event
	if err := proto.Unmarshal(data, &pb); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	vec := pb.Signal.Value.GetVector()
	if vec == nil {
		t.Fatal("expected vector value, got nil")
	}
	if vec.X != 0.1 || vec.Y != -9.8 || vec.Z != 0.3 {
		t.Errorf("vector = (%v, %v, %v), want (0.1, -9.8, 0.3)", vec.X, vec.Y, vec.Z)
	}
}

func TestProtobufEncoder_TextValue(t *testing.T) {
	enc := NewProtobufEncoder()

	event := models.Event{
		SchemaVersion: "hsi.input.v1",
		EventID:       "status-789",
		Timestamp:     "2025-01-02T10:00:00Z",
		Source:        models.Source{Type: "phone", ID: "phone-1"},
		Session:       models.Session{RunID: "run-1", Scenario: "baseline", Seed: 1},
		Signal:        models.Signal{Name: "device.status", Unit: "", Value: "active", Quality: 1.0},
		Meta:          models.Meta{Sequence: 3},
	}

	data, err := enc.Encode(event)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var pb hsi.Event
	if err := proto.Unmarshal(data, &pb); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if pb.Signal.Value.GetText() != "active" {
		t.Errorf("signal.value.text = %q, want active", pb.Signal.Value.GetText())
	}
}

func TestProtobufEncoder_ContentType(t *testing.T) {
	enc := NewProtobufEncoder()
	if ct := enc.ContentType(); ct != "application/x-protobuf" {
		t.Errorf("content type = %q, want application/x-protobuf", ct)
	}
}

func TestNewEncoder_Factory(t *testing.T) {
	jsonEnc := NewEncoder(FormatJSON)
	if jsonEnc.ContentType() != "application/json" {
		t.Errorf("json encoder content type = %q", jsonEnc.ContentType())
	}

	protoEnc := NewEncoder(FormatProtobuf)
	if protoEnc.ContentType() != "application/x-protobuf" {
		t.Errorf("protobuf encoder content type = %q", protoEnc.ContentType())
	}
}
