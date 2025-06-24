package cmd

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cmd Suite")
}

var _ = Describe("Server Command", func() {
	Describe("Data Directory Configuration", func() {
		var (
			originalDataDir string
			tempDir         string
		)

		BeforeEach(func() {
			// Save original env var
			originalDataDir = os.Getenv("KECS_DATA_DIR")
			
			// Create temp directory
			var err error
			tempDir, err = os.MkdirTemp("", "kecs-test-*")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			// Restore original env var
			if originalDataDir != "" {
				os.Setenv("KECS_DATA_DIR", originalDataDir)
			} else {
				os.Unsetenv("KECS_DATA_DIR")
			}
			
			// Clean up temp directory
			os.RemoveAll(tempDir)
		})

		It("should use KECS_DATA_DIR environment variable when set", func() {
			// Set environment variable
			testDataDir := filepath.Join(tempDir, "custom-data")
			os.Setenv("KECS_DATA_DIR", testDataDir)

			// Reset server command to pick up new env var
			serverCmd.ResetFlags()
			init()

			// Get the default value from the flag
			dataDirFlag := serverCmd.Flag("data-dir")
			Expect(dataDirFlag).ToNot(BeNil())
			Expect(dataDirFlag.DefValue).To(Equal(testDataDir))
		})

		It("should use home directory when KECS_DATA_DIR is not set", func() {
			// Unset environment variable
			os.Unsetenv("KECS_DATA_DIR")

			// Reset server command
			serverCmd.ResetFlags()
			init()

			// Get the default value from the flag
			dataDirFlag := serverCmd.Flag("data-dir")
			Expect(dataDirFlag).ToNot(BeNil())
			
			// Should contain .kecs/data
			Expect(dataDirFlag.DefValue).To(ContainSubstring(".kecs"))
			Expect(dataDirFlag.DefValue).To(ContainSubstring("data"))
		})

		It("should handle empty KECS_DATA_DIR", func() {
			// Set empty environment variable
			os.Setenv("KECS_DATA_DIR", "")

			// Reset server command
			serverCmd.ResetFlags()
			init()

			// Get the default value from the flag
			dataDirFlag := serverCmd.Flag("data-dir")
			Expect(dataDirFlag).ToNot(BeNil())
			
			// Should fall back to home directory
			Expect(dataDirFlag.DefValue).To(ContainSubstring(".kecs"))
			Expect(dataDirFlag.DefValue).To(ContainSubstring("data"))
		})
	})
})