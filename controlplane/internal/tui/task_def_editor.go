package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TaskDefEditorMode represents the current mode of the editor
type TaskDefEditorMode int

const (
	EditorModeNormal TaskDefEditorMode = iota
	EditorModeInsert
	EditorModeCommand
)

// NewTaskDefinitionEditor creates a new task definition editor
func NewTaskDefinitionEditor(family string, baseRevision *int) *TaskDefinitionEditor {
	editor := &TaskDefinitionEditor{
		family:       family,
		baseRevision: baseRevision,
		content:      "{\n  \n}",
		cursorLine:   1,
		cursorCol:    2,
		mode:         EditorModeNormal,
		errors:       []ValidationError{},
	}

	// If we have a base revision, we'll load its content
	if baseRevision != nil {
		// Content will be loaded asynchronously
		editor.content = "// Loading task definition..."
	}

	return editor
}

// Update handles editor updates
func (e *TaskDefinitionEditor) Update(msg tea.Msg) (*TaskDefinitionEditor, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch e.mode {
		case EditorModeNormal:
			return e.handleNormalMode(msg)
		case EditorModeInsert:
			return e.handleInsertMode(msg)
		case EditorModeCommand:
			return e.handleCommandMode(msg)
		}
	case taskDefJSONLoadedMsg:
		// Load the JSON content for editing
		e.content = msg.json
		e.cursorLine = 0
		e.cursorCol = 0
		e.validateJSON()
		return e, nil
	}

	return e, nil
}

// handleNormalMode handles key presses in normal mode
func (e *TaskDefinitionEditor) handleNormalMode(msg tea.KeyMsg) (*TaskDefinitionEditor, tea.Cmd) {
	lines := strings.Split(e.content, "\n")

	switch msg.String() {
	case "i":
		// Enter insert mode
		e.mode = EditorModeInsert

	case "h", "left":
		// Move cursor left
		if e.cursorCol > 0 {
			e.cursorCol--
		}

	case "l", "right":
		// Move cursor right
		if e.cursorLine < len(lines) && e.cursorCol < len(lines[e.cursorLine]) {
			e.cursorCol++
		}

	case "j", "down":
		// Move cursor down
		if e.cursorLine < len(lines)-1 {
			e.cursorLine++
			// Adjust column if needed
			if e.cursorCol > len(lines[e.cursorLine]) {
				e.cursorCol = len(lines[e.cursorLine])
			}
		}
		
	case "enter", "+":
		// Move to beginning of next line (Vim behavior)
		if e.cursorLine < len(lines)-1 {
			e.cursorLine++
			e.cursorCol = 0
		}
		
	case "-":
		// Move to beginning of previous line (Vim behavior)
		if e.cursorLine > 0 {
			e.cursorLine--
			e.cursorCol = 0
		}

	case "k", "up":
		// Move cursor up
		if e.cursorLine > 0 {
			e.cursorLine--
			// Adjust column if needed
			if e.cursorCol > len(lines[e.cursorLine]) {
				e.cursorCol = len(lines[e.cursorLine])
			}
		}

	case "0", "home":
		// Move to beginning of line
		e.cursorCol = 0

	case "$", "end":
		// Move to end of line
		if e.cursorLine < len(lines) {
			e.cursorCol = len(lines[e.cursorLine])
		}
		
	case "^":
		// Move to first non-blank character of line
		if e.cursorLine < len(lines) {
			line := lines[e.cursorLine]
			for i, ch := range line {
				if ch != ' ' && ch != '\t' {
					e.cursorCol = i
					return e, nil
				}
			}
			// If all spaces, stay at beginning
			e.cursorCol = 0
		}
		
	case "w":
		// Move to next word
		if e.cursorLine < len(lines) {
			e.moveToNextWord(lines)
		}
		
	case "b":
		// Move to previous word
		if e.cursorLine < len(lines) {
			e.moveToPrevWord(lines)
		}

	case "g":
		// Move to first line
		e.cursorLine = 0
		e.cursorCol = 0

	case "G":
		// Move to last line
		e.cursorLine = len(lines) - 1
		if e.cursorLine < 0 {
			e.cursorLine = 0
		}
		e.cursorCol = 0

	case ":":
		// Enter command mode
		e.mode = EditorModeCommand
		e.commandBuffer = ""

	case "v":
		// Validate JSON
		e.validateJSON()

	case "ctrl+f":
		// Format JSON
		return e, e.formatJSON()
		
	case "ctrl+q":
		// Quick quit without saving
		return e, func() tea.Msg {
			return editorQuitMsg{}
		}
	}

	return e, nil
}

