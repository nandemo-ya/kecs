// Package ptr provides helper functions for converting between pointer and non-pointer values.
// This package is inspired by the AWS SDK v2 pointer helpers and provides a convenient way
// to work with pointer types in the generated ECS API code.
package ptr

import "time"

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

// ToString returns the value of the string pointer passed in or
// "" if the pointer is nil.
func ToString(p *string) string {
	if p != nil {
		return *p
	}
	return ""
}

// ToStringValue is an alias for ToString for compatibility.
func ToStringValue(p *string) string {
	return ToString(p)
}

// Bool returns a pointer to the bool value passed in.
func Bool(v bool) *bool {
	return &v
}

// ToBool returns the value of the bool pointer passed in or
// false if the pointer is nil.
func ToBool(p *bool) bool {
	if p != nil {
		return *p
	}
	return false
}

// Int returns a pointer to the int value passed in.
func Int(v int) *int {
	return &v
}

// ToInt returns the value of the int pointer passed in or
// 0 if the pointer is nil.
func ToInt(p *int) int {
	if p != nil {
		return *p
	}
	return 0
}

// Int32 returns a pointer to the int32 value passed in.
func Int32(v int32) *int32 {
	return &v
}

// ToInt32 returns the value of the int32 pointer passed in or
// 0 if the pointer is nil.
func ToInt32(p *int32) int32 {
	if p != nil {
		return *p
	}
	return 0
}

// Int64 returns a pointer to the int64 value passed in.
func Int64(v int64) *int64 {
	return &v
}

// ToInt64 returns the value of the int64 pointer passed in or
// 0 if the pointer is nil.
func ToInt64(p *int64) int64 {
	if p != nil {
		return *p
	}
	return 0
}

// Float32 returns a pointer to the float32 value passed in.
func Float32(v float32) *float32 {
	return &v
}

// ToFloat32 returns the value of the float32 pointer passed in or
// 0 if the pointer is nil.
func ToFloat32(p *float32) float32 {
	if p != nil {
		return *p
	}
	return 0
}

// Float64 returns a pointer to the float64 value passed in.
func Float64(v float64) *float64 {
	return &v
}

// ToFloat64 returns the value of the float64 pointer passed in or
// 0 if the pointer is nil.
func ToFloat64(p *float64) float64 {
	if p != nil {
		return *p
	}
	return 0
}

// Time returns a pointer to the time.Time value passed in.
func Time(v time.Time) *time.Time {
	return &v
}

// ToTime returns the value of the time.Time pointer passed in or
// time.Time{} if the pointer is nil.
func ToTime(p *time.Time) time.Time {
	if p != nil {
		return *p
	}
	return time.Time{}
}

// ToTimeValue returns the value of the time.Time pointer passed in or
// time.Time{} if the pointer is nil. This is an alias for ToTime.
func ToTimeValue(p *time.Time) time.Time {
	return ToTime(p)
}

// StringSlice returns a slice of string pointers from the values passed in.
func StringSlice(v []string) []*string {
	out := make([]*string, len(v))
	for i := range v {
		out[i] = &v[i]
	}
	return out
}

// ToStringSlice returns a slice of strings from the string pointers passed in.
// Nil pointers are skipped.
func ToStringSlice(p []*string) []string {
	out := make([]string, 0, len(p))
	for _, v := range p {
		if v != nil {
			out = append(out, *v)
		}
	}
	return out
}

// StringMap returns a map of string pointers from the values passed in.
func StringMap(v map[string]string) map[string]*string {
	out := make(map[string]*string, len(v))
	for k, val := range v {
		out[k] = String(val)
	}
	return out
}

// ToStringMap returns a map of strings from the string pointers passed in.
// Nil pointers are skipped.
func ToStringMap(p map[string]*string) map[string]string {
	out := make(map[string]string)
	for k, v := range p {
		if v != nil {
			out[k] = *v
		}
	}
	return out
}
