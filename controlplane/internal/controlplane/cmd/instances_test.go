package cmd_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/cmd"
)

var _ = Describe("Instances Command", func() {
	var tempDir string

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "kecs-instances-test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("ContainerConfig", func() {
		It("should load valid configuration", func() {
			configPath := filepath.Join(tempDir, "instances.yaml")
			configContent := `
defaultInstance: dev
instances:
  - name: dev
    image: ghcr.io/nandemo-ya/kecs:latest
    ports:
      api: 8080
      admin: 8081
    dataDir: /tmp/kecs/dev
    autoStart: true
  - name: test
    ports:
      api: 8090
      admin: 8091
`
			err := os.WriteFile(configPath, []byte(configContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			config, err := cmd.LoadContainerConfig(configPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(config).NotTo(BeNil())
			Expect(config.DefaultInstance).To(Equal("dev"))
			Expect(config.Instances).To(HaveLen(2))
			
			// Check first instance
			Expect(config.Instances[0].Name).To(Equal("dev"))
			Expect(config.Instances[0].Image).To(Equal("ghcr.io/nandemo-ya/kecs:latest"))
			Expect(config.Instances[0].Ports.API).To(Equal(8080))
			Expect(config.Instances[0].Ports.Admin).To(Equal(8081))
			Expect(config.Instances[0].DataDir).To(Equal("/tmp/kecs/dev"))
			Expect(config.Instances[0].AutoStart).To(BeTrue())

			// Check second instance with defaults
			Expect(config.Instances[1].Name).To(Equal("test"))
			Expect(config.Instances[1].Image).To(Equal("ghcr.io/nandemo-ya/kecs:latest")) // Default
			Expect(config.Instances[1].Ports.API).To(Equal(8090))
			Expect(config.Instances[1].Ports.Admin).To(Equal(8091))
		})

		It("should handle missing configuration file", func() {
			configPath := filepath.Join(tempDir, "nonexistent.yaml")
			_, err := cmd.LoadContainerConfig(configPath)
			Expect(err).To(HaveOccurred())
		})

		It("should save configuration", func() {
			configPath := filepath.Join(tempDir, "save-test.yaml")
			config := &cmd.InstancesConfig{
				DefaultInstance: "prod",
				Instances: []cmd.ContainerConfig{
					{
						Name: "prod",
						Image: "kecs:production",
						Ports: cmd.ContainerPortConfig{
							API:   9080,
							Admin: 9081,
						},
						DataDir: "/var/kecs/prod",
						AutoStart: true,
					},
				},
			}

			err := cmd.SaveContainerConfig(configPath, config)
			Expect(err).NotTo(HaveOccurred())

			// Verify saved file
			loadedConfig, err := cmd.LoadContainerConfig(configPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(loadedConfig.DefaultInstance).To(Equal("prod"))
			Expect(loadedConfig.Instances).To(HaveLen(1))
			Expect(loadedConfig.Instances[0].Name).To(Equal("prod"))
		})
	})

	Describe("Commands Registration", func() {
		It("should register instances command", func() {
			rootCmd := cmd.RootCmd
			instancesCmd := getSubcommand(rootCmd, "instances")
			
			Expect(instancesCmd).NotTo(BeNil())
			Expect(instancesCmd.Use).To(Equal("instances"))
			Expect(instancesCmd.Short).To(ContainSubstring("Manage multiple KECS instances"))
		})

		It("should register subcommands", func() {
			rootCmd := cmd.RootCmd
			instancesCmd := getSubcommand(rootCmd, "instances")
			
			Expect(instancesCmd).NotTo(BeNil())
			
			// Check subcommands
			listCmd := getSubcommand(instancesCmd, "list")
			Expect(listCmd).NotTo(BeNil())
			Expect(listCmd.Use).To(Equal("list"))
			
			startAllCmd := getSubcommand(instancesCmd, "start-all")
			Expect(startAllCmd).NotTo(BeNil())
			Expect(startAllCmd.Use).To(Equal("start-all"))
			
			stopAllCmd := getSubcommand(instancesCmd, "stop-all")
			Expect(stopAllCmd).NotTo(BeNil())
			Expect(stopAllCmd.Use).To(Equal("stop-all"))
		})

		It("should have config flag", func() {
			rootCmd := cmd.RootCmd
			instancesCmd := getSubcommand(rootCmd, "instances")
			
			Expect(instancesCmd).NotTo(BeNil())
			
			configFlag := instancesCmd.PersistentFlags().Lookup("config")
			Expect(configFlag).NotTo(BeNil())
			Expect(configFlag.Usage).To(ContainSubstring("Path to instances configuration file"))
		})
	})

	Describe("Multiple Instance Support", func() {
		It("should support auto-port flag in start command", func() {
			rootCmd := cmd.RootCmd
			startCmd := getSubcommand(rootCmd, "start")
			
			Expect(startCmd).NotTo(BeNil())
			
			autoPortFlag := startCmd.Flags().Lookup("auto-port")
			Expect(autoPortFlag).NotTo(BeNil())
			Expect(autoPortFlag.DefValue).To(Equal("false"))
			Expect(autoPortFlag.Usage).To(ContainSubstring("Automatically find available ports"))
		})

		It("should support config flag in start command", func() {
			rootCmd := cmd.RootCmd
			startCmd := getSubcommand(rootCmd, "start")
			
			Expect(startCmd).NotTo(BeNil())
			
			configFlag := startCmd.Flags().Lookup("config")
			Expect(configFlag).NotTo(BeNil())
			Expect(configFlag.Usage).To(ContainSubstring("Path to configuration file"))
		})
	})
})