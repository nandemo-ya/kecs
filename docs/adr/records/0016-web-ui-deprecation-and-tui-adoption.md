# ADR-0016. Web UI Deprecation and TUI Adoption

Date: 2025-06-26

## Status

Proposed

## Context

KECS currently provides a React-based Web UI for managing ECS resources. While functional, this approach presents several challenges:

1. **Maintenance Burden**: Requires maintaining two separate codebases (Go backend + React frontend)
2. **Contribution Barrier**: Contributors need expertise in both Go and React/TypeScript
3. **Build Complexity**: Additional build steps for frontend assets and embedding
4. **Dependency Management**: Managing npm dependencies alongside Go modules
5. **Target Audience Mismatch**: KECS is a developer tool primarily used in terminals

### Current Web UI Architecture
- React 19 with TypeScript
- Embedded into Go binary using `embed` package
- Real-time updates via WebSocket
- Separate build pipeline for frontend assets

### Alternative Considered
We evaluated continuing with the Web UI but concluded that a Terminal User Interface (TUI) better aligns with our goals and user base.

## Decision

We will deprecate the Web UI in favor of a Terminal User Interface (TUI) built with [Bubbletea](https://github.com/charmbracelet/bubbletea).

### Implementation Plan

1. **TUI Framework**: Use Bubbletea and the Charm ecosystem
   - Bubbletea for the application framework
   - Bubbles for common UI components
   - Lipgloss for styling
   - Glamour for markdown rendering

2. **Feature Parity**: The TUI will provide equivalent functionality to the Web UI:
   - Dashboard view with resource overview
   - Cluster management
   - Service management
   - Task monitoring
   - Task definition management
   - Real-time updates

3. **Integration**: The TUI will be a subcommand of the main KECS binary:
   ```bash
   kecs tui              # Launch TUI interface
   kecs tui --endpoint   # Connect to remote KECS instance
   ```

4. **Migration Timeline**:
   - v0.x.x: Introduce TUI as experimental feature
   - v0.x.x: Feature parity with Web UI
   - v1.0.0: Remove Web UI completely

## Consequences

### Positive

1. **Unified Codebase**: Single language (Go) for entire project
2. **Lower Contribution Barrier**: Contributors only need Go knowledge
3. **Simplified Build**: No frontend build pipeline or asset embedding
4. **Better Developer Experience**: Native terminal integration, keyboard shortcuts, copy/paste
5. **Reduced Dependencies**: No npm/node dependencies
6. **Code Reuse**: Direct use of existing Go client code and models
7. **SSH Access**: TUI works over SSH without port forwarding
8. **Lightweight**: No browser required, lower resource usage

### Negative

1. **Learning Curve**: Team needs to learn Bubbletea framework
2. **UI Limitations**: TUIs have inherent limitations compared to web UIs
3. **Browser Accessibility**: Some users might prefer web interfaces
4. **Migration Effort**: Need to rebuild UI functionality in TUI

### Mitigation Strategies

1. **Progressive Enhancement**: Start with core features, add advanced features iteratively
2. **Documentation**: Comprehensive keyboard shortcuts and navigation guide
3. **Export Options**: Provide data export for users who need browser-based visualization
4. **API-First**: Ensure all functionality remains available via API for custom integrations

## Technical Details

### TUI Architecture

```
kecs tui
├── cmd/tui/          # TUI command entry point
├── internal/tui/
│   ├── app/          # Main application model
│   ├── views/        # Different views/screens
│   │   ├── dashboard/
│   │   ├── clusters/
│   │   ├── services/
│   │   ├── tasks/
│   │   └── taskdefs/
│   ├── components/   # Reusable UI components
│   ├── styles/       # Lipgloss styles
│   └── keys/         # Keyboard bindings
```

### Key Features

1. **Split Pane Layout**: Similar to k9s, with resource list and details
2. **Real-time Updates**: Using existing WebSocket or polling
3. **Keyboard Navigation**: Vim-like keybindings
4. **Context Switching**: Easy switching between clusters
5. **Resource Actions**: Create, update, delete operations
6. **Log Viewing**: Integrated task log viewer
7. **Help System**: Context-sensitive help

## References

- [Bubbletea](https://github.com/charmbracelet/bubbletea)
- [k9s](https://k9scli.io/) - Inspiration for TUI design
- [lazydocker](https://github.com/jesseduffield/lazydocker) - Another excellent TUI example
- [ADR-0005](./0005-web-ui.md) - Original Web UI decision (superseded)