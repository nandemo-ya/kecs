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

package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// executeAction handles both global and view-specific actions
func (m Model) executeAction(action KeyAction) (Model, tea.Cmd) {
	// For certain views, handle MoveUp/MoveDown specially
	if m.currentView == ViewTaskDescribe && (action == ActionMoveUp || action == ActionMoveDown) {
		return m.executeTaskDescribeAction(action)
	}

	// Check if it's a global action first
	// Global actions are: Quit, Back, Help, Search, Command, MoveUp, MoveDown, Refresh, GoHome, SwitchInstance, DeleteInstance
	switch action {
	case ActionQuit, ActionBack, ActionHelp, ActionSearch, ActionCommand,
		ActionMoveUp, ActionMoveDown, ActionRefresh, ActionGoHome, ActionSwitchInstance, ActionDeleteInstance:
		return m.executeGlobalAction(action)
	}

	// Handle view-specific actions
	switch m.currentView {
	case ViewInstances:
		return m.executeInstanceAction(action)
	case ViewClusters:
		return m.executeClusterAction(action)
	case ViewServices:
		return m.executeServiceAction(action)
	case ViewTasks:
		return m.executeTaskAction(action)
	case ViewTaskDescribe:
		return m.executeTaskDescribeAction(action)
	case ViewLogs:
		return m.executeLogAction(action)
	case ViewTaskDefinitionFamilies:
		return m.executeTaskDefFamiliesAction(action)
	case ViewTaskDefinitionRevisions:
		return m.executeTaskDefRevisionsAction(action)
	}

	return m, nil
}

// executeGlobalAction handles global key actions that work across all views
// Returns the updated model and optional command
func (m Model) executeGlobalAction(action KeyAction) (Model, tea.Cmd) {
	switch action {
	case ActionQuit:
		if !m.searchMode && !m.commandMode && m.currentView != ViewTaskDefinitionEditor {
			return m, tea.Quit
		}

	case ActionBack:
		if m.searchMode {
			m.searchMode = false
			m.searchQuery = ""
		} else if m.commandMode {
			m.commandMode = false
			m.commandInput = ""
		} else if m.showHelp {
			m.showHelp = false
		} else if m.currentView == ViewCommandPalette {
			m.currentView = m.previousView
			m.commandPalette.Reset()
		} else if m.currentView == ViewTaskDescribe {
			m.currentView = m.previousView
		} else if m.currentView == ViewTaskDefinitionRevisions && m.showTaskDefJSON {
			m.showTaskDefJSON = false
		} else if m.currentView != ViewInstances {
			m.goBack()
			return m, m.loadDataFromAPI()
		}

	case ActionHelp:
		if !m.searchMode && !m.commandMode {
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.previousView = m.currentView
				m.currentView = ViewHelp
			} else {
				m.currentView = m.previousView
			}
		}

	case ActionSearch:
		if !m.searchMode && !m.commandMode {
			m.searchMode = true
			m.searchQuery = ""
		}

	case ActionCommand:
		if !m.searchMode && !m.commandMode {
			m.commandMode = true
			m.commandInput = ""
		}

	case ActionMoveUp:
		m.moveCursorUp()

	case ActionMoveDown:
		m.moveCursorDown()

	case ActionRefresh:
		return m, m.loadDataFromAPI()

	case ActionGoHome:
		// Navigate to home (Instances view)
		if m.currentView != ViewInstances {
			m.currentView = ViewInstances
			m.selectedInstance = ""
			m.selectedCluster = ""
			m.selectedService = ""
			m.selectedTask = ""
			m.instanceCursor = 0
			return m, m.loadDataFromAPI()
		}

	case ActionSwitchInstance:
		if len(m.instances) > 1 {
			m.instanceSwitcher = NewInstanceSwitcher(m.instances)
			m.previousView = m.currentView
			m.currentView = ViewInstanceSwitcher
		}

	case ActionDeleteInstance:
		// Delete currently selected instance
		if m.selectedInstance != "" {
			if m.selectedInstance == "default" {
				m.err = fmt.Errorf("Cannot delete default instance")
				return m, nil
			}
			if m.isDeleting {
				return m, nil
			}

			m.confirmDialog = DeleteInstanceDialog(
				m.selectedInstance,
				func() error { return nil },
				func() {},
			)
			m.pendingCommand = m.deleteInstanceCmd(m.selectedInstance)
			m.previousView = m.currentView
			m.currentView = ViewConfirmDialog
		}
	}

	return m, nil
}

