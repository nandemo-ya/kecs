// Copyright 2025 The KECS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/nandemo-ya/kecs/controlplane/internal/tui"
	"github.com/spf13/cobra"
)

var (
	tuiEndpoint string
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the terminal user interface",
	Long: `Launch the KECS terminal user interface (TUI) for interactive management
of ECS resources.

The TUI provides a keyboard-driven interface for managing clusters, services,
tasks, and task definitions with real-time updates.`,
	Example: `  # Launch TUI connected to local KECS instance
  kecs tui

  # Connect to remote KECS instance
  kecs tui --endpoint http://remote-kecs:8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Note: endpoint parameter is not used in TUI v2 mock implementation
		// It will be used when we integrate with real backend
		
		// Run the new TUI implementation
		return tui.Run()
	},
}

func init() {
	RootCmd.AddCommand(tuiCmd)

	tuiCmd.Flags().StringVar(&tuiEndpoint, "endpoint", "", "KECS API endpoint (default: http://localhost:8080)")
}