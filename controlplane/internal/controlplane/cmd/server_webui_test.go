package cmd_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/cmd"
)

var _ = Describe("Server Command Web UI Options", func() {
	BeforeEach(func() {
		config.ResetConfig()
	})

	AfterEach(func() {
		os.Unsetenv("KECS_WEBUI_ENABLED")
		config.ResetConfig()
	})

	Describe("--no-webui flag", func() {
		It("should be registered in server command", func() {
			rootCmd := cmd.RootCmd
			serverCmd := getSubcommand(rootCmd, "server")
			
			Expect(serverCmd).NotTo(BeNil())
			
			noWebUIFlag := serverCmd.Flags().Lookup("no-webui")
			Expect(noWebUIFlag).NotTo(BeNil())
			Expect(noWebUIFlag.Usage).To(Equal("Disable Web UI"))
			Expect(noWebUIFlag.DefValue).To(Equal("false"))
		})
	})

	Describe("Web UI configuration", func() {
		It("should respect KECS_WEBUI_ENABLED environment variable", func() {
			os.Setenv("KECS_WEBUI_ENABLED", "false")
			err := config.InitConfig()
			Expect(err).NotTo(HaveOccurred())
			
			Expect(config.GetBool("ui.enabled")).To(BeFalse())
		})

		It("should default to enabled when environment variable is not set", func() {
			err := config.InitConfig()
			Expect(err).NotTo(HaveOccurred())
			
			Expect(config.GetBool("ui.enabled")).To(BeTrue())
		})

		It("should handle 'true' value in environment variable", func() {
			os.Setenv("KECS_WEBUI_ENABLED", "true")
			err := config.InitConfig()
			Expect(err).NotTo(HaveOccurred())
			
			Expect(config.GetBool("ui.enabled")).To(BeTrue())
		})
	})
})