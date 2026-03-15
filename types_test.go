package threads

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimeMarshalJSON(t *testing.T) {
	now := time.Date(2025, 3, 15, 10, 30, 0, 0, time.UTC)
	threadsTime := &Time{Time: now}

	data, err := threadsTime.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	// Should produce a valid JSON string in RFC3339 format
	var result string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Parse the result back
	parsed, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Fatalf("Failed to parse RFC3339 time: %v", err)
	}

	if !parsed.Equal(now) {
		t.Errorf("Expected time %v, got %v", now, parsed)
	}
}

func TestTimeMarshalJSONRoundTrip(t *testing.T) {
	original := &Time{Time: time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal back
	var result Time
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !result.Time.Equal(original.Time) {
		t.Errorf("Round trip failed: expected %v, got %v", original.Time, result.Time)
	}
}

func TestTimeMarshalJSON_InStruct(t *testing.T) {
	type TestStruct struct {
		Timestamp Time `json:"timestamp"`
	}

	ts := TestStruct{
		Timestamp: Time{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	data, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result TestStruct
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !result.Timestamp.Time.Equal(ts.Timestamp.Time) {
		t.Errorf("Expected %v, got %v", ts.Timestamp.Time, result.Timestamp.Time)
	}
}
