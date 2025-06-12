package ptr_test

import (
	"testing"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
)

func TestString(t *testing.T) {
	// Test String function
	v := "test"
	p := ptr.String(v)
	if p == nil || *p != v {
		t.Errorf("Expected %s, got %v", v, p)
	}

	// Test ToString function
	if ptr.ToString(p) != v {
		t.Errorf("Expected %s, got %s", v, ptr.ToString(p))
	}

	// Test ToString with nil
	if ptr.ToString(nil) != "" {
		t.Errorf("Expected empty string for nil, got %s", ptr.ToString(nil))
	}
}

func TestBool(t *testing.T) {
	// Test Bool function
	v := true
	p := ptr.Bool(v)
	if p == nil || *p != v {
		t.Errorf("Expected %v, got %v", v, p)
	}

	// Test ToBool function
	if ptr.ToBool(p) != v {
		t.Errorf("Expected %v, got %v", v, ptr.ToBool(p))
	}

	// Test ToBool with nil
	if ptr.ToBool(nil) != false {
		t.Errorf("Expected false for nil, got %v", ptr.ToBool(nil))
	}
}

func TestInt32(t *testing.T) {
	// Test Int32 function
	v := int32(42)
	p := ptr.Int32(v)
	if p == nil || *p != v {
		t.Errorf("Expected %d, got %v", v, p)
	}

	// Test ToInt32 function
	if ptr.ToInt32(p) != v {
		t.Errorf("Expected %d, got %d", v, ptr.ToInt32(p))
	}

	// Test ToInt32 with nil
	if ptr.ToInt32(nil) != 0 {
		t.Errorf("Expected 0 for nil, got %d", ptr.ToInt32(nil))
	}
}

func TestTime(t *testing.T) {
	// Test Time function
	v := time.Now()
	p := ptr.Time(v)
	if p == nil || !p.Equal(v) {
		t.Errorf("Expected %v, got %v", v, p)
	}

	// Test ToTime function
	if !ptr.ToTime(p).Equal(v) {
		t.Errorf("Expected %v, got %v", v, ptr.ToTime(p))
	}

	// Test ToTime with nil
	if !ptr.ToTime(nil).IsZero() {
		t.Errorf("Expected zero time for nil, got %v", ptr.ToTime(nil))
	}
}

func TestStringSlice(t *testing.T) {
	// Test StringSlice function
	v := []string{"a", "b", "c"}
	p := ptr.StringSlice(v)
	
	if len(p) != len(v) {
		t.Errorf("Expected length %d, got %d", len(v), len(p))
	}
	
	for i := range v {
		if p[i] == nil || *p[i] != v[i] {
			t.Errorf("Expected %s at index %d, got %v", v[i], i, p[i])
		}
	}

	// Test ToStringSlice function
	result := ptr.ToStringSlice(p)
	if len(result) != len(v) {
		t.Errorf("Expected length %d, got %d", len(v), len(result))
	}
	
	for i := range v {
		if result[i] != v[i] {
			t.Errorf("Expected %s at index %d, got %s", v[i], i, result[i])
		}
	}

	// Test ToStringSlice with nil values
	pWithNil := []*string{ptr.String("a"), nil, ptr.String("c")}
	resultWithNil := ptr.ToStringSlice(pWithNil)
	if len(resultWithNil) != 2 {
		t.Errorf("Expected length 2 (skipping nil), got %d", len(resultWithNil))
	}
}

func TestStringMap(t *testing.T) {
	// Test StringMap function
	v := map[string]string{"key1": "value1", "key2": "value2"}
	p := ptr.StringMap(v)
	
	if len(p) != len(v) {
		t.Errorf("Expected length %d, got %d", len(v), len(p))
	}
	
	for k, val := range v {
		if p[k] == nil || *p[k] != val {
			t.Errorf("Expected %s for key %s, got %v", val, k, p[k])
		}
	}

	// Test ToStringMap function
	result := ptr.ToStringMap(p)
	if len(result) != len(v) {
		t.Errorf("Expected length %d, got %d", len(v), len(result))
	}
	
	for k, val := range v {
		if result[k] != val {
			t.Errorf("Expected %s for key %s, got %s", val, k, result[k])
		}
	}
}