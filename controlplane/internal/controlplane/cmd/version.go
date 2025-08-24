package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/version"
)

var (
	jsonOutput bool

	// versionCmd represents the version command
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  `Print the version, git commit, and build date of the KECS Control Plane.`,
		Run: func(cmd *cobra.Command, args []string) {
			info := version.GetInfo()

			if jsonOutput {
				output, _ := json.MarshalIndent(info, "", "  ")
				fmt.Println(string(output))
			} else {
				fmt.Printf("KECS Control Plane\n")
				fmt.Printf("Version:    %s\n", info.Version)
				fmt.Printf("Git commit: %s\n", info.GitCommit)
				fmt.Printf("Built:      %s\n", info.BuildDate)
				fmt.Printf("Go version: %s\n", info.GoVersion)
			}
		},
	}
)

func init() {
	versionCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output version information in JSON format")
}
