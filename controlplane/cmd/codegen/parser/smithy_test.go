package parser

import (
	"testing"
)

func TestGetJSONName(t *testing.T) {
	tests := []struct {
		name      string
		member    *SmithyMember
		fieldName string
		expected  string
	}{
		{
			name:      "Should preserve PascalCase field name",
			member:    &SmithyMember{},
			fieldName: "SecretId",
			expected:  "SecretId",
		},
		{
			name:      "Should preserve mixed case field name",
			member:    &SmithyMember{},
			fieldName: "ARN",
			expected:  "ARN",
		},
		{
			name: "Should use jsonName trait when present",
			member: &SmithyMember{
				Traits: map[string]interface{}{
					"smithy.api#jsonName": "customName",
				},
			},
			fieldName: "OriginalName",
			expected:  "customName",
		},
		{
			name:      "Should preserve lowercase field name",
			member:    &SmithyMember{},
			fieldName: "lowercase",
			expected:  "lowercase",
		},
		{
			name:      "Should preserve complex PascalCase",
			member:    &SmithyMember{},
			fieldName: "VersionIdsToStages",
			expected:  "VersionIdsToStages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.member.GetJSONName(tt.fieldName)
			if result != tt.expected {
				t.Errorf("GetJSONName(%s) = %s, want %s", tt.fieldName, result, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SecretId", "secretId"},
		{"ARN", "arn"},
		{"VersionIdsToStages", "versionIdsToStages"},
		{"ID", "id"},
		{"HTTPSConnection", "hTTPSConnection"}, // Current implementation behavior
		{"lowercase", "lowercase"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("toCamelCase(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}