package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateTaskID generates a task ID in the format used by AWS ECS
// Returns a 32-character hexadecimal string (e.g., "36374d1d33ad4ec0b5a9980f30402ead")
func GenerateTaskID() (string, error) {
	// Generate 16 random bytes
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hexadecimal string (32 characters)
	return hex.EncodeToString(bytes), nil
}

// GenerateTaskIDFromString generates a deterministic task ID from a string (e.g., pod name)
// This ensures that the same pod name always generates the same task ID
// Returns a 32-character hexadecimal string
func GenerateTaskIDFromString(input string) string {
	// Create SHA256 hash of the input
	hash := sha256.Sum256([]byte(input))

	// Take first 16 bytes and convert to hex (32 characters)
	return hex.EncodeToString(hash[:16])
}
