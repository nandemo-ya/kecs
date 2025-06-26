package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsAcknowledged(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "kecs-security-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test 1: Not acknowledged (file doesn't exist)
	if IsAcknowledged(tmpDir) {
		t.Error("Expected IsAcknowledged to return false when file doesn't exist")
	}

	// Test 2: Create acknowledgment file
	ackFile := filepath.Join(tmpDir, ".security-acknowledged")
	if err := os.WriteFile(ackFile, []byte("acknowledged\n"), 0644); err != nil {
		t.Fatalf("Failed to create acknowledgment file: %v", err)
	}

	// Test 3: Should be acknowledged now
	if !IsAcknowledged(tmpDir) {
		t.Error("Expected IsAcknowledged to return true when file exists")
	}
}

func TestClearAcknowledgment(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "kecs-security-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create acknowledgment file
	ackFile := filepath.Join(tmpDir, ".security-acknowledged")
	if err := os.WriteFile(ackFile, []byte("acknowledged\n"), 0644); err != nil {
		t.Fatalf("Failed to create acknowledgment file: %v", err)
	}

	// Clear acknowledgment
	if err := ClearAcknowledgment(tmpDir); err != nil {
		t.Errorf("Failed to clear acknowledgment: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(ackFile); !os.IsNotExist(err) {
		t.Error("Expected acknowledgment file to be deleted")
	}

	// Test clearing when file doesn't exist (should not error)
	if err := ClearAcknowledgment(tmpDir); err != nil {
		t.Errorf("ClearAcknowledgment should not error when file doesn't exist: %v", err)
	}
}

func TestShowDisclaimerIfNeeded(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "kecs-security-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test 1: Skip disclaimer when explicitly disabled
	err = ShowDisclaimerIfNeeded(true, false, tmpDir)
	if err != nil {
		t.Errorf("Expected no error when skipDisclaimer is true, got: %v", err)
	}

	// Test 2: Skip disclaimer when already acknowledged via config
	err = ShowDisclaimerIfNeeded(false, true, tmpDir)
	if err != nil {
		t.Errorf("Expected no error when securityAcknowledged is true, got: %v", err)
	}

	// Test 3: Skip disclaimer when acknowledgment file exists
	ackFile := filepath.Join(tmpDir, ".security-acknowledged")
	if err := os.WriteFile(ackFile, []byte("acknowledged\n"), 0644); err != nil {
		t.Fatalf("Failed to create acknowledgment file: %v", err)
	}
	
	err = ShowDisclaimerIfNeeded(false, false, tmpDir)
	if err != nil {
		t.Errorf("Expected no error when acknowledgment file exists, got: %v", err)
	}
}