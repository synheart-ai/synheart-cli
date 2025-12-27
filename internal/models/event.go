package models

import "time"

// Event represents an HSI-compatible event envelope
type Event struct {
	SchemaVersion string   `json:"schema_version"`
	EventID       string   `json:"event_id"`
	Timestamp     string   `json:"ts"`
	Source        Source   `json:"source"`
	Session       Session  `json:"session"`
	Signal        Signal   `json:"signal"`
	Meta          Meta     `json:"meta"`
}

// Source represents the origin of the sensor data
type Source struct {
	Type string  `json:"type"` // "wearable" or "phone"
	ID   string  `json:"id"`
	Side *string `json:"side,omitempty"` // "left" or "right" for wearables
}

// Session contains metadata about the mock session
type Session struct {
	RunID    string `json:"run_id"`
	Scenario string `json:"scenario"`
	Seed     int64  `json:"seed"`
}

// Signal represents a single sensor measurement
type Signal struct {
	Name    string      `json:"name"`  // e.g., "ppg.hr_bpm"
	Unit    string      `json:"unit"`  // e.g., "bpm"
	Value   interface{} `json:"value"` // Can be number, string, or array
	Quality float64     `json:"quality"`
}

// Meta contains additional event metadata
type Meta struct {
	Sequence int64 `json:"sequence"`
}

// NewEvent creates a new Event with current timestamp
func NewEvent(eventID string, source Source, session Session, signal Signal, sequence int64) Event {
	return Event{
		SchemaVersion: "hsi.input.v1",
		EventID:       eventID,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Source:        source,
		Session:       session,
		Signal:        signal,
		Meta: Meta{
			Sequence: sequence,
		},
	}
}
