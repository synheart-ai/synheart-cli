package generator

import (
	"encoding/json"
	"time"

	"github.com/synheart/synheart-cli/internal/models"
)

// Aggregator collects individual events and packages them for Flux
type Aggregator struct {
	events []models.Event
}

func NewAggregator() *Aggregator {
	return &Aggregator{
		events: make([]models.Event, 0),
	}
}

func (a *Aggregator) Add(event models.Event) {
	a.events = append(a.events, event)
}

// ToWhoopJSON converts collected events to a Whoop-like JSON
func (a *Aggregator) ToWhoopJSON() (string, error) {
	type WhoopPayload struct {
		Sleep    []interface{} `json:"sleep"`
		Recovery []interface{} `json:"recovery"`
		Cycle    []interface{} `json:"cycle"`
	}

	payload := WhoopPayload{
		Sleep:    make([]interface{}, 0),
		Recovery: make([]interface{}, 0),
		Cycle:    make([]interface{}, 0),
	}

	now := time.Now().UTC()
	hrv, rhr := a.extractPhysiology()

	payload.Recovery = append(payload.Recovery, map[string]interface{}{
		"cycle_id":   1,
		"created_at": now.Format(time.RFC3339),
		"score": map[string]interface{}{
			"recovery_score":     75.0,
			"resting_heart_rate": rhr,
			"hrv_rmssd_milli":    hrv,
		},
	})

	payload.Cycle = append(payload.Cycle, map[string]interface{}{
		"id":    1,
		"start": now.Add(-12 * time.Hour).Format(time.RFC3339),
		"end":   now.Format(time.RFC3339),
		"score": map[string]interface{}{
			"strain":             12.5,
			"kilojoule":          8000.0,
			"average_heart_rate": rhr + 10,
			"max_heart_rate":     rhr + 50,
		},
	})

	payload.Sleep = append(payload.Sleep, map[string]interface{}{
		"id":    1,
		"start": now.Add(-20 * time.Hour).Format(time.RFC3339),
		"end":   now.Add(-12 * time.Hour).Format(time.RFC3339),
		"score": map[string]interface{}{
			"stage_summary": map[string]interface{}{
				"total_in_bed_time_milli":          28800000,
				"total_awake_time_milli":           1800000,
				"total_light_sleep_time_milli":    12600000,
				"total_slow_wave_sleep_time_milli": 7200000,
				"total_rem_sleep_time_milli":       7200000,
				"total_sleep_time_milli":           27000000,
				"disturbance_count":                3,
			},
			"sleep_performance_percentage": 85.0,
			"respiratory_rate":             14.5,
		},
	})

	bytes, err := json.Marshal(payload)
	return string(bytes), err
}

// ToGarminJSON converts collected events to a Garmin-like JSON
func (a *Aggregator) ToGarminJSON() (string, error) {
	type GarminPayload struct {
		Dailies []map[string]interface{} `json:"dailies"`
		Sleep   []map[string]interface{} `json:"sleep"`
	}

	hrv, rhr := a.extractPhysiology()
	today := time.Now().Format("2006-01-02")
	nowMs := time.Now().UnixMilli()

	payload := GarminPayload{
		Dailies: []map[string]interface{}{{
			"calendarDate":          today,
			"totalSteps":            8500,
			"totalKilocalories":     2200,
			"restingHeartRate":      int(rhr),
			"restingHeartRateHrv":   hrv,
			"averageHeartRate":      int(rhr + 10),
			"maxHeartRate":          int(rhr + 50),
			"bodyBatteryChargedValue": 72,
			"trainingLoadBalance":   45.5,
		}},
		Sleep: []map[string]interface{}{{
			"calendarDate":           today,
			"sleepTimeSeconds":       25200,
			"awakeSleepSeconds":      1800,
			"lightSleepSeconds":      10800,
			"deepSleepSeconds":       6300,
			"remSleepSeconds":        6300,
			"awakeCount":             2,
			"avgSleepRespiration":    13.5,
			"sleepScores": map[string]interface{}{
				"overallScore": 78.0,
			},
			"sleepStartTimestampGmt": nowMs - (20 * 3600 * 1000),
			"sleepEndTimestampGmt":   nowMs - (12 * 3600 * 1000),
		}},
	}

	bytes, err := json.Marshal(payload)
	return string(bytes), err
}

func (a *Aggregator) extractPhysiology() (float64, float64) {
	hrv := 50.0
	rhr := 60.0
	for _, e := range a.events {
		if e.Signal.Name == "ppg.hrv_rmssd_ms" {
			if v, ok := e.Signal.Value.(float64); ok {
				hrv = v
			}
		}
		if e.Signal.Name == "ppg.hr_bpm" {
			if v, ok := e.Signal.Value.(float64); ok {
				rhr = v
			}
		}
	}
	return hrv, rhr
}

func (a *Aggregator) Clear() {
	a.events = a.events[:0]
}

func (a *Aggregator) Count() int {
	return len(a.events)
}
