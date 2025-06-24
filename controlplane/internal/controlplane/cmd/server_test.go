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
	Describe("getDefaultDataDir", func() {
		var originalDataDir string

		BeforeEach(func() {
			// Save original env var
			originalDataDir = os.Getenv("KECS_DATA_DIR")
		})

		AfterEach(func() {
			// Restore original env var
			if originalDataDir != "" {
				os.Setenv("KECS_DATA_DIR", originalDataDir)
			} else {
				os.Unsetenv("KECS_DATA_DIR")
			}
		})

		It("should use KECS_DATA_DIR when set", func() {
			testDir := "/custom/data/path"
			os.Setenv("KECS_DATA_DIR", testDir)

			result := getDefaultDataDir()
			Expect(result).To(Equal(testDir))
		})

		It("should use home directory when KECS_DATA_DIR is not set", func() {
			os.Unsetenv("KECS_DATA_DIR")

			result := getDefaultDataDir()
			
			// Should contain .kecs/data pattern
			if home, err := os.UserHomeDir(); err == nil {
				expectedPath := filepath.Join(home, ".kecs", "data")
				Expect(result).To(Equal(expectedPath))
			} else {
				Expect(result).To(Equal("~/.kecs/data"))
			}
		})

		It("should use home directory when KECS_DATA_DIR is empty", func() {
			os.Setenv("KECS_DATA_DIR", "")

			result := getDefaultDataDir()
			
			// Should contain .kecs/data pattern
			if home, err := os.UserHomeDir(); err == nil {
				expectedPath := filepath.Join(home, ".kecs", "data")
				Expect(result).To(Equal(expectedPath))
			} else {
				Expect(result).To(Equal("~/.kecs/data"))
			}
		})

		It("should handle absolute paths in KECS_DATA_DIR", func() {
			os.Setenv("KECS_DATA_DIR", "/absolute/path/to/data")

			result := getDefaultDataDir()
			Expect(result).To(Equal("/absolute/path/to/data"))
		})

		It("should handle relative paths in KECS_DATA_DIR", func() {
			os.Setenv("KECS_DATA_DIR", "./relative/data")

			result := getDefaultDataDir()
			Expect(result).To(Equal("./relative/data"))
		})
	})
})