package common

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// UnixTime is a custom time type that handles AWS API's numeric Unix timestamps
// AWS APIs return timestamps as numbers (Unix epoch with fractional seconds)
// rather than RFC3339 strings.
type UnixTime struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler for UnixTime
// It can handle both numeric Unix timestamps and RFC3339 strings
func (t *UnixTime) UnmarshalJSON(data []byte) error {
	// First try to unmarshal as a number (Unix timestamp)
	var timestamp float64
	if err := json.Unmarshal(data, &timestamp); err == nil {
		// Convert Unix timestamp to time.Time
		// Round to millisecond precision to avoid floating point issues
		// AWS typically provides timestamps with millisecond precision
		millis := int64(math.Round(timestamp * 1000))
		sec := millis / 1000
		nsec := (millis % 1000) * 1e6
		t.Time = time.Unix(sec, nsec)
		return nil
	}

	// Fall back to string format (RFC3339)
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("cannot unmarshal %s into UnixTime", data)
	}

	parsedTime, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return fmt.Errorf("cannot parse %s as RFC3339: %w", str, err)
	}

	t.Time = parsedTime
	return nil
}

// MarshalJSON implements json.Marshaler for UnixTime
// It marshals the time as a Unix timestamp (number) to match AWS API format
func (t UnixTime) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}

	// Convert to Unix timestamp with fractional seconds
	timestamp := float64(t.Time.Unix()) + float64(t.Time.Nanosecond())/1e9
	
	// Round to 3 decimal places to match AWS format
	timestamp = math.Round(timestamp*1000) / 1000
	
	return json.Marshal(timestamp)
}

// NewUnixTime creates a new UnixTime from a time.Time
func NewUnixTime(t time.Time) *UnixTime {
	return &UnixTime{Time: t}
}

// ToTime converts UnixTime pointer to time.Time pointer
func (t *UnixTime) ToTime() *time.Time {
	if t == nil {
		return nil
	}
	return &t.Time
}

// FromTime converts time.Time pointer to UnixTime pointer
func FromTime(t *time.Time) *UnixTime {
	if t == nil {
		return nil
	}
	return &UnixTime{Time: *t}
}