// executeInstanceAction handles actions specific to the Instances view
func (m Model) executeInstanceAction(action KeyAction) (Model, tea.Cmd) {
	switch action {
	case ActionSelect:
		if len(m.instances) > 0 {
			m.selectedInstance = m.instances[m.instanceCursor].Name
			m.currentView = ViewClusters
			m.clusterCursor = 0
			return m, m.loadDataFromAPI()
		}

	case ActionNewInstance:
		if m.instanceForm == nil {
			m.instanceForm = NewInstanceFormWithSuggestions(m.instances)
		} else {
			m.instanceForm = NewInstanceFormWithSuggestions(m.instances)
		}
		m.previousView = m.currentView
		m.currentView = ViewInstanceCreate

	case ActionToggleInstance:
		if len(m.instances) > 0 && m.instanceCursor < len(m.instances) {
			instanceName := m.instances[m.instanceCursor].Name
			instanceStatus := strings.ToLower(m.instances[m.instanceCursor].Status)

			if instanceStatus == "stopped" {
				m.confirmDialog = StartInstanceDialog(
					instanceName,
					func() error {
						m.instances[m.instanceCursor].Status = "Starting"
						return nil
					},
					func() {},
				)
				m.pendingCommand = m.startInstanceCmd(instanceName)
			} else if instanceStatus == "running" {
				m.confirmDialog = StopInstanceDialog(
					instanceName,
					func() error {
						m.instances[m.instanceCursor].Status = "Stopping"
						return nil
					},
					func() {},
				)
				m.pendingCommand = m.stopInstanceCmd(instanceName)
			} else {
				m.err = fmt.Errorf("Cannot start/stop instance in %s state", instanceStatus)
				return m, nil
			}
			m.previousView = m.currentView
			m.currentView = ViewConfirmDialog
		}

	case ActionNavigateTaskDefs:
		if m.selectedInstance != "" {
			m.currentView = ViewTaskDefinitionFamilies
			m.taskDefFamilyCursor = 0
			return m, m.loadTaskDefinitionFamiliesCmd()
		}

	case ActionYank:
		if len(m.instances) > 0 && m.instanceCursor < len(m.instances) {
			inst := m.instances[m.instanceCursor]
			err := copyToClipboard(inst.Name)
			if err == nil {
				m.clipboardMsg = fmt.Sprintf("Copied: %s", inst.Name)
			} else {
				m.clipboardMsg = fmt.Sprintf("Copy failed: %v", err)
			}
			m.clipboardMsgTime = time.Now()
		}
	}

	return m, nil
}

// executeClusterAction handles actions specific to the Clusters view
func (m Model) executeClusterAction(action KeyAction) (Model, tea.Cmd) {
	switch action {
	case ActionSelect:
		if len(m.clusters) > 0 {
			m.selectedCluster = m.clusters[m.clusterCursor].Name
			m.currentView = ViewServices
			m.serviceCursor = 0
			return m, m.loadDataFromAPI()
		}

	case ActionCreateCluster:
		if m.selectedInstance != "" {
			m.clusterForm = NewClusterForm()
			m.previousView = m.currentView
			m.currentView = ViewClusterCreate
		}

	case ActionNavigateServices:
		if m.selectedCluster != "" {
			m.currentView = ViewServices
		}

	case ActionNavigateTaskDefs:
		if m.selectedInstance != "" {
			m.currentView = ViewTaskDefinitionFamilies
			m.taskDefFamilyCursor = 0
			return m, m.loadTaskDefinitionFamiliesCmd()
		}

	case ActionNavigateAllTasks:
		if m.selectedInstance != "" && len(m.clusters) > 0 {
			if m.selectedCluster == "" && m.clusterCursor < len(m.clusters) {
				m.selectedCluster = m.clusters[m.clusterCursor].Name
			}
			if m.selectedCluster != "" {
				m.currentView = ViewTasks
				m.taskCursor = 0
				m.selectedService = ""
				return m, m.loadDataFromAPI()
			}
		}

	case ActionNavigateLoadBalancers:
		if m.selectedInstance != "" {
			m.currentView = ViewLoadBalancers
			m.lbCursor = 0
			return m, m.loadELBv2DataCmd()
		}

	case ActionNavigateTargetGroups:
		if m.selectedInstance != "" {
			m.currentView = ViewTargetGroups
			m.tgCursor = 0
			return m, m.loadELBv2DataCmd()
		}
	}

	return m, nil
}

