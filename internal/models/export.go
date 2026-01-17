package models

import "time"

// HSIExport represents the HSI Export Schema v1 payload
type HSIExport struct {
	Schema       string       `json:"schema"`
	ExportID     string       `json:"export_id"`
	CreatedAtUTC string       `json:"created_at_utc"`
	Range        ExportRange  `json:"range"`
	Device       ExportDevice `json:"device"`
	Summaries    []Summary    `json:"summaries"`
	Insights     []Insight    `json:"insights"`
}

// ExportRange represents the time range of the export
type ExportRange struct {
	FromUTC string `json:"from_utc"`
	ToUTC   string `json:"to_utc"`
}

// ExportDevice contains device metadata
type ExportDevice struct {
	Platform   string `json:"platform"`
	AppVersion string `json:"app_version"`
}

// Summary represents a summary entry in the export
type Summary struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Timestamp string         `json:"timestamp"`
	Data      map[string]any `json:"data,omitempty"`
}

// Insight represents an insight entry in the export
type Insight struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Timestamp string         `json:"timestamp"`
	Data      map[string]any `json:"data,omitempty"`
}

// Validate checks if the export payload is valid according to schema v1
func (e *HSIExport) Validate() error {
	if e.Schema != "synheart.hsi.export.v1" {
		return &ValidationError{Field: "schema", Message: "must be 'synheart.hsi.export.v1'"}
	}
	if e.ExportID == "" {
		return &ValidationError{Field: "export_id", Message: "is required"}
	}
	if e.CreatedAtUTC == "" {
		return &ValidationError{Field: "created_at_utc", Message: "is required"}
	}
	if _, err := time.Parse(time.RFC3339, e.CreatedAtUTC); err != nil {
		return &ValidationError{Field: "created_at_utc", Message: "must be valid RFC3339 timestamp"}
	}
	if e.Range.FromUTC == "" || e.Range.ToUTC == "" {
		return &ValidationError{Field: "range", Message: "from_utc and to_utc are required"}
	}
	if e.Device.Platform == "" {
		return &ValidationError{Field: "device.platform", Message: "is required"}
	}
	if e.Device.AppVersion == "" {
		return &ValidationError{Field: "device.app_version", Message: "is required"}
	}
	return nil
}

// ValidationError represents a schema validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ExportReceipt represents the summary of a received export
type ExportReceipt struct {
	ExportID     string `json:"export_id"`
	ReceivedAt   string `json:"received_at"`
	Range        string `json:"range"`
	SummaryCount int    `json:"summary_count"`
	InsightCount int    `json:"insight_count"`
	Platform     string `json:"platform"`
	Duplicate    bool   `json:"duplicate,omitempty"`
}

// NewExportReceipt creates a receipt from an HSI export
func NewExportReceipt(export *HSIExport, duplicate bool) ExportReceipt {
	return ExportReceipt{
		ExportID:     export.ExportID,
		ReceivedAt:   time.Now().UTC().Format(time.RFC3339),
		Range:        export.Range.FromUTC + " to " + export.Range.ToUTC,
		SummaryCount: len(export.Summaries),
		InsightCount: len(export.Insights),
		Platform:     export.Device.Platform,
		Duplicate:    duplicate,
	}
}
