# TUI KECS Startup Feature

This document describes the KECS startup functionality integrated into the TUI.

## Overview

The TUI now includes automatic detection and startup of KECS instances. When launching the TUI, it will:

1. Check if KECS is running at the specified endpoint
2. If not running, show a confirmation dialog
3. If confirmed, start KECS with live log streaming
4. Transition to the main TUI once KECS is ready

## User Experience

### Startup Dialog

When KECS is not running, users see:

```
┌─────────────────────────────────────────────────────────┐
│                    KECS Not Running                       │
│                                                           │
│ Could not connect to KECS at http://localhost:8080.      │
│                                                           │
│ Would you like to start KECS now?                        │
│                                                           │
│ This will:                                                │
│ • Start a new KECS instance                               │
│ • Create necessary Kubernetes resources                   │
│ • Display startup logs                                    │
│                                                           │
│       [Y] Yes, start KECS  [N] No, exit                  │
└─────────────────────────────────────────────────────────┘
```

### Startup Log Viewer

If the user confirms, they see live startup logs:

```
Starting KECS Instance: dev

⠋ Starting KECS...

┌──────────────────────────────────────────────────────────┐
│ [19:28:09] Checking prerequisites...                      │
│ [19:28:10] Creating Kubernetes cluster...                 │
│ [19:28:15] Waiting for cluster to be ready...             │
│ [19:28:25] Deploying KECS components...                   │
│ [19:28:30] Starting API server...                         │
│ [19:28:35] KECS is ready!                                 │
│                                                           │
│ ✅ KECS started successfully!                             │
│ Press ENTER to continue to the TUI                        │
└──────────────────────────────────────────────────────────┘

Press ENTER to continue
```

## Implementation Details

### Components

1. **Dialog Component** (`views/startup/dialog.go`)
   - Shows confirmation dialog
   - Handles user input (Y/N)
   - Returns confirmation state

2. **Log Viewer Component** (`views/startup/logviewer.go`)
   - Displays live logs from KECS startup
   - Shows progress spinner
   - Handles completion/failure states

3. **Starter Logic** (`views/startup/starter.go`)
   - Executes `kecs start` command
   - Streams stdout/stderr logs
   - Monitors startup progress
   - Checks API readiness

4. **App Integration** (`app/startup.go`, `app/startup_handler.go`)
   - State machine for startup flow
   - Integrates with main TUI lifecycle
   - Handles transitions between states

### Startup States

1. **StartupStateChecking**: Initial state, checking KECS status
2. **StartupStateDialog**: Showing confirmation dialog
3. **StartupStateStarting**: KECS is starting, showing logs
4. **StartupStateReady**: KECS is ready, proceed to main TUI

### Port Detection

The system automatically detects the appropriate port based on:
- Endpoint URL (extracts port)
- Instance name mapping (dev=8080, staging=8090, etc.)
- Hash-based allocation for custom instances

## Configuration

No additional configuration is required. The feature works automatically when:
- TUI is launched with `kecs`
- KECS is not running at the specified endpoint

## Future Enhancements

1. **Instance Selection**: Allow users to select which instance to start
2. **Configuration Options**: Let users configure startup parameters
3. **Recovery Options**: Handle partial startup failures gracefully
4. **Background Monitoring**: Continue monitoring KECS health after startup