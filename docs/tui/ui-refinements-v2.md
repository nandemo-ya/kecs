# KECS TUI v2 UI Refinements

## Overview

The KECS TUI v2 has been refined with a new layout that provides better visual separation between navigation and resource areas, following a 3:7 height ratio design.

## Layout Structure

### Navigation Panel (30% height)
The top navigation panel contains:
- **Header**: KECS version, environment name, and connection status
- **Breadcrumb**: Navigation path showing the current location
- **Summary**: Contextual information about the current view with resource counts

### Resource Panel (70% height)
The bottom resource panel contains:
- **Resource List**: The main content area showing instances, clusters, services, or tasks
- **Scrolling**: Automatic scrolling with visual indicators when content exceeds available space
- **Selection**: Clear visual indication of the selected item with arrow (▸) marker

### Footer
Fixed footer showing context-sensitive keyboard shortcuts

## Key Improvements

### 1. Visual Separation
- Rounded borders clearly separate navigation and resource panels
- Consistent padding and spacing throughout
- Professional color scheme with good contrast

### 2. Responsive Column Widths
- Column widths automatically adjust based on terminal width
- Proportional sizing ensures optimal use of space
- Long values are truncated with ellipsis (...)

### 3. Enhanced Scrolling
- Scroll indicators show current position (e.g., "[Showing 1-10 of 25 instances]")
- Smooth scrolling that follows the cursor
- Maintains context when navigating large lists

### 4. Contextual Summary Information
- Each view shows relevant statistics in the navigation panel
- Examples:
  - Instances: "Total Instances: 3 | Active: 2 | Stopped: 1"
  - Clusters: "Instance: dev | Clusters: 2 | Total Services: 5 | Total Tasks: 12"
  - Services: "Cluster: default | Services: 5 | Desired Tasks: 10 | Running Tasks: 8"
  - Tasks: "Service: web-app | Tasks: 3 | Running: 3 | Healthy: 2"

### 5. Improved Color Coding
- **Green (#00ff00)**: Active/Running/Healthy resources
- **Yellow (#ffff00)**: Warning states, pending operations
- **Red (#ff0000)**: Failed/Stopped/Unhealthy resources
- **Cyan (#00ffff)**: Selected items and timestamps
- **Blue (#0000ff)**: Inactive resources
- **Orange (#ff8800)**: Transitional states (stopping, provisioning)
- **Gray (#808080)**: Headers and secondary information

## Terminal Compatibility

The refined UI is designed to work well with:
- Minimum terminal size: 80x24
- Recommended size: 120x40 or larger
- Full Unicode support for box drawing characters
- 256-color terminal support

## Navigation Flow

1. **Instance Selection**: Start at the instances view, select an instance
2. **Drill Down**: Navigate through clusters → services → tasks
3. **Breadcrumb Trail**: Always shows current location and navigation path
4. **Quick Navigation**: Use keyboard shortcuts (i, c, s, t) to jump between resource types
5. **Back Navigation**: Backspace to go up one level

## Implementation Details

### Height Calculation
```go
navHeight := int(float64(m.height-1) * 0.3)     // 30% for navigation
resourceHeight := int(float64(m.height-1) * 0.7) // 70% for resources
```

### Dynamic Column Sizing
```go
nameWidth := int(float64(availableWidth) * 0.25)      // 25% for names
statusWidth := int(float64(availableWidth) * 0.12)    // 12% for status
// ... other columns proportionally sized
```

### Scroll Position Management
The view automatically adjusts the visible window when the cursor moves outside the current view:
```go
if m.instanceCursor >= visibleRows {
    startIdx = m.instanceCursor - visibleRows + 1
}
```

## Future Enhancements

1. **Search Highlighting**: Highlight matching text when searching
2. **Multi-select Mode**: Allow selecting multiple resources for batch operations
3. **Resource Preview**: Show detailed information in a side panel
4. **Customizable Themes**: Allow users to customize colors and styles
5. **Export Options**: Export resource lists to various formats

## Usage Tips

1. **Maximize Terminal**: Use full-screen mode for the best experience
2. **Navigation**: Use vim-style keys (j/k) or arrow keys
3. **Quick Switch**: Use Ctrl+I to quickly switch between instances
4. **Refresh**: Press R to refresh the current view
5. **Help**: Press ? at any time to see keyboard shortcuts