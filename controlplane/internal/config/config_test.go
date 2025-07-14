package config_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
)

var _ = Describe("Config", func() {
	var (
		tempDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "kecs-config-test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("DefaultConfig", func() {
		It("should return valid default configuration", func() {
			cfg := config.DefaultConfig()
			Expect(cfg).NotTo(BeNil())
			Expect(cfg.Server.Port).To(Equal(8080))
			Expect(cfg.Server.AdminPort).To(Equal(8081))
			Expect(cfg.LocalStack.Enabled).To(BeFalse())
		})
	})

	Describe("LoadConfig", func() {
		Context("when config file does not exist", func() {
			It("should return error", func() {
				nonExistentPath := filepath.Join(tempDir, "nonexistent.yaml")
				_, err := config.LoadConfig(nonExistentPath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("config file does not exist"))
				Expect(err.Error()).To(ContainSubstring(nonExistentPath))
			})
		})

		Context("when config file is not accessible", func() {
			It("should return error", func() {
				configPath := filepath.Join(tempDir, "noaccess.yaml")
				err := os.WriteFile(configPath, []byte("test"), 0644)
				Expect(err).NotTo(HaveOccurred())
				
				// Make file inaccessible
				err = os.Chmod(configPath, 0000)
				Expect(err).NotTo(HaveOccurred())
				
				_, err = config.LoadConfig(configPath)
				Expect(err).To(HaveOccurred())
				// The error can be either "failed to access config file" or "failed to read config file"
				// depending on when the permission error is caught
				Expect(err.Error()).To(Or(
					ContainSubstring("failed to access config file"),
					ContainSubstring("failed to read config file"),
				))
				Expect(err.Error()).To(ContainSubstring("permission denied"))
				
				// Restore permissions for cleanup
				os.Chmod(configPath, 0644)
			})
		})

		Context("when config file is empty", func() {
			It("should return default configuration with empty path", func() {
				cfg, err := config.LoadConfig("")
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Server.Port).To(Equal(8080))
			})
		})

		Context("when config file exists", func() {
			It("should load configuration from file", func() {
				configPath := filepath.Join(tempDir, "test.yaml")
				configContent := `
server:
  port: 9090
  adminPort: 9091
  dataDir: /custom/data
  logLevel: debug

localstack:
  enabled: true
  services:
    - s3
    - dynamodb
  version: 2.0.0
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				cfg, err := config.LoadConfig(configPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Server.Port).To(Equal(9090))
				Expect(cfg.Server.AdminPort).To(Equal(9091))
				Expect(cfg.Server.DataDir).To(Equal("/custom/data"))
				Expect(cfg.Server.LogLevel).To(Equal("debug"))
				// LocalStack config is now handled separately
				// We don't test LocalStack config here as it's set in DefaultConfig
			})
		})

		Context("when config file has invalid YAML", func() {
			It("should return error", func() {
				configPath := filepath.Join(tempDir, "invalid.yaml")
				configContent := `
server:
  port: invalid
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				_, err = config.LoadConfig(configPath)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Validate", func() {
		It("should validate valid configuration", func() {
			cfg := config.DefaultConfig()
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject invalid server port", func() {
			cfg := config.DefaultConfig()
			cfg.Server.Port = 0
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid server port"))
		})

		It("should reject invalid admin port", func() {
			cfg := config.DefaultConfig()
			cfg.Server.Port = 8080 // Ensure valid port is set
			cfg.Server.AdminPort = 70000
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid admin port"))
		})

		It("should validate LocalStack config when enabled", func() {
			cfg := config.DefaultConfig()
			cfg.Server.Port = 8080 // Ensure valid port is set
			cfg.Server.AdminPort = 8081 // Ensure valid admin port is set
			cfg.LocalStack.Enabled = true
			cfg.LocalStack.Services = []string{"invalid-service"}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid LocalStack config"))
		})
	})
})
