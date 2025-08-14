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

	"github.com/charmbracelet/lipgloss"
)

// ConfirmDialog represents a confirmation dialog
type ConfirmDialog struct {
	title   string
	message string
	onYes   func() error
	onNo    func()
	focused bool // true = Yes, false = No
}

// NewConfirmDialog creates a new confirmation dialog
func NewConfirmDialog(title, message string, onYes func() error, onNo func()) *ConfirmDialog {
	return &ConfirmDialog{
		title:   title,
		message: message,
		onYes:   onYes,
		onNo:    onNo,
		focused: false, // Default to No for safety
	}
}

// FocusYes focuses the Yes button
func (d *ConfirmDialog) FocusYes() {
	d.focused = true
}

// FocusNo focuses the No button
func (d *ConfirmDialog) FocusNo() {
	d.focused = false
}

// Execute runs the selected action
func (d *ConfirmDialog) Execute() error {
	if d.focused {
		return d.onYes()
	}
	if d.onNo != nil {
		d.onNo()
	}
	return nil
}

// Render renders the dialog
func (d *ConfirmDialog) Render(width, height int) string {
	// Styles
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("220"))

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		MarginBottom(1)

	activeButtonStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 2).
		Bold(true)

	inactiveButtonStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("238")).
		Foreground(lipgloss.Color("250")).
		Padding(0, 2)

	// Build content
	var content strings.Builder

	// Title
	if d.title != "" {
		content.WriteString(titleStyle.Render(d.title))
		content.WriteString("\n\n")
	}

	// Message
	content.WriteString(messageStyle.Render(d.message))
	content.WriteString("\n")

	// Buttons
	var yesButton, noButton string
	if d.focused {
		yesButton = activeButtonStyle.Render("Yes")
		noButton = inactiveButtonStyle.Render("No")
	} else {
		yesButton = inactiveButtonStyle.Render("Yes")
		noButton = activeButtonStyle.Render("No")
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Center,
		yesButton,
		"  ", // Space between buttons
		noButton,
	)

	content.WriteString(lipgloss.PlaceHorizontal(
		dialogStyle.GetHorizontalFrameSize()+30,
		lipgloss.Center,
		buttons,
	))

	// Apply dialog style
	dialog := dialogStyle.Render(content.String())

	// Center the dialog
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// DeleteInstanceDialog creates a dialog for instance deletion
func DeleteInstanceDialog(instanceName string, onDelete func() error, onCancel func()) *ConfirmDialog {
	title := "Delete Instance"
	message := fmt.Sprintf("Are you sure you want to delete instance '%s'?\nThis action cannot be undone.", instanceName)
	return NewConfirmDialog(title, message, onDelete, onCancel)
}

// StartInstanceDialog creates a dialog for instance start confirmation
func StartInstanceDialog(instanceName string, onStart func() error, onCancel func()) *ConfirmDialog {
	title := "Start Instance"
	message := fmt.Sprintf("Start instance '%s'?", instanceName)
	return NewConfirmDialog(title, message, onStart, onCancel)
}

// StopInstanceDialog creates a dialog for instance stop confirmation
func StopInstanceDialog(instanceName string, onStop func() error, onCancel func()) *ConfirmDialog {
	title := "Stop Instance"
	message := fmt.Sprintf("Stop instance '%s'?\nAll running tasks will be terminated.", instanceName)
	return NewConfirmDialog(title, message, onStop, onCancel)
}