// executeServiceAction handles actions specific to the Services view
func (m Model) executeServiceAction(action KeyAction) (Model, tea.Cmd) {
	switch action {
	case ActionSelect:
		if len(m.services) > 0 {
			m.selectedService = m.services[m.serviceCursor].Name
			m.currentView = ViewTasks
			m.taskCursor = 0
			return m, m.loadDataFromAPI()
		}

	case ActionScaleService:
		if len(m.services) > 0 && m.serviceCursor < len(m.services) {
			service := m.services[m.serviceCursor]
			m.serviceScaleDialog = NewServiceScaleDialog(service.Name, service.Desired)
		}

	case ActionUpdateService:
		if len(m.services) > 0 && m.serviceCursor < len(m.services) {
			service := m.services[m.serviceCursor]
			return m, m.fetchTaskDefinitionsForUpdate(service.Name, service.TaskDef)
		}

	case ActionViewLogs:
		if len(m.services) > 0 {
			m.previousView = m.currentView
			m.currentView = ViewLogs
			return m, m.loadDataFromAPI()
		}

	case ActionNavigateClusters:
		m.currentView = ViewClusters
		m.selectedCluster = ""

	case ActionNavigateTaskDefs:
		if m.selectedInstance != "" {
			m.currentView = ViewTaskDefinitionFamilies
			m.taskDefFamilyCursor = 0
			return m, m.loadTaskDefinitionFamiliesCmd()
		}
	}

	return m, nil
}

// executeTaskAction handles actions specific to the Tasks view
func (m Model) executeTaskAction(action KeyAction) (Model, tea.Cmd) {
	switch action {
	case ActionDescribeTask:
		if len(m.tasks) > 0 && m.taskCursor < len(m.tasks) {
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("executeTaskAction", "Describe task: %s", m.tasks[m.taskCursor].ID)
			}
			m.selectedTask = m.tasks[m.taskCursor].ID
			m.previousView = m.currentView
			m.currentView = ViewTaskDescribe
			m.selectedTaskDetail = nil
			m.taskDescribeScroll = 0
			return m, m.loadTaskDetailsCmd()
		}

	case ActionStopTask:
		if len(m.tasks) > 0 && m.taskCursor < len(m.tasks) {
			task := m.tasks[m.taskCursor]
			m.confirmDialog = StopTaskDialog(
				task.ID,
				func() error { return nil },
				func() {},
			)
			m.pendingCommand = m.stopTaskCmd(task.ARN)
			m.previousView = m.currentView
			m.currentView = ViewConfirmDialog
		}

	case ActionViewLogs:
		if len(m.tasks) > 0 && m.taskCursor < len(m.tasks) {
			task := m.tasks[m.taskCursor]
			m.selectedTask = task.ID
			m.previousView = m.currentView

			containerName := ""
			if len(task.Containers) > 0 {
				containerName = task.Containers[0]
			}

			return m, m.viewTaskLogsCmd(task.ARN, containerName)
		}

	case ActionNavigateClusters:
		m.currentView = ViewClusters
		m.selectedCluster = ""

	case ActionNavigateTaskDefs:
		if m.selectedInstance != "" {
			m.currentView = ViewTaskDefinitionFamilies
			m.taskDefFamilyCursor = 0
			return m, m.loadTaskDefinitionFamiliesCmd()
		}
	}

	// Handle back navigation differently for tasks view
	if action == ActionBack {
		if m.selectedService != "" {
			m.currentView = ViewServices
		} else {
			m.currentView = ViewClusters
		}
		m.selectedTask = ""
	}

	return m, nil
}

// executeTaskDescribeAction handles actions specific to the Task Describe view
func (m Model) executeTaskDescribeAction(action KeyAction) (Model, tea.Cmd) {
	maxScroll := 100
	if m.selectedTaskDetail != nil {
		estimatedLines := 30
		estimatedLines += len(m.selectedTaskDetail.Containers) * 10
		maxScroll = estimatedLines
	}

	switch action {
	case ActionMoveUp:
		// Move container selection up
		if m.selectedTaskDetail != nil && len(m.selectedTaskDetail.Containers) > 0 {
			if m.selectedContainer > 0 {
				m.selectedContainer--
			}
		}

	case ActionMoveDown:
		// Move container selection down
		if m.selectedTaskDetail != nil && len(m.selectedTaskDetail.Containers) > 0 {
			if m.selectedContainer < len(m.selectedTaskDetail.Containers)-1 {
				m.selectedContainer++
			}
		}

	case ActionPageUp:
		m.taskDescribeScroll -= 10
		if m.taskDescribeScroll < 0 {
			m.taskDescribeScroll = 0
		}

	case ActionPageDown:
		m.taskDescribeScroll += 10
		if m.taskDescribeScroll > maxScroll {
			m.taskDescribeScroll = maxScroll
		}

	case ActionHome:
		m.taskDescribeScroll = 0

	case ActionEnd:
		m.taskDescribeScroll = maxScroll

	case ActionViewLogs:
		if m.selectedTaskDetail != nil {
			m.previousView = m.currentView
			containerName := ""
			if len(m.selectedTaskDetail.Containers) > 0 && m.selectedContainer < len(m.selectedTaskDetail.Containers) {
				containerName = m.selectedTaskDetail.Containers[m.selectedContainer].Name
			}
			return m, m.viewTaskLogsCmd(m.selectedTaskDetail.TaskARN, containerName)
		}

	case ActionRestartTask:
		// TODO: Implement restart task

	case ActionStopTask:
		// TODO: Implement stop task from describe view

	case ActionBack:
		m.currentView = m.previousView
		m.selectedTaskDetail = nil
		m.taskDescribeScroll = 0
	}

	return m, nil
}

