package utils

import (
	"crypto/rand"
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