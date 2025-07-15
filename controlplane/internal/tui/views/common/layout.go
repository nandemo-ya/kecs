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

package common

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// RenderFullScreen renders content centered in the full available space
func RenderFullScreen(width, height int, content string) string {
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		styles.Content.Render(content),
	)
}

// RenderListView renders a list view with proper full-screen layout
func RenderListView(width, height int, content string) string {
	return lipgloss.Place(
		width,
		height,
		lipgloss.Left,
		lipgloss.Top,
		styles.Content.Render(content),
	)
}

// CalculateTableHeight calculates appropriate table height based on view height
func CalculateTableHeight(viewHeight int, hasTitle bool, hasFooter bool) int {
	reserved := 2 // Base padding
	if hasTitle {
		reserved += 2 // Title and spacing
	}
	if hasFooter {
		reserved += 2 // Footer info and spacing
	}
	
	tableHeight := viewHeight - reserved
	if tableHeight < 5 {
		tableHeight = 5 // Minimum height
	}
	return tableHeight
}

// DistributeColumnWidths distributes available width among table columns
func DistributeColumnWidths(availableWidth int, minWidths []int, distribution []int) []int {
	totalMin := 0
	for _, w := range minWidths {
		totalMin += w
	}
	
	// If not enough space, use minimum widths
	if availableWidth <= totalMin {
		return minWidths
	}
	
	// Calculate extra space to distribute
	extra := availableWidth - totalMin
	
	// Calculate total distribution weight
	totalWeight := 0
	for _, weight := range distribution {
		totalWeight += weight
	}
	
	// Distribute extra space based on weights
	widths := make([]int, len(minWidths))
	for i, minWidth := range minWidths {
		if i < len(distribution) && totalWeight > 0 {
			extraForColumn := extra * distribution[i] / totalWeight
			widths[i] = minWidth + extraForColumn
		} else {
			widths[i] = minWidth
		}
	}
	
	return widths
}

// CreateTableColumns creates table columns with given widths and titles
func CreateTableColumns(titles []string, widths []int) []table.Column {
	columns := make([]table.Column, 0, len(titles))
	for i, title := range titles {
		width := 10 // default width
		if i < len(widths) {
			width = widths[i]
		}
		columns = append(columns, table.Column{
			Title: title,
			Width: width,
		})
	}
	return columns
}