// handleInsertMode handles key presses in insert mode
func (e *TaskDefinitionEditor) handleInsertMode(msg tea.KeyMsg) (*TaskDefinitionEditor, tea.Cmd) {
	lines := strings.Split(e.content, "\n")

	switch msg.String() {
	case "esc":
		// Exit insert mode
		e.mode = EditorModeNormal
		// Validate on exit
		e.validateJSON()

	case "enter":
		// Insert newline
		if e.cursorLine >= len(lines) {
			lines = append(lines, "")
		}

		line := lines[e.cursorLine]
		before := ""
		after := ""

		if e.cursorCol <= len(line) {
			before = line[:e.cursorCol]
			if e.cursorCol < len(line) {
				after = line[e.cursorCol:]
			}
		}

		// Insert new line
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:e.cursorLine]...)
		newLines = append(newLines, before)
		newLines = append(newLines, after)
		if e.cursorLine+1 < len(lines) {
			newLines = append(newLines, lines[e.cursorLine+1:]...)
		}

		e.content = strings.Join(newLines, "\n")
		e.cursorLine++
		e.cursorCol = 0

	case "backspace":
		// Delete character before cursor
		if e.cursorCol > 0 {
			if e.cursorLine < len(lines) {
				line := lines[e.cursorLine]
				if e.cursorCol <= len(line) {
					lines[e.cursorLine] = line[:e.cursorCol-1] + line[e.cursorCol:]
					e.content = strings.Join(lines, "\n")
					e.cursorCol--
				}
			}
		} else if e.cursorLine > 0 {
			// Join with previous line
			prevLine := lines[e.cursorLine-1]
			currentLine := ""
			if e.cursorLine < len(lines) {
				currentLine = lines[e.cursorLine]
			}

			newLines := make([]string, 0, len(lines)-1)
			newLines = append(newLines, lines[:e.cursorLine-1]...)
			newLines = append(newLines, prevLine+currentLine)
			if e.cursorLine+1 < len(lines) {
				newLines = append(newLines, lines[e.cursorLine+1:]...)
			}

			e.content = strings.Join(newLines, "\n")
			e.cursorLine--
			e.cursorCol = len(prevLine)
		}

	case "tab":
		// Insert spaces (2 spaces for indentation)
		e.insertText("  ")

	default:
		// Insert character
		if len(msg.String()) == 1 {
			e.insertText(msg.String())
		}
	}

	return e, nil
}

// handleCommandMode handles command mode input
func (e *TaskDefinitionEditor) handleCommandMode(msg tea.KeyMsg) (*TaskDefinitionEditor, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Exit command mode
		e.mode = EditorModeNormal
		e.commandBuffer = ""

	case "enter":
		// Execute command
		cmd := e.executeCommand(e.commandBuffer)
		e.mode = EditorModeNormal
		e.commandBuffer = ""
		return e, cmd

	case "backspace":
		// Delete character from command buffer
		if len(e.commandBuffer) > 0 {
			e.commandBuffer = e.commandBuffer[:len(e.commandBuffer)-1]
		}

	default:
		// Add to command buffer
		if len(msg.String()) == 1 {
			e.commandBuffer += msg.String()
		}
	}

	return e, nil
}

