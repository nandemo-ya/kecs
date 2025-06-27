package security

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const disclaimerText = `
╔════════════════════════════════════════════════════════════════════════════╗
║                      SECURITY NOTICE - PLEASE READ                         ║
╠════════════════════════════════════════════════════════════════════════════╣
║                                                                            ║
║  KECS requires access to the Docker daemon to create and manage local      ║
║  Kubernetes clusters. This provides significant capabilities:              ║
║                                                                            ║
║  • Full access to Docker daemon (equivalent to root access)               ║
║  • Ability to create, modify, and delete containers                        ║
║  • Access to the host filesystem through volume mounts                     ║
║  • Network configuration capabilities                                      ║
║                                                                            ║
║  SUPPORTED ENVIRONMENTS:                                                   ║
║  ✓ Local development machines                                              ║
║  ✓ CI/CD pipelines (GitHub Actions, GitLab CI, Jenkins, etc.)            ║
║  ✓ Isolated test environments                                              ║
║                                                                            ║
║  NOT SUPPORTED/UNSAFE:                                                     ║
║  ✗ Production environments                                                 ║
║  ✗ Public-facing deployments                                              ║
║  ✗ Multi-tenant systems                                                    ║
║  ✗ Environments with untrusted users                                      ║
║                                                                            ║
║  By continuing, you acknowledge that:                                      ║
║  1. You understand the security implications                               ║
║  2. You trust KECS with Docker daemon access                               ║
║  3. You will only use KECS in supported environments                       ║
║                                                                            ║
║  To skip this notice in the future, you can:                              ║
║  • Set KECS_SECURITY_ACKNOWLEDGED=true                                     ║
║  • Add 'features.securityAcknowledged: true' to your config file          ║
║                                                                            ║
╚════════════════════════════════════════════════════════════════════════════╝
`

// ShowDisclaimerIfNeeded shows the security disclaimer if not already acknowledged
func ShowDisclaimerIfNeeded(skipDisclaimer, securityAcknowledged bool, dataDir string) error {
	// Skip if explicitly disabled or already acknowledged
	if skipDisclaimer || securityAcknowledged {
		return nil
	}

	// Check if acknowledgment file exists
	ackFile := filepath.Join(dataDir, ".security-acknowledged")
	if _, err := os.Stat(ackFile); err == nil {
		// File exists, disclaimer was previously acknowledged
		return nil
	}

	// Display disclaimer
	fmt.Print(disclaimerText)
	fmt.Println()

	// Ask for acknowledgment
	if !promptForAcknowledgment() {
		return fmt.Errorf("security disclaimer not acknowledged")
	}

	// Create acknowledgment file
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := os.WriteFile(ackFile, []byte("acknowledged\n"), 0644); err != nil {
		return fmt.Errorf("failed to write acknowledgment file: %w", err)
	}

	fmt.Println("\n✓ Security disclaimer acknowledged. This message will not appear again.")
	fmt.Println()

	return nil
}

// promptForAcknowledgment prompts the user to acknowledge the security disclaimer
func promptForAcknowledgment() bool {
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Print("Do you acknowledge and accept these terms? (yes/no): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "yes" || response == "y"
}

// IsAcknowledged checks if the security disclaimer has been acknowledged
func IsAcknowledged(dataDir string) bool {
	ackFile := filepath.Join(dataDir, ".security-acknowledged")
	_, err := os.Stat(ackFile)
	return err == nil
}

// ClearAcknowledgment removes the acknowledgment file (for testing)
func ClearAcknowledgment(dataDir string) error {
	ackFile := filepath.Join(dataDir, ".security-acknowledged")
	err := os.Remove(ackFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}