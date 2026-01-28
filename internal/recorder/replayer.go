package recorder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Replayer reads and replays records from an NDJSON file
type Replayer struct {
	filename string
	speed    float64
	loop     bool
}

// NewReplayer creates a new replayer
func NewReplayer(filename string, speed float64, loop bool) *Replayer {
	return &Replayer{
		filename: filename,
		speed:    speed,
		loop:     loop,
	}
}

// Replay reads records and sends them to the output channel with timing
func (r *Replayer) Replay(ctx context.Context, output chan<- []byte) error {
	for {
		if err := r.replayOnce(ctx, output); err != nil {
			return err
		}

		if !r.loop {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue looping
		}
	}

	return nil
}

func (r *Replayer) replayOnce(ctx context.Context, output chan<- []byte) error {
	file, err := os.Open(r.filename)
	if err != nil {
		return fmt.Errorf("failed to open recording file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lastTimestamp time.Time
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		data := scanner.Bytes()
		
		// Attempt to extract timestamp for timing
		timestamp := r.extractTimestamp(data)
		if timestamp.IsZero() {
			// Fallback: 100ms between records if no timestamp found
			if lineNum > 1 {
				time.Sleep(100 * time.Millisecond)
			}
		} else {
			// Calculate delay
			if !lastTimestamp.IsZero() {
				delay := timestamp.Sub(lastTimestamp)
				if r.speed != 1.0 {
					delay = time.Duration(float64(delay) / r.speed)
				}

				// Wait for the delay
				if delay > 0 {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(delay):
					}
				}
			}
			lastTimestamp = timestamp
		}

		// Send record
		select {
		case <-ctx.Done():
			return ctx.Err()
		case output <- append([]byte(nil), data...):
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

// extractTimestamp tries to find a timestamp in several known formats (Legacy Event, HSI 1.0)
func (r *Replayer) extractTimestamp(data []byte) time.Time {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return time.Time{}
	}

	// Try legacy event format: "ts": "..."
	if ts, ok := m["ts"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			return t
		}
	}

	// Try HSI 1.0 format: "provenance": { "observed_at_utc": "..." }
	if prov, ok := m["provenance"].(map[string]interface{}); ok {
		if ts, ok := prov["observed_at_utc"].(string); ok {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				return t
			}
		}
	}

	return time.Time{}
}

// CountEvents returns the number of records in the recording
func (r *Replayer) CountEvents() (int, error) {
	file, err := os.Open(r.filename)
	if err != nil {
		return 0, fmt.Errorf("failed to open recording file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading file: %w", err)
	}

	return count, nil
}

// GetFirstRecordInfo returns the first record as a map for info display
func (r *Replayer) GetFirstRecordInfo() (map[string]interface{}, error) {
	file, err := os.Open(r.filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open recording file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, fmt.Errorf("recording file is empty")
	}

	var m map[string]interface{}
	if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
		return nil, fmt.Errorf("failed to parse first record: %w", err)
	}

	return m, nil
}
