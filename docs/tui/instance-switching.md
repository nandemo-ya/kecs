# TUI Instance Switching Implementation

This document describes the implementation of instance switching functionality in the KECS TUI.

## Overview

The TUI now supports switching between different KECS instances without restarting the application. Users can:

1. View all available instances with their status (running/stopped)
2. Start stopped instances
3. Stop running instances  
4. Destroy instances (with optional data deletion)
5. Switch between running instances using the quick switch dialog ('i' key)

## Implementation Details

### Port Mapping

Currently, the port mapping is handled with a simple scheme:
- `dev`: port 8080
- `staging`: port 8090  
- `test`: port 8100
- `local`: port 8110
- `prod`: port 8200
- Other instances: hash-based port allocation (8300-8999)

This should be replaced with actual port detection from k3d in the future.

### Instance Switching Flow

1. User presses 'i' to open quick switch dialog
2. Dialog shows only running instances
3. User selects an instance and presses Enter
4. TUI updates the API client endpoint to the new instance
5. All views are re-initialized with the new endpoint
6. Dashboard shows the current instance name

### Key Components Modified

1. **api/instances.go**: Added port mapping logic and GetCurrentInstance method
2. **app/app.go**: Added switchToInstance method and instance switching message handling
3. **views/*/list.go**: Added SetEndpoint methods to all view models
4. **views/instances/**: Complete instance management views with operations
5. **styles/styles.go**: Added missing styles for forms and UI elements

## Usage

1. Press '6' to go to the instances view
2. Use arrow keys to navigate instances
3. Press Enter on a stopped instance to start it
4. Press Enter on a running instance to switch to it
5. Press 's' to stop an instance
6. Press 'd' to destroy an instance
7. Press 'i' from any view to quickly switch instances

## Future Improvements

1. Retrieve actual port mappings from k3d container inspection
2. Store instance configuration in a persistent format
3. Support for remote instance management
4. Better error handling and notifications for failed operations
5. Instance health checks and resource monitoring