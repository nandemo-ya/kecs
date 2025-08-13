package config_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
)

var _ = Describe("Config Recovery Defaults", func() {
	BeforeEach(func() {
		// Reset config before each test
		config.ResetConfig()
	})

	Context("when getting default configuration", func() {
		It("should have autoRecoverState enabled by default", func() {
			// Initialize config without any environment variables
			err := config.InitConfig()
			Expect(err).To(BeNil())

			// Get the default configuration
			cfg := config.DefaultConfig()
			Expect(cfg).NotTo(BeNil())

			// Verify autoRecoverState is true by default
			Expect(cfg.Features.AutoRecoverState).To(BeTrue(),
				"autoRecoverState should be enabled by default to ensure k3d clusters are recreated after KECS restart")
		})

		It("should respect KECS_AUTO_RECOVER_STATE environment variable", func() {
			// Test with false value
			os.Setenv("KECS_AUTO_RECOVER_STATE", "false")
			defer os.Unsetenv("KECS_AUTO_RECOVER_STATE")

			// Re-initialize config to pick up env var
			config.ResetConfig()
			config.InitConfig()

			cfg := config.DefaultConfig()
			Expect(cfg.Features.AutoRecoverState).To(BeFalse(),
				"autoRecoverState should be false when KECS_AUTO_RECOVER_STATE=false")
		})
	})
})
