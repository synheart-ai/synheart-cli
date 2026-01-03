package recorder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/synheart/synheart-cli/internal/models"
)

// Replayer reads and replays events from an NDJSON file
type Replayer struct {
	filename   string
	speed      float64
	loop       bool
	eventCount int
	firstEvent *models.Event
	loaded     bool
}

// NewReplayer creates a new replayer
func NewReplayer(filename string, speed float64, loop bool) *Replayer {
	return &Replayer{
		filename: filename,
		speed:    speed,
		loop:     loop,
	}
}

// loadMetadata reads the file once to cache count and first event
func (r *Replayer) loadMetadata() error {
	if r.loaded {
		return nil
	}

	file, err := os.Open(r.filename)
	if err != nil {
		return fmt.Errorf("failed to open recording file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	r.eventCount = 0

	for scanner.Scan() {
		r.eventCount++
		if r.eventCount == 1 {
			var event models.Event
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				return fmt.Errorf("failed to parse first event: %w", err)
			}
			r.firstEvent = &event
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	r.loaded = true
	return nil
}

// Replay reads events and sends them to the output channel with timing
func (r *Replayer) Replay(ctx context.Context, output chan<- models.Event) error {
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

func (r *Replayer) replayOnce(ctx context.Context, output chan<- models.Event) error {
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

		var event models.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("failed to parse event at line %d: %w", lineNum, err)
		}

		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339Nano, event.Timestamp)
		if err != nil {
			return fmt.Errorf("failed to parse timestamp at line %d: %w", lineNum, err)
		}

		// Calculate delay
		if lineNum == 1 {
			lastTimestamp = timestamp
		} else {
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

			lastTimestamp = timestamp
		}

		// Send event
		select {
		case <-ctx.Done():
			return ctx.Err()
		case output <- event:
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

// CountEvents returns the number of events in the recording
func (r *Replayer) CountEvents() (int, error) {
	if err := r.loadMetadata(); err != nil {
		return 0, err
	}
	return r.eventCount, nil
}

// GetFirstEvent returns the first event in the recording
func (r *Replayer) GetFirstEvent() (*models.Event, error) {
	if err := r.loadMetadata(); err != nil {
		return nil, err
	}
	if r.firstEvent == nil {
		return nil, fmt.Errorf("recording file is empty")
	}
	return r.firstEvent, nil
}
