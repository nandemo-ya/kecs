# ELBv2 Keyboard Navigation Fix

## Problem
The ELBv2 views (Load Balancers, Target Groups, Listeners) were not responding to keyboard input correctly. Only the Tab key was working, while other navigation keys (up/down, j/k, c, b, g, enter, etc.) were not functioning.

## Root Cause
The issue was in the event routing in `app.go`. The `Update` function was checking for keybindings and routing to `executeAction` BEFORE checking if the current view was an ELBv2 view that needed special handling via `handleELBv2Keys`.

This meant that keys registered in the keybindings were being intercepted and sent to `executeAction`, which didn't have proper ELBv2 view handling logic.

## Solution
Modified the `Update` function in `app.go` to check for ELBv2 views FIRST and route directly to `handleELBv2Keys` before any keybinding checks. This ensures that ELBv2 views get their custom key handling.

### Key Changes:

1. **app.go** - Reordered key event handling:
   - ELBv2 views are now checked first
   - Keys are routed directly to `handleELBv2Keys` for these views
   - This bypasses the keybinding system for ELBv2 views

2. **elbv2_commands.go** - Added comprehensive debug logging:
   - Logs every key press with detailed context
   - Tracks cursor movements
   - Logs navigation actions
   - Helps diagnose any remaining issues

3. **debug_logger.go** - Debug logging system:
   - Writes to `~/.kecs/tui-debug.log`
   - Provides detailed trace of keyboard events
   - Essential for troubleshooting TUI issues

## Testing
To test the fix:
1. Build the CLI: `make build-cli`
2. Run the TUI: `./bin/kecs`
3. Navigate to an instance and select it
4. Press 'b' to go to Load Balancers view
5. Test keyboard navigation:
   - up/down or j/k: Move cursor
   - c: Navigate to Clusters
   - b: Navigate to Load Balancers
   - g: Navigate to Target Groups
   - enter: View Listeners for selected LB
   - y: Yank ARN
   - r: Refresh data
6. Check debug log: `tail -f ~/.kecs/tui-debug.log`

## Debug Output
The debug logger will show:
- Which keys are pressed
- Current view and cursor positions
- Routing decisions
- Action executions
- Any unhandled keys