// insertText inserts text at the current cursor position
func (e *TaskDefinitionEditor) insertText(text string) {
	lines := strings.Split(e.content, "\n")

	if e.cursorLine >= len(lines) {
		// Extend lines if needed
		for i := len(lines); i <= e.cursorLine; i++ {
			lines = append(lines, "")
		}
	}

	line := lines[e.cursorLine]
	before := ""
	after := ""

	if e.cursorCol <= len(line) {
		before = line[:e.cursorCol]
		if e.cursorCol < len(line) {
			after = line[e.cursorCol:]
		}
	} else {
		before = line
	}

	lines[e.cursorLine] = before + text + after
	e.content = strings.Join(lines, "\n")
	e.cursorCol += len(text)
}

// validateJSON validates the current content as JSON
func (e *TaskDefinitionEditor) validateJSON() {
	e.errors = []ValidationError{}

	// Try to parse as JSON
	var result interface{}
	decoder := json.NewDecoder(strings.NewReader(e.content))

	err := decoder.Decode(&result)
	if err != nil {
		// Try to extract line and column from error
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			line, col := findLineAndColumn(e.content, int(syntaxErr.Offset))
			e.errors = append(e.errors, ValidationError{
				Line:    line,
				Column:  col,
				Message: syntaxErr.Error(),
			})
		} else {
			e.errors = append(e.errors, ValidationError{
				Line:    0,
				Column:  0,
				Message: err.Error(),
			})
		}
	}

	// Additional validation for task definition structure
	if len(e.errors) == 0 {
		e.validateTaskDefinitionStructure(result)
	}
}

// validateTaskDefinitionStructure validates the task definition structure
func (e *TaskDefinitionEditor) validateTaskDefinitionStructure(data interface{}) {
	taskDef, ok := data.(map[string]interface{})
	if !ok {
		e.errors = append(e.errors, ValidationError{
			Line:    0,
			Column:  0,
			Message: "Root must be a JSON object",
		})
		return
	}

	// Check for required fields
	requiredFields := []string{"family", "containerDefinitions"}
	for _, field := range requiredFields {
		if _, exists := taskDef[field]; !exists {
			e.errors = append(e.errors, ValidationError{
				Line:    0,
				Column:  0,
				Message: fmt.Sprintf("Required field '%s' is missing", field),
			})
		}
	}

	// Validate containerDefinitions
	if containerDefs, exists := taskDef["containerDefinitions"]; exists {
		if containers, ok := containerDefs.([]interface{}); ok {
			if len(containers) == 0 {
				e.errors = append(e.errors, ValidationError{
					Line:    0,
					Column:  0,
					Message: "containerDefinitions must contain at least one container",
				})
			}

			// Validate each container
			for i, container := range containers {
				if containerMap, ok := container.(map[string]interface{}); ok {
					// Check required container fields
					containerRequired := []string{"name", "image"}
					for _, field := range containerRequired {
						if _, exists := containerMap[field]; !exists {
							e.errors = append(e.errors, ValidationError{
								Line:    0,
								Column:  0,
								Message: fmt.Sprintf("Container %d: required field '%s' is missing", i, field),
							})
						}
					}
				}
			}
		} else {
			e.errors = append(e.errors, ValidationError{
				Line:    0,
				Column:  0,
				Message: "containerDefinitions must be an array",
			})
		}
	}
}

// formatJSON formats the JSON content
func (e *TaskDefinitionEditor) formatJSON() tea.Cmd {
	return func() tea.Msg {
		// Try to parse and format
		var data interface{}
		err := json.Unmarshal([]byte(e.content), &data)
		if err != nil {
			return nil
		}

		formatted, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil
		}

		e.content = string(formatted)
		return nil
	}
}

// executeCommand executes a command mode command
func (e *TaskDefinitionEditor) executeCommand(command string) tea.Cmd {
	switch command {
	case "w", "write":
		// Save command - would trigger save
		return e.saveTaskDefinition()

	case "q", "quit":
		// Quit without saving
		return func() tea.Msg {
			return editorQuitMsg{}
		}

	case "wq":
		// Save and quit
		return tea.Sequence(
			e.saveTaskDefinition(),
			func() tea.Msg {
				return editorQuitMsg{}
			},
		)

	case "format", "fmt":
		// Format JSON
		return e.formatJSON()

	default:
		// Unknown command
		e.errors = []ValidationError{{
			Line:    0,
			Column:  0,
			Message: fmt.Sprintf("Unknown command: %s", command),
		}}
		return nil
	}
}

