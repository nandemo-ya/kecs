//go:build nocli
// +build nocli

package cmd

import "github.com/spf13/cobra"

// serverCmd is a stub when building without server support (no DuckDB)
var serverCmd = &cobra.Command{
	Use:    "server",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
