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

package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/startup"
)

// StartupState represents the startup flow state
type StartupState int

const (
	StartupStateChecking StartupState = iota
	StartupStateDialog
	StartupStateStarting
	StartupStateReady
)

// StartupFlow manages the KECS startup flow
type StartupFlow struct {
	state        StartupState
	endpoint     string
	instanceName string
	dialog       *startup.DialogModel
	logViewer    *startup.LogViewerModel
}

// NewStartupFlow creates a new startup flow
func NewStartupFlow(endpoint string) *StartupFlow {
	// Extract instance name from endpoint
	instanceName := extractInstanceName(endpoint)
	
	return &StartupFlow{
		state:        StartupStateChecking,
		endpoint:     endpoint,
		instanceName: instanceName,
	}
}

// CheckKECSAndInit checks if KECS is running and returns initialization commands
func CheckKECSAndInit(endpoint string) (bool, tea.Cmd) {
	// Check if KECS is running
	running, err := startup.CheckKECSStatus(endpoint)
	if err != nil {
		// Log error but treat as not running
		running = false
	}
	
	if running {
		// KECS is running, proceed normally
		return true, nil
	}
	
	// KECS is not running, return command to show dialog
	return false, func() tea.Msg {
		return showStartupDialogMsg{}
	}
}

// extractInstanceName extracts instance name from endpoint
func extractInstanceName(endpoint string) string {
	// Extract port from endpoint
	parts := strings.Split(endpoint, ":")
	if len(parts) < 3 {
		return "default"
	}
	
	portStr := strings.TrimSuffix(parts[2], "/")
	
	// Map port to instance name
	switch portStr {
	case "8080":
		return "dev"
	case "8090":
		return "staging"
	case "8100":
		return "test"
	case "8110":
		return "local"
	case "8200":
		return "prod"
	default:
		// For other ports, we can't determine the exact name
		return fmt.Sprintf("instance-%s", portStr)
	}
}

// Message types for startup flow

type showStartupDialogMsg struct{}

type startupDialogConfirmedMsg struct{}

type startupDialogCancelledMsg struct{}

type kecsStartupCompleteMsg struct{}

type kecsStartupFailedMsg struct {
	err error
}