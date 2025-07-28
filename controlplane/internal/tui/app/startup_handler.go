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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/startup"
)

// handleStartupFlow handles the startup flow state machine
func (a *App) handleStartupFlow(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		
		// Update startup components sizes
		if a.startupDialog != nil {
			a.startupDialog.SetSize(msg.Width, msg.Height)
		}
		if a.startupLogViewer != nil {
			a.startupLogViewer.SetSize(msg.Width, msg.Height)
		}
		
	case showStartupDialogMsg:
		// Create and show startup dialog
		a.startupDialog = startup.NewDialog(a.endpoint)
		a.startupDialog.SetSize(a.width, a.height)
		return a, a.startupDialog.Init()
		
	case tea.KeyMsg:
		// Handle key messages based on current state
		switch a.startupState {
		case StartupStateDialog:
			if a.startupDialog != nil {
				newDialog, cmd := a.startupDialog.Update(msg)
				a.startupDialog = newDialog
				
				// Check dialog result
				if a.startupDialog.IsConfirmed() {
					// User confirmed, start KECS
					a.startupState = StartupStateStarting
					a.startupLogViewer = startup.NewLogViewer(a.startupFlow.instanceName)
					a.startupLogViewer.SetSize(a.width, a.height)
					return a, a.startupLogViewer.Init()
				} else if a.startupDialog.IsCancelled() {
					// User cancelled, exit
					return a, tea.Quit
				}
				
				return a, cmd
			}
			
		case StartupStateStarting:
			if a.startupLogViewer != nil {
				newViewer, cmd := a.startupLogViewer.Update(msg)
				a.startupLogViewer = newViewer
				
				// Check if startup completed
				if a.startupLogViewer.IsCompleted() && msg.String() == "enter" {
					// Startup successful, transition to main TUI
					a.startupState = StartupStateReady
					return a, a.initializeMainTUI()
				} else if a.startupLogViewer.IsFailed() && msg.String() == "esc" {
					// Startup failed, exit
					return a, tea.Quit
				}
				
				return a, cmd
			}
		}
	}
	
	// Handle other messages during startup
	switch a.startupState {
	case StartupStateDialog:
		if a.startupDialog != nil {
			newDialog, cmd := a.startupDialog.Update(msg)
			a.startupDialog = newDialog
			
			// Check dialog result
			if a.startupDialog.IsConfirmed() {
				// User confirmed, start KECS
				a.startupState = StartupStateStarting
				a.startupLogViewer = startup.NewLogViewer(a.startupFlow.instanceName)
				a.startupLogViewer.SetSize(a.width, a.height)
				return a, a.startupLogViewer.Init()
			} else if a.startupDialog.IsCancelled() {
				// User cancelled, exit
				return a, tea.Quit
			}
			
			return a, cmd
		}
		
	case StartupStateStarting:
		if a.startupLogViewer != nil {
			newViewer, cmd := a.startupLogViewer.Update(msg)
			a.startupLogViewer = newViewer
			
			
			// Check if startup completed
			if a.startupLogViewer.IsCompleted() {
				// Don't automatically transition, wait for user to press enter
				return a, cmd
			} else if a.startupLogViewer.IsFailed() {
				// Startup failed, wait for user to press esc
				return a, cmd
			}
			
			return a, cmd
		}
	}
	
	return a, nil
}

// initializeMainTUI initializes the main TUI after successful startup
func (a *App) initializeMainTUI() tea.Cmd {
	// Reset startup components
	a.startupDialog = nil
	a.startupLogViewer = nil
	
	// Initialize main TUI components
	return tea.Batch(
		a.dashboard.Init(),
		a.clusterList.Init(),
		a.serviceList.Init(),
		a.taskList.Init(),
		a.taskDefList.Init(),
		a.instanceList.Init(),
	)
}