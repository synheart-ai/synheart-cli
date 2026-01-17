package models

import "testing"

func TestHSIExport_Validate_Valid(t *testing.T) {
	export := HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "test-123",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: ExportDevice{
			Platform:   "ios",
			AppVersion: "1.0.0",
		},
		Summaries: []Summary{},
		Insights:  []Insight{},
	}

	if err := export.Validate(); err != nil {
		t.Errorf("expected valid export, got error: %v", err)
	}
}

func TestHSIExport_Validate_InvalidSchema(t *testing.T) {
	export := HSIExport{
		Schema:       "wrong.schema",
		ExportID:     "test-123",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: ExportDevice{
			Platform:   "ios",
			AppVersion: "1.0.0",
		},
	}

	err := export.Validate()
	if err == nil {
		t.Error("expected error for invalid schema")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if valErr.Field != "schema" {
		t.Errorf("expected field 'schema', got '%s'", valErr.Field)
	}
}

func TestHSIExport_Validate_MissingExportID(t *testing.T) {
	export := HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: ExportDevice{
			Platform:   "ios",
			AppVersion: "1.0.0",
		},
	}

	err := export.Validate()
	if err == nil {
		t.Error("expected error for missing export_id")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if valErr.Field != "export_id" {
		t.Errorf("expected field 'export_id', got '%s'", valErr.Field)
	}
}

func TestHSIExport_Validate_InvalidTimestamp(t *testing.T) {
	export := HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "test-123",
		CreatedAtUTC: "not-a-timestamp",
		Range: ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: ExportDevice{
			Platform:   "ios",
			AppVersion: "1.0.0",
		},
	}

	err := export.Validate()
	if err == nil {
		t.Error("expected error for invalid timestamp")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if valErr.Field != "created_at_utc" {
		t.Errorf("expected field 'created_at_utc', got '%s'", valErr.Field)
	}
}

func TestHSIExport_Validate_MissingRange(t *testing.T) {
	export := HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "test-123",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: ExportRange{
			FromUTC: "",
			ToUTC:   "",
		},
		Device: ExportDevice{
			Platform:   "ios",
			AppVersion: "1.0.0",
		},
	}

	err := export.Validate()
	if err == nil {
		t.Error("expected error for missing range")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if valErr.Field != "range" {
		t.Errorf("expected field 'range', got '%s'", valErr.Field)
	}
}

func TestHSIExport_Validate_MissingPlatform(t *testing.T) {
	export := HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "test-123",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: ExportDevice{
			Platform:   "",
			AppVersion: "1.0.0",
		},
	}

	err := export.Validate()
	if err == nil {
		t.Error("expected error for missing platform")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if valErr.Field != "device.platform" {
		t.Errorf("expected field 'device.platform', got '%s'", valErr.Field)
	}
}

func TestHSIExport_Validate_MissingAppVersion(t *testing.T) {
	export := HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "test-123",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: ExportDevice{
			Platform:   "ios",
			AppVersion: "",
		},
	}

	err := export.Validate()
	if err == nil {
		t.Error("expected error for missing app_version")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if valErr.Field != "device.app_version" {
		t.Errorf("expected field 'device.app_version', got '%s'", valErr.Field)
	}
}

func TestNewExportReceipt(t *testing.T) {
	export := &HSIExport{
		Schema:       "synheart.hsi.export.v1",
		ExportID:     "receipt-test-123",
		CreatedAtUTC: "2026-01-16T12:00:00Z",
		Range: ExportRange{
			FromUTC: "2026-01-15T00:00:00Z",
			ToUTC:   "2026-01-16T00:00:00Z",
		},
		Device: ExportDevice{
			Platform:   "android",
			AppVersion: "2.0.0",
		},
		Summaries: []Summary{
			{ID: "s1", Type: "activity"},
			{ID: "s2", Type: "sleep"},
		},
		Insights: []Insight{
			{ID: "i1", Type: "stress"},
		},
	}

	receipt := NewExportReceipt(export, false)

	if receipt.ExportID != "receipt-test-123" {
		t.Errorf("expected export_id 'receipt-test-123', got '%s'", receipt.ExportID)
	}

	if receipt.SummaryCount != 2 {
		t.Errorf("expected summary_count 2, got %d", receipt.SummaryCount)
	}

	if receipt.InsightCount != 1 {
		t.Errorf("expected insight_count 1, got %d", receipt.InsightCount)
	}

	if receipt.Platform != "android" {
		t.Errorf("expected platform 'android', got '%s'", receipt.Platform)
	}

	if receipt.Duplicate {
		t.Error("expected duplicate to be false")
	}

	// Test duplicate flag
	receiptDup := NewExportReceipt(export, true)
	if !receiptDup.Duplicate {
		t.Error("expected duplicate to be true")
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "test_field",
		Message: "is invalid",
	}

	expected := "test_field: is invalid"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}
}
