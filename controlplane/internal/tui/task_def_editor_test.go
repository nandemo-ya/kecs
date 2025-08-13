package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTaskDefinitionEditor(t *testing.T) {
	t.Run("NewTaskDefinitionEditor creates editor with default content", func(t *testing.T) {
		editor := NewTaskDefinitionEditor("test-family", nil)
		
		if editor.family != "test-family" {
			t.Errorf("expected family 'test-family', got '%s'", editor.family)
		}
		
		if editor.baseRevision != nil {
			t.Error("expected baseRevision to be nil")
		}
		
		if editor.content != "{\n  \n}" {
			t.Errorf("expected default content, got '%s'", editor.content)
		}
		
		if editor.mode != EditorModeNormal {
			t.Error("expected editor to start in normal mode")
		}
	})
	
	t.Run("Editor validates JSON correctly", func(t *testing.T) {
		editor := NewTaskDefinitionEditor("test-family", nil)
		
		// Test with valid JSON
		editor.content = `{
  "family": "test-family",
  "containerDefinitions": [
    {
      "name": "main",
      "image": "nginx:latest"
    }
  ]
}`
		editor.validateJSON()
		
		if len(editor.errors) != 0 {
			t.Errorf("expected no errors for valid JSON, got %d errors", len(editor.errors))
		}
		
		// Test with invalid JSON
		editor.content = `{
  "family": "test-family",
  "containerDefinitions": [
}`
		editor.validateJSON()
		
		if len(editor.errors) == 0 {
			t.Error("expected errors for invalid JSON")
		}
	})
	
	t.Run("Editor mode transitions work correctly", func(t *testing.T) {
		editor := NewTaskDefinitionEditor("test-family", nil)
		
		// Start in normal mode
		if editor.mode != EditorModeNormal {
			t.Error("expected to start in normal mode")
		}
		
		// Press 'i' to enter insert mode
		editor, _ = editor.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
		if editor.mode != EditorModeInsert {
			t.Error("expected to enter insert mode after pressing 'i'")
		}
		
		// Press ESC to return to normal mode
		editor, _ = editor.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if editor.mode != EditorModeNormal {
			t.Error("expected to return to normal mode after pressing ESC")
		}
		
		// Press ':' to enter command mode
		editor, _ = editor.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		if editor.mode != EditorModeCommand {
			t.Error("expected to enter command mode after pressing ':'")
		}
	})
	
	t.Run("Editor handles JSON loading", func(t *testing.T) {
		editor := NewTaskDefinitionEditor("test-family", nil)
		
		testJSON := `{
  "family": "test-family",
  "containerDefinitions": []
}`
		
		editor, _ = editor.Update(taskDefJSONLoadedMsg{
			revision: 1,
			json:     testJSON,
		})
		
		if editor.content != testJSON {
			t.Errorf("expected content to be loaded, got '%s'", editor.content)
		}
		
		if editor.cursorLine != 0 || editor.cursorCol != 0 {
			t.Error("expected cursor to be reset after loading")
		}
	})
	
	t.Run("Editor validates task definition structure", func(t *testing.T) {
		editor := NewTaskDefinitionEditor("test-family", nil)
		
		// Missing required fields
		editor.content = `{}`
		editor.validateJSON()
		
		// Should have errors for missing family and containerDefinitions
		hasErrors := false
		for _, err := range editor.errors {
			if strings.Contains(err.Message, "family") || 
			   strings.Contains(err.Message, "containerDefinitions") {
				hasErrors = true
				break
			}
		}
		
		if !hasErrors {
			t.Error("expected validation errors for missing required fields")
		}
		
		// Empty containerDefinitions
		editor.content = `{
  "family": "test",
  "containerDefinitions": []
}`
		editor.validateJSON()
		
		hasContainerError := false
		for _, err := range editor.errors {
			if strings.Contains(err.Message, "at least one container") {
				hasContainerError = true
				break
			}
		}
		
		if !hasContainerError {
			t.Error("expected validation error for empty containerDefinitions")
		}
	})
	
	t.Run("Editor quit commands work correctly", func(t *testing.T) {
		editor := NewTaskDefinitionEditor("test-family", nil)
		
		// Test Ctrl+Q in normal mode
		editor, cmd := editor.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
		if cmd == nil {
			t.Error("expected Ctrl+Q to return a quit command")
		}
		
		// Test :q command
		editor.mode = EditorModeCommand
		editor.commandBuffer = "q"
		editor, cmd = editor.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected :q to return a quit command")
		}
		
		// Verify mode returned to normal after command
		if editor.mode != EditorModeNormal {
			t.Error("expected mode to return to normal after executing command")
		}
	})
	
	t.Run("Editor navigation keys work correctly", func(t *testing.T) {
		editor := NewTaskDefinitionEditor("test-family", nil)
		editor.content = "line 1\nline 2\nline 3"
		
		// Test Enter key - moves to next line beginning
		editor.cursorLine = 0
		editor.cursorCol = 3
		editor, _ = editor.Update(tea.KeyMsg{Type: tea.KeyEnter})
		
		if editor.cursorLine != 1 || editor.cursorCol != 0 {
			t.Errorf("expected cursor at line 1, col 0 after Enter, got line %d, col %d", 
				editor.cursorLine, editor.cursorCol)
		}
		
		// Test - key - moves to previous line beginning
		editor, _ = editor.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
		
		if editor.cursorLine != 0 || editor.cursorCol != 0 {
			t.Errorf("expected cursor at line 0, col 0 after -, got line %d, col %d", 
				editor.cursorLine, editor.cursorCol)
		}
		
		// Test ^ key - moves to first non-blank character
		editor.content = "  hello world"
		editor.cursorCol = 0
		editor, _ = editor.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'^'}})
		
		if editor.cursorCol != 2 {
			t.Errorf("expected cursor at col 2 (first non-blank), got col %d", editor.cursorCol)
		}
		
		// Test word navigation
		editor.content = "hello world json"
		editor.cursorLine = 0
		editor.cursorCol = 0
		
		// Test w - next word
		editor, _ = editor.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
		if editor.cursorCol != 6 { // "world" starts at position 6
			t.Errorf("expected cursor at col 6 after w, got col %d", editor.cursorCol)
		}
		
		// Test b - previous word
		editor, _ = editor.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		if editor.cursorCol != 0 { // back to "hello"
			t.Errorf("expected cursor at col 0 after b, got col %d", editor.cursorCol)
		}
	})
}