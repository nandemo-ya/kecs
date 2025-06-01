package utils

import (
	"strings"
	"testing"
)

func TestGenerateRandomName(t *testing.T) {
	// Test multiple times to ensure randomness
	names := make(map[string]bool)
	
	for i := 0; i < 100; i++ {
		name, err := GenerateRandomName()
		if err != nil {
			t.Fatalf("GenerateRandomName() error = %v", err)
		}
		
		// Check format: should be "adjective-noun"
		parts := strings.Split(name, "-")
		if len(parts) != 2 {
			t.Errorf("GenerateRandomName() = %v, want format 'adjective-noun'", name)
		}
		
		// Check that adjective and noun are from our lists
		adjFound := false
		for _, adj := range adjectives {
			if parts[0] == adj {
				adjFound = true
				break
			}
		}
		if !adjFound {
			t.Errorf("Adjective '%s' not found in adjectives list", parts[0])
		}
		
		nounFound := false
		for _, noun := range nouns {
			if parts[1] == noun {
				nounFound = true
				break
			}
		}
		if !nounFound {
			t.Errorf("Noun '%s' not found in nouns list", parts[1])
		}
		
		names[name] = true
	}
	
	// Check that we got different names (at least 50% unique in 100 attempts)
	if len(names) < 50 {
		t.Errorf("Got only %d unique names out of 100 attempts, expected more variety", len(names))
	}
}

func TestGenerateClusterName(t *testing.T) {
	for i := 0; i < 10; i++ {
		name, err := GenerateClusterName()
		if err != nil {
			t.Fatalf("GenerateClusterName() error = %v", err)
		}
		
		// Check that it starts with "kecs-"
		if !strings.HasPrefix(name, "kecs-") {
			t.Errorf("GenerateClusterName() = %v, want prefix 'kecs-'", name)
		}
		
		// Check format after prefix
		withoutPrefix := strings.TrimPrefix(name, "kecs-")
		parts := strings.Split(withoutPrefix, "-")
		if len(parts) != 2 {
			t.Errorf("GenerateClusterName() = %v, want format 'kecs-adjective-noun'", name)
		}
	}
}

func TestGenerateClusterNameWithFallback(t *testing.T) {
	// Test normal case - should generate random name
	name := GenerateClusterNameWithFallback("fallback")
	if !strings.HasPrefix(name, "kecs-") {
		t.Errorf("GenerateClusterNameWithFallback() = %v, want prefix 'kecs-'", name)
	}
	
	// The name should have 3 parts: kecs-adjective-noun
	parts := strings.Split(name, "-")
	if len(parts) != 3 {
		t.Errorf("GenerateClusterNameWithFallback() = %v, want format 'kecs-adjective-noun'", name)
	}
}

func BenchmarkGenerateRandomName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateRandomName()
	}
}

func BenchmarkGenerateClusterName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateClusterName()
	}
}