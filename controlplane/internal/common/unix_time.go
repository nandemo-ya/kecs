package common

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// UnixTime is a custom time type that can unmarshal from both Unix timestamps and RFC3339 strings
type UnixTime struct {
	time.Time
}

// UnmarshalJSON handles both numeric Unix timestamps and RFC3339 string formats
func (t *UnixTime) UnmarshalJSON(data []byte) error {
	// Check for null
	if string(data) == "null" {
		t.Time = time.Time{}
		return nil
	}

	// First try to unmarshal as a float64 (Unix timestamp with fractional seconds)
	var timestamp float64
	if err := json.Unmarshal(data, &timestamp); err == nil {
		// Convert to milliseconds and round to avoid floating point precision issues
		millis := int64(math.Round(timestamp * 1000))
		sec := millis / 1000
		nsec := (millis % 1000) * 1e6
		t.Time = time.Unix(sec, nsec)
		return nil
	}

	// Fall back to string format (RFC3339)
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("time should be a Unix timestamp or RFC3339 string")
	}

	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}

	t.Time = parsed
	return nil
}

// MarshalJSON converts the time to a Unix timestamp with fractional seconds
func (t UnixTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	// Convert to Unix timestamp with fractional seconds
	timestamp := float64(t.UnixNano()) / 1e9
	return json.Marshal(timestamp)
}

// ToTime returns the underlying time.Time value
func (t *UnixTime) ToTime() *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	return &t.Time
}

// String returns the time formatted as RFC3339
func (t UnixTime) String() string {
	return t.Time.Format(time.RFC3339)
}