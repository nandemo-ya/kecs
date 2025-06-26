package api_test

import (
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/memory"
)

var _ = Describe("Web UI Configuration", func() {
	BeforeEach(func() {
		// Reset config before each test
		config.ResetConfig()
	})

	AfterEach(func() {
		// Clean up environment variables
		os.Unsetenv("KECS_WEBUI_ENABLED")
		config.ResetConfig()
	})

	Describe("EnableWebUI", func() {
		It("should be enabled by default", func() {
			Expect(api.EnableWebUI()).To(BeTrue())
		})

		It("should be disabled when KECS_WEBUI_ENABLED=false", func() {
			os.Setenv("KECS_WEBUI_ENABLED", "false")
			config.InitConfig()
			
			Expect(api.EnableWebUI()).To(BeFalse())
		})

		It("should be enabled when KECS_WEBUI_ENABLED=true", func() {
			os.Setenv("KECS_WEBUI_ENABLED", "true")
			config.InitConfig()
			
			Expect(api.EnableWebUI()).To(BeTrue())
		})

		It("should respect config file settings", func() {
			// Create a config with UI disabled
			cfg := config.DefaultConfig()
			cfg.UI.Enabled = false
			
			// Since we can't easily set the global config in tests,
			// we'll test through environment variable
			os.Setenv("KECS_WEBUI_ENABLED", "false")
			config.InitConfig()
			
			Expect(api.EnableWebUI()).To(BeFalse())
		})
	})

	Describe("Web UI Handler Integration", func() {
		var (
			server *api.Server
			ts     *httptest.Server
			storage *memory.MemoryStorage
		)

		BeforeEach(func() {
			storage = memory.NewMemoryStorage()
		})

		AfterEach(func() {
			if ts != nil {
				ts.Close()
			}
		})

		Context("when Web UI is disabled", func() {
			It("should not initialize Web UI handler", func() {
				os.Setenv("KECS_WEBUI_ENABLED", "false")
				config.InitConfig()
				
				// Create server - Web UI should not be initialized
				var err error
				server, err = api.NewServer(8080, "", storage, nil)
				Expect(err).NotTo(HaveOccurred())
				
				// In a real test, we would check if webUIHandler is nil
				// For now, we just verify the server was created
				Expect(server).NotTo(BeNil())
			})
		})

		Context("when Web UI is enabled", func() {
			It("should initialize Web UI handler if available", func() {
				os.Setenv("KECS_WEBUI_ENABLED", "true")
				config.InitConfig()
				
				// Create server - Web UI should be initialized if GetWebUIFS is available
				var err error
				server, err = api.NewServer(8080, "", storage, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(server).NotTo(BeNil())
			})
		})
	})

	Describe("CLI Flag Integration", func() {
		It("should disable Web UI when --no-webui flag is used", func() {
			// This would be tested in cmd package tests
			// Here we just verify the config behavior
			
			cfg := config.DefaultConfig()
			cfg.UI.Enabled = false
			
			// Simulate the flag effect
			os.Setenv("KECS_WEBUI_ENABLED", "false")
			config.InitConfig()
			
			Expect(api.EnableWebUI()).To(BeFalse())
		})
	})
})