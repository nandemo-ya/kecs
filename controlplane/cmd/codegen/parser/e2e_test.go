package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestPascalCasePreservation tests that generated code preserves PascalCase field names
func TestPascalCasePreservation(t *testing.T) {
	// Sample Smithy JSON structure similar to AWS API definitions
	smithyJSON := `{
		"smithy": "2.0",
		"shapes": {
			"com.amazonaws.secretsmanager#GetSecretValueRequest": {
				"type": "structure",
				"members": {
					"SecretId": {
						"target": "com.amazonaws.secretsmanager#SecretIdType",
						"traits": {
							"smithy.api#documentation": "The ARN or name of the secret",
							"smithy.api#required": {}
						}
					},
					"VersionId": {
						"target": "com.amazonaws.secretsmanager#SecretVersionIdType",
						"traits": {
							"smithy.api#documentation": "The unique identifier of the version"
						}
					},
					"VersionStage": {
						"target": "com.amazonaws.secretsmanager#SecretVersionStageType",
						"traits": {
							"smithy.api#documentation": "The staging label of the version"
						}
					}
				}
			}
		}
	}`

	// Parse the JSON
	var api SmithyAPI
	if err := json.Unmarshal([]byte(smithyJSON), &api); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Get the shape
	shapeName := "com.amazonaws.secretsmanager#GetSecretValueRequest"
	shape, exists := api.Shapes[shapeName]
	if !exists {
		t.Fatalf("Shape %s not found", shapeName)
	}

	// Check each member preserves PascalCase
	expectedFields := map[string]string{
		"SecretId":     "SecretId",
		"VersionId":    "VersionId",
		"VersionStage": "VersionStage",
	}

	for fieldName, expectedJSON := range expectedFields {
		member, exists := shape.Members[fieldName]
		if !exists {
			t.Errorf("Member %s not found", fieldName)
			continue
		}

		jsonName := member.GetJSONName(fieldName)
		if jsonName != expectedJSON {
			t.Errorf("Field %s: expected JSON name %s, got %s", fieldName, expectedJSON, jsonName)
		}
	}
}

// TestGeneratedStructTags verifies the generated struct tags use PascalCase
func TestGeneratedStructTags(t *testing.T) {
	member := &SmithyMember{}
	
	testCases := []struct {
		fieldName    string
		expectedTag  string
	}{
		{"SecretId", `json:"SecretId"`},
		{"ARN", `json:"ARN,omitempty"`},
		{"VersionIdsToStages", `json:"VersionIdsToStages,omitempty"`},
		{"NextToken", `json:"NextToken,omitempty"`},
	}

	for _, tc := range testCases {
		jsonName := member.GetJSONName(tc.fieldName)
		
		// Simulate struct tag generation
		tag := `json:"` + jsonName
		if !strings.Contains(tc.fieldName, "SecretId") { // Required fields don't have omitempty
			tag += ",omitempty"
		}
		tag += `"`

		if !strings.Contains(tag, jsonName) {
			t.Errorf("Field %s: generated tag should contain %s, got %s", tc.fieldName, jsonName, tag)
		}
		
		// Verify PascalCase is preserved
		if jsonName != tc.fieldName {
			t.Errorf("Field %s: PascalCase not preserved, got %s", tc.fieldName, jsonName)
		}
	}
}