// saveTaskDefinition saves the task definition
func (e *TaskDefinitionEditor) saveTaskDefinition() tea.Cmd {
	// Validate before saving
	e.validateJSON()
	if len(e.errors) > 0 {
		return nil
	}

	return func() tea.Msg {
		// In a real implementation, this would call the API
		// For now, just return a success message
		return editorSaveMsg{
			family:   e.family,
			revision: 1, // Would be determined by API
		}
	}
}

// Render renders the editor view
func (e *TaskDefinitionEditor) Render(width, height int) string {
	// Styles
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1e1e2e")).
		Foreground(lipgloss.Color("#cdd6f4")).
		Padding(0, 1).
		Bold(true)

	lineNumberStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086"))

	cursorStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#74c7ec")).
		Foreground(lipgloss.Color("#1e1e2e"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f38ba8"))

	modeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#a6e3a1")).
		Foreground(lipgloss.Color("#1e1e2e")).
		Bold(true).
		Padding(0, 1)

	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdd6f4"))

	// Build header
	title := fmt.Sprintf("Task Definition Editor - %s", e.family)
	if e.baseRevision != nil {
		title += fmt.Sprintf(" (based on revision %d)", *e.baseRevision)
	}
	header := headerStyle.Width(width).Render(title)

	// Calculate available space
	availableHeight := height - 3 // Header, status line, mode line

	// Build content with line numbers
	lines := strings.Split(e.content, "\n")
	contentLines := []string{}

	// Calculate visible range
	startLine := 0
	if e.cursorLine >= availableHeight {
		startLine = e.cursorLine - availableHeight + 1
	}
	endLine := startLine + availableHeight
	if endLine > len(lines) {
		endLine = len(lines)
	}

	// Render visible lines
	for i := startLine; i < endLine; i++ {
		lineNum := lineNumberStyle.Render(fmt.Sprintf("%4d ", i+1))
		line := lines[i]

		// Apply cursor if on this line
		if i == e.cursorLine && e.mode != EditorModeCommand {
			lineRunes := []rune(line)
			before := ""
			cursor := " "
			after := ""

			if e.cursorCol < len(lineRunes) {
				before = string(lineRunes[:e.cursorCol])
				cursor = string(lineRunes[e.cursorCol])
				if e.cursorCol+1 < len(lineRunes) {
					after = string(lineRunes[e.cursorCol+1:])
				}
			} else {
				before = line
			}

			line = before + cursorStyle.Render(cursor) + after
		}

		// Check for errors on this line
		hasError := false
		for _, err := range e.errors {
			if err.Line == i+1 {
				hasError = true
				break
			}
		}

		if hasError {
			line = errorStyle.Render("✗ ") + line
		} else {
			line = "  " + line
		}

		contentLines = append(contentLines, lineNum+line)
	}

	content := strings.Join(contentLines, "\n")

	// Build status line
	statusParts := []string{}

	// Mode indicator
	modeText := "NORMAL"
	switch e.mode {
	case EditorModeInsert:
		modeText = "INSERT"
		modeStyle = modeStyle.Background(lipgloss.Color("#fab387"))
	case EditorModeCommand:
		modeText = "COMMAND"
		modeStyle = modeStyle.Background(lipgloss.Color("#f9e2af"))
	}
	statusParts = append(statusParts, modeStyle.Render(modeText))

	// Position indicator
	positionText := fmt.Sprintf(" %d:%d ", e.cursorLine+1, e.cursorCol+1)
	statusParts = append(statusParts, positionText)

	// Error indicator
	if len(e.errors) > 0 {
		errorText := fmt.Sprintf(" %d error(s) ", len(e.errors))
		statusParts = append(statusParts, errorStyle.Render(errorText))
	}

	statusLine := strings.Join(statusParts, "")

	// Command line (if in command mode)
	bottomLine := ""
	if e.mode == EditorModeCommand {
		bottomLine = commandStyle.Render(":" + e.commandBuffer + "_")
	} else if len(e.errors) > 0 {
		// Show first error
		bottomLine = errorStyle.Render(fmt.Sprintf("Error: %s", e.errors[0].Message))
	} else {
		// Show help
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
		helpText := "i: insert | :w: save | :q: quit | ^Q: quick quit | v: validate | ^F: format | ESC: exit"
		bottomLine = helpStyle.Render(helpText)
	}

	// Combine all parts
	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		content,
		statusLine,
		bottomLine,
	)
}

// Helper functions

// findLineAndColumn finds the line and column number for a byte offset
func findLineAndColumn(content string, offset int) (line, column int) {
	line = 1
	column = 1

	for i, ch := range content {
		if i >= offset {
			break
		}

		if ch == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}

	return line, column
}

// moveToNextWord moves cursor to the beginning of the next word
func (e *TaskDefinitionEditor) moveToNextWord(lines []string) {
	if e.cursorLine >= len(lines) {
		return
	}
	
	line := lines[e.cursorLine]
	inWord := false
	
	// Start from current position
	for i := e.cursorCol; i < len(line); i++ {
		ch := line[i]
		isWordChar := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || 
		             (ch >= '0' && ch <= '9') || ch == '_'
		
		if !inWord && isWordChar {
			// Found start of word after current position
			if i > e.cursorCol {
				e.cursorCol = i
				return
			}
			inWord = true
		} else if inWord && !isWordChar {
			// End of current word
			inWord = false
		}
	}
	
	// If no word found on current line, try next line
	if e.cursorLine < len(lines)-1 {
		e.cursorLine++
		e.cursorCol = 0
		// Find first word on new line
		line = lines[e.cursorLine]
		for i, ch := range line {
			isWordChar := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || 
			             (ch >= '0' && ch <= '9') || ch == '_'
			if isWordChar {
				e.cursorCol = i
				return
			}
		}
	}
}

// moveToPrevWord moves cursor to the beginning of the previous word
func (e *TaskDefinitionEditor) moveToPrevWord(lines []string) {
	if e.cursorLine >= len(lines) {
		return
	}
	
	// If at beginning of line, go to previous line
	if e.cursorCol == 0 && e.cursorLine > 0 {
		e.cursorLine--
		line := lines[e.cursorLine]
		// Find last word on previous line
		for i := len(line) - 1; i >= 0; i-- {
			ch := line[i]
			isWordChar := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || 
			             (ch >= '0' && ch <= '9') || ch == '_'
			if isWordChar {
				// Found end of word, find beginning
				for j := i; j >= 0; j-- {
					ch2 := line[j]
					isWordChar2 := (ch2 >= 'a' && ch2 <= 'z') || (ch2 >= 'A' && ch2 <= 'Z') || 
					              (ch2 >= '0' && ch2 <= '9') || ch2 == '_'
					if !isWordChar2 || j == 0 {
						if !isWordChar2 {
							e.cursorCol = j + 1
						} else {
							e.cursorCol = 0
						}
						return
					}
				}
			}
		}
		e.cursorCol = 0
		return
	}
	
	// Search backwards on current line
	line := lines[e.cursorLine]
	foundWord := false
	
	for i := e.cursorCol - 1; i >= 0; i-- {
		ch := line[i]
		isWordChar := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || 
		             (ch >= '0' && ch <= '9') || ch == '_'
		
		if isWordChar && !foundWord {
			foundWord = true
		} else if !isWordChar && foundWord {
			// Found beginning of word
			e.cursorCol = i + 1
			return
		}
	}
	
	// If found word that extends to beginning of line
	if foundWord {
		e.cursorCol = 0
	}
}

// Message types for editor
type editorSaveMsg struct {
	family   string
	revision int
}

type editorQuitMsg struct{}
