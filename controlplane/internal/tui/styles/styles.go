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

package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primary   = lipgloss.AdaptiveColor{Light: "#007ACC", Dark: "#40A6FF"}
	secondary = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	success   = lipgloss.AdaptiveColor{Light: "#10B981", Dark: "#34D399"}
	warning   = lipgloss.AdaptiveColor{Light: "#F59E0B", Dark: "#FBBF24"}
	danger    = lipgloss.AdaptiveColor{Light: "#EF4444", Dark: "#F87171"}
	muted     = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"}

	// Base styles
	BaseStyle = lipgloss.NewStyle()

	// Header and footer
	Header = BaseStyle.
		Background(primary).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Bold(true)

	Footer = BaseStyle.
		Background(lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#374151"}).
		Foreground(lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"}).
		Padding(0, 1)

	// Content area
	Content = BaseStyle.
		Padding(1, 2)

	// List styles
	ListTitle = BaseStyle.
		Foreground(primary).
		Bold(true).
		Padding(0, 1)

	ListItem = BaseStyle.
		Padding(0, 2)

	SelectedListItem = ListItem.
		Background(lipgloss.AdaptiveColor{Light: "#E0E7FF", Dark: "#1E3A8A"}).
		Foreground(lipgloss.AdaptiveColor{Light: "#1E40AF", Dark: "#DBEAFE"})

	// Status styles
	StatusRunning = BaseStyle.
		Foreground(success).
		Bold(true)

	StatusPending = BaseStyle.
		Foreground(warning).
		Bold(true)

	StatusStopped = BaseStyle.
		Foreground(danger).
		Bold(true)

	StatusUnknown = BaseStyle.
		Foreground(muted).
		Bold(true)

	// Table styles
	TableHeader = BaseStyle.
		Foreground(primary).
		Bold(true).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(muted)

	TableCell = BaseStyle.
		Padding(0, 1)

	// Help styles
	HelpKey = BaseStyle.
		Foreground(primary).
		Bold(true)

	HelpDesc = BaseStyle.
		Foreground(secondary)

	// Error styles
	Error = BaseStyle.
		Foreground(danger).
		Bold(true)

	// Success styles
	Success = BaseStyle.
		Foreground(success).
		Bold(true)

	// Info styles
	Info = BaseStyle.
		Foreground(primary)

	// Border styles
	ActiveBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	InactiveBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
	}

	// Panel styles
	ActivePanel = BaseStyle.
		Border(ActiveBorder).
		BorderForeground(primary)

	InactivePanel = BaseStyle.
		Border(InactiveBorder).
		BorderForeground(muted)
)

// GetStatusStyle returns the appropriate style for a given status
func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "RUNNING", "ACTIVE", "HEALTHY":
		return StatusRunning
	case "PENDING", "PROVISIONING", "ACTIVATING":
		return StatusPending
	case "STOPPED", "INACTIVE", "FAILED":
		return StatusStopped
	default:
		return StatusUnknown
	}
}