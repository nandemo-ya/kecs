package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information
var (
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildDate = "unknown"

	// versionCmd represents the version command
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  `Print the version, git commit, and build date of the KECS Control Plane.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("KECS Control Plane\n")
			fmt.Printf("Version:    %s\n", Version)
			fmt.Printf("Git commit: %s\n", GitCommit)
			fmt.Printf("Built:      %s\n", BuildDate)
		},
	}
)
