# TUI JSON Editor for Task Definitions

The KECS TUI includes a built-in JSON editor for creating and editing task definitions. This editor provides vim-like key bindings and real-time JSON validation.

## Accessing the Editor

### From Task Definition Families View
1. Navigate to task definitions: Press `T` from any view with an instance selected
2. Press `N` to create a new task definition
3. The editor opens with a basic template

### From Task Definition Revisions View
1. Select a task definition family
2. Navigate to the desired revision
3. Press `e` to edit as a new revision
4. The editor opens with the selected revision's content

## Editor Modes

The editor has three modes:

### Normal Mode (default)
- Navigate and perform commands
- Move cursor with `h`, `j`, `k`, `l` or arrow keys
- Jump to start/end of line with `0`/`$`
- Jump to first/last line with `g`/`G`

### Insert Mode
- Press `i` to enter insert mode
- Type to insert text at cursor position
- Press `ESC` to return to normal mode
- Use `Tab` to insert 2 spaces for indentation

### Command Mode
- Press `:` to enter command mode
- Available commands:
  - `:w` or `:write` - Save the task definition
  - `:q` or `:quit` - Quit without saving
  - `:wq` - Save and quit
  - `:format` or `:fmt` - Format the JSON

## Key Bindings

### Normal Mode
- `i` - Enter insert mode
- `h`/`←` - Move cursor left
- `l`/`→` - Move cursor right
- `j`/`↓` - Move cursor down
- `k`/`↑` - Move cursor up
- `Enter`/`+` - Move to beginning of next line
- `-` - Move to beginning of previous line
- `0`/`Home` - Move to beginning of line
- `$`/`End` - Move to end of line
- `^` - Move to first non-blank character of line
- `w` - Move to next word
- `b` - Move to previous word
- `g` - Move to first line
- `G` - Move to last line
- `:` - Enter command mode
- `v` - Validate JSON
- `Ctrl+F` - Format JSON
- `Ctrl+Q` - Quick quit without saving
- `ESC` - Exit editor (when in normal mode)

### Insert Mode
- `ESC` - Return to normal mode
- `Enter` - Insert newline
- `Backspace` - Delete character before cursor
- `Tab` - Insert 2 spaces

### Command Mode
- `ESC` - Cancel command
- `Enter` - Execute command
- `Backspace` - Delete character from command

## JSON Validation

The editor provides real-time validation:

1. **Syntax Validation**: Checks for valid JSON syntax
2. **Structure Validation**: Ensures required fields are present:
   - `family` - Task definition family name
   - `containerDefinitions` - Array with at least one container
   - Each container must have `name` and `image` fields

Validation errors are shown:
- Error markers (`✗`) on lines with errors
- Error count in the status bar
- First error message displayed at the bottom

## Status Bar

The status bar shows:
- Current mode (NORMAL/INSERT/COMMAND)
- Cursor position (line:column)
- Error count (if any)

## Example Task Definition Template

```json
{
  "family": "my-app",
  "containerDefinitions": [
    {
      "name": "main",
      "image": "nginx:latest",
      "memory": 512,
      "cpu": 256,
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "protocol": "tcp"
        }
      ]
    }
  ],
  "requiresCompatibilities": ["EC2"],
  "networkMode": "bridge",
  "memory": "512",
  "cpu": "256"
}
```

## Tips

1. Use `Ctrl+F` to auto-format your JSON for better readability
2. Press `v` in normal mode to manually trigger validation
3. The editor automatically validates when exiting insert mode
4. Use the template as a starting point and modify as needed
5. Save frequently with `:w` to avoid losing work