// executeLogAction handles actions specific to the Logs view
func (m Model) executeLogAction(action KeyAction) (Model, tea.Cmd) {
	switch action {
	case ActionToggleSplitView:
		m.logSplitView = !m.logSplitView

	case ActionSaveLogs:
		// TODO: Implement save logs functionality
	}

	return m, nil
}

// executeTaskDefFamiliesAction handles actions specific to Task Definition Families view
func (m Model) executeTaskDefFamiliesAction(action KeyAction) (Model, tea.Cmd) {
	switch action {
	case ActionSelect:
		if len(m.taskDefFamilies) > 0 && m.taskDefFamilyCursor < len(m.taskDefFamilies) {
			m.selectedFamily = m.taskDefFamilies[m.taskDefFamilyCursor].Family
			m.currentView = ViewTaskDefinitionRevisions
			m.taskDefRevisionCursor = 0
			return m, m.loadTaskDefinitionRevisionsCmd()
		}

	case ActionNewTaskDef:
		// TODO: Implement new task definition

	case ActionCopyTaskDef:
		// TODO: Implement copy latest task definition
	}

	return m, nil
}

// executeTaskDefRevisionsAction handles actions specific to Task Definition Revisions view
func (m Model) executeTaskDefRevisionsAction(action KeyAction) (Model, tea.Cmd) {
	switch action {
	case ActionToggleJSON:
		if len(m.taskDefRevisions) > 0 && m.taskDefRevisionCursor < len(m.taskDefRevisions) {
			m.showTaskDefJSON = !m.showTaskDefJSON
			if m.showTaskDefJSON {
				rev := m.taskDefRevisions[m.taskDefRevisionCursor]
				if _, cached := m.taskDefJSONCache[rev.Revision]; !cached {
					taskDefArn := fmt.Sprintf("%s:%d", rev.Family, rev.Revision)
					return m, m.loadTaskDefinitionJSONCmd(taskDefArn)
				}
			}
		}

	case ActionYank:
		if len(m.taskDefRevisions) > 0 && m.taskDefRevisionCursor < len(m.taskDefRevisions) {
			rev := m.taskDefRevisions[m.taskDefRevisionCursor]
			taskDefName := fmt.Sprintf("%s:%d", rev.Family, rev.Revision)
			err := copyToClipboard(taskDefName)
			if err == nil {
				m.clipboardMsg = fmt.Sprintf("Copied: %s", taskDefName)
			} else {
				m.clipboardMsg = fmt.Sprintf("Copy failed: %v", err)
			}
			m.clipboardMsgTime = time.Now()
		}

	case ActionEditTaskDef:
		if len(m.taskDefRevisions) > 0 && m.taskDefRevisionCursor < len(m.taskDefRevisions) {
			rev := m.taskDefRevisions[m.taskDefRevisionCursor]
			revision := &rev.Revision
			m.taskDefEditor = NewTaskDefinitionEditor(rev.Family, revision)
			m.previousView = m.currentView
			m.currentView = ViewTaskDefinitionEditor
		}

	case ActionCopyJSON:
		// TODO: Implement copy JSON to clipboard

	case ActionDeregisterTaskDef:
		// TODO: Implement deregister task definition

	case ActionScrollUp:
		if m.showTaskDefJSON && m.taskDefJSONScroll > 0 {
			m.taskDefJSONScroll -= 5
			if m.taskDefJSONScroll < 0 {
				m.taskDefJSONScroll = 0
			}
		}

	case ActionScrollDown:
		if m.showTaskDefJSON {
			m.taskDefJSONScroll += 5
		}
	}

	return m, nil
}
