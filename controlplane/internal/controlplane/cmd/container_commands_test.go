package cmd_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/cmd"
)

var _ = Describe("Container Commands", func() {
	Describe("Start Command", func() {
		It("should be registered in root command", func() {
			rootCmd := cmd.RootCmd
			startCmd := getSubcommand(rootCmd, "start")
			
			Expect(startCmd).NotTo(BeNil())
			Expect(startCmd.Use).To(Equal("start"))
			Expect(startCmd.Short).To(ContainSubstring("Start KECS server in a container"))
		})

		It("should have required flags", func() {
			rootCmd := cmd.RootCmd
			startCmd := getSubcommand(rootCmd, "start")
			
			Expect(startCmd).NotTo(BeNil())
			
			// Check flags
			nameFlag := startCmd.Flags().Lookup("name")
			Expect(nameFlag).NotTo(BeNil())
			Expect(nameFlag.DefValue).To(Equal("kecs-server"))
			
			imageFlag := startCmd.Flags().Lookup("image")
			Expect(imageFlag).NotTo(BeNil())
			Expect(imageFlag.DefValue).To(Equal("ghcr.io/nandemo-ya/kecs:latest"))
			
			apiPortFlag := startCmd.Flags().Lookup("api-port")
			Expect(apiPortFlag).NotTo(BeNil())
			Expect(apiPortFlag.DefValue).To(Equal("8080"))
			
			adminPortFlag := startCmd.Flags().Lookup("admin-port")
			Expect(adminPortFlag).NotTo(BeNil())
			Expect(adminPortFlag.DefValue).To(Equal("8081"))
			
			dataDirFlag := startCmd.Flags().Lookup("data-dir")
			Expect(dataDirFlag).NotTo(BeNil())
			
			detachFlag := startCmd.Flags().Lookup("detach")
			Expect(detachFlag).NotTo(BeNil())
			Expect(detachFlag.DefValue).To(Equal("true"))
			
			localBuildFlag := startCmd.Flags().Lookup("local-build")
			Expect(localBuildFlag).NotTo(BeNil())
			Expect(localBuildFlag.DefValue).To(Equal("false"))
		})
	})

	Describe("Stop Command", func() {
		It("should be registered in root command", func() {
			rootCmd := cmd.RootCmd
			stopCmd := getSubcommand(rootCmd, "stop")
			
			Expect(stopCmd).NotTo(BeNil())
			Expect(stopCmd.Use).To(Equal("stop"))
			Expect(stopCmd.Short).To(ContainSubstring("Stop KECS server container"))
		})

		It("should have required flags", func() {
			rootCmd := cmd.RootCmd
			stopCmd := getSubcommand(rootCmd, "stop")
			
			Expect(stopCmd).NotTo(BeNil())
			
			nameFlag := stopCmd.Flags().Lookup("name")
			Expect(nameFlag).NotTo(BeNil())
			Expect(nameFlag.DefValue).To(Equal("kecs-server"))
			
			forceFlag := stopCmd.Flags().Lookup("force")
			Expect(forceFlag).NotTo(BeNil())
			Expect(forceFlag.DefValue).To(Equal("false"))
		})
	})

	Describe("Status Command", func() {
		It("should be registered in root command", func() {
			rootCmd := cmd.RootCmd
			statusCmd := getSubcommand(rootCmd, "status")
			
			Expect(statusCmd).NotTo(BeNil())
			Expect(statusCmd.Use).To(Equal("status"))
			Expect(statusCmd.Short).To(ContainSubstring("Show KECS server container status"))
		})

		It("should have required flags", func() {
			rootCmd := cmd.RootCmd
			statusCmd := getSubcommand(rootCmd, "status")
			
			Expect(statusCmd).NotTo(BeNil())
			
			nameFlag := statusCmd.Flags().Lookup("name")
			Expect(nameFlag).NotTo(BeNil())
			Expect(nameFlag.DefValue).To(Equal(""))
			
			allFlag := statusCmd.Flags().Lookup("all")
			Expect(allFlag).NotTo(BeNil())
			Expect(allFlag.DefValue).To(Equal("false"))
		})
	})

	Describe("Logs Command", func() {
		It("should be registered in root command", func() {
			rootCmd := cmd.RootCmd
			logsCmd := getSubcommand(rootCmd, "logs")
			
			Expect(logsCmd).NotTo(BeNil())
			Expect(logsCmd.Use).To(Equal("logs"))
			Expect(logsCmd.Short).To(ContainSubstring("Show logs from KECS server container"))
		})

		It("should have required flags", func() {
			rootCmd := cmd.RootCmd
			logsCmd := getSubcommand(rootCmd, "logs")
			
			Expect(logsCmd).NotTo(BeNil())
			
			nameFlag := logsCmd.Flags().Lookup("name")
			Expect(nameFlag).NotTo(BeNil())
			Expect(nameFlag.DefValue).To(Equal("kecs-server"))
			
			followFlag := logsCmd.Flags().Lookup("follow")
			Expect(followFlag).NotTo(BeNil())
			Expect(followFlag.DefValue).To(Equal("false"))
			
			tailFlag := logsCmd.Flags().Lookup("tail")
			Expect(tailFlag).NotTo(BeNil())
			Expect(tailFlag.DefValue).To(Equal("all"))
			
			timestampsFlag := logsCmd.Flags().Lookup("timestamps")
			Expect(timestampsFlag).NotTo(BeNil())
			Expect(timestampsFlag.DefValue).To(Equal("false"))
		})
	})

	Context("Integration Tests", func() {
		It("should not conflict with existing commands", func() {
			// Test that help works with new commands
			rootCmd := cmd.RootCmd
			rootCmd.SetArgs([]string{"--help"})
			
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			
			err := rootCmd.Execute()
			w.Close()
			os.Stdout = oldStdout
			
			Expect(err).NotTo(HaveOccurred())
			
			// Read output
			buf := make([]byte, 4096)
			n, _ := r.Read(buf)
			output := string(buf[:n])
			
			// Verify all commands are present
			Expect(output).To(ContainSubstring("start"))
			Expect(output).To(ContainSubstring("stop"))
			Expect(output).To(ContainSubstring("status"))
			Expect(output).To(ContainSubstring("logs"))
			Expect(output).To(ContainSubstring("server"))
		})
	})
})

// Helper function to get subcommand by name
func getSubcommand(rootCmd *cobra.Command, name string) *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}