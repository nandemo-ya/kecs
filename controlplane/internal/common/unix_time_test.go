package common

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUnixTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, ut *UnixTime)
	}{
		{
			name:    "numeric unix timestamp",
			input:   `1755005925.233`,
			wantErr: false,
			check: func(t *testing.T, ut *UnixTime) {
				expected := time.Unix(1755005925, 233000000)
				if !ut.Time.Equal(expected) {
					t.Errorf("expected %v, got %v", expected, ut.Time)
				}
			},
		},
		{
			name:    "integer unix timestamp",
			input:   `1755005925`,
			wantErr: false,
			check: func(t *testing.T, ut *UnixTime) {
				expected := time.Unix(1755005925, 0)
				if !ut.Time.Equal(expected) {
					t.Errorf("expected %v, got %v", expected, ut.Time)
				}
			},
		},
		{
			name:    "RFC3339 string",
			input:   `"2025-01-10T15:30:00Z"`,
			wantErr: false,
			check: func(t *testing.T, ut *UnixTime) {
				expected, _ := time.Parse(time.RFC3339, "2025-01-10T15:30:00Z")
				if !ut.Time.Equal(expected) {
					t.Errorf("expected %v, got %v", expected, ut.Time)
				}
			},
		},
		{
			name:    "invalid format",
			input:   `"not a timestamp"`,
			wantErr: true,
		},
		{
			name:    "null value",
			input:   `null`,
			wantErr: false,
			check: func(t *testing.T, ut *UnixTime) {
				if !ut.Time.IsZero() {
					t.Errorf("expected zero time for null, got %v", ut.Time)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ut UnixTime
			err := json.Unmarshal([]byte(tt.input), &ut)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, &ut)
			}
		})
	}
}

func TestUnixTime_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		time    UnixTime
		want    string
		wantErr bool
	}{
		{
			name: "normal timestamp",
			time: UnixTime{Time: time.Unix(1755005925, 233000000)},
			want: `1755005925.233`,
		},
		{
			name: "zero time",
			time: UnixTime{},
			want: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.time.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("MarshalJSON() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestUnixTime_ToTime(t *testing.T) {
	tests := []struct {
		name string
		time *UnixTime
		want *time.Time
	}{
		{
			name: "valid time",
			time: &UnixTime{Time: time.Unix(1755005925, 0)},
			want: func() *time.Time { t := time.Unix(1755005925, 0); return &t }(),
		},
		{
			name: "nil UnixTime",
			time: nil,
			want: nil,
		},
		{
			name: "zero time",
			time: &UnixTime{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.time.ToTime()
			if (got == nil) != (tt.want == nil) {
				t.Errorf("ToTime() = %v, want %v", got, tt.want)
				return
			}
			if got != nil && tt.want != nil && !got.Equal(*tt.want) {
				t.Errorf("ToTime() = %v, want %v", *got, *tt.want)
			}
		})
	}
}