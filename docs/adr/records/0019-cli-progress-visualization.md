# ADR-0019: CLI Progress Visualization Improvements

Date: 2025-07-17

## Status

Proposed

## Context

The current `start-v2` and `stop-v2` commands provide limited visibility into the deployment process. Users experience:

1. **Poor Progress Visibility**: Simple dot (`.`) progress indicators don't convey meaningful information
2. **Unclear Parallel Operations**: When deploying Control Plane and LocalStack simultaneously, users can't see individual progress
3. **Long Wait Times**: During extended operations (2-5 minutes), users have no insight into what's happening
4. **Limited Error Context**: Error messages lack details needed for troubleshooting

Example of current output:
```
Waiting for control plane deployment to be ready...........
```

This creates a poor user experience, especially for operations that can take several minutes.

## Decision

We will implement enhanced progress visualization for CLI commands using modern terminal UI libraries.

### Implementation Approach

#### Phase 1: CLI Progress Enhancement (This ADR)
Improve the command-line output of `start-v2` and `stop-v2` commands with:
- Progress bars with time estimates
- Parallel operation visualization
- Structured, colorized output
- Enhanced error messages with context

#### Phase 2: TUI Integration (Update to ADR-0016)
Extend the existing TUI to include deployment monitoring capabilities.

### Technical Implementation

#### Libraries
- **Progress Bars**: `github.com/schollz/progressbar/v3` - Thread-safe, customizable progress bars
- **Formatted Output**: `github.com/pterm/pterm` - Beautified terminal output with sections and colors

#### Progress Display Architecture

```go
// internal/progress/progress.go
type ProgressTracker struct {
    bars     map[string]*progressbar.ProgressBar
    mu       sync.Mutex
    pterm    *pterm.SpinnerPrinter
}

type ParallelProgress struct {
    tasks    []ProgressTask
    tracker  *ProgressTracker
}
```

### Visual Examples

#### Before
```
Starting KECS instance 'determined-elephant'...
Creating infrastructure for KECS instance 'determined-elephant'
Waiting for control plane deployment to be ready...........
```

#### After
```
┌─ Creating KECS instance 'determined-elephant' ─────────────────────
│
│ ► Creating k3d cluster                    [████████████████] 100% (45s)
│ ► Creating kecs-system namespace          [████████████████] 100% (2s)
│
│ ▼ Deploying components (parallel)
│   ├─ Control Plane    [████████░░░░░░░░] 65% (2m15s) Waiting for pods...
│   └─ LocalStack       [████████████████] 100% (1m30s) ✓ Ready
│
│ ► Deploying Traefik gateway              [██░░░░░░░░░░░░░░] 15% (10s)
│
└─────────────────────────────────────────────────────────────────────
```

#### Error Display Enhancement

Before:
```
failed to deploy control plane: failed to apply manifests: exit status 1
```

After:
```
✗ Failed to deploy control plane

  Error: Manifest application failed
  
  Details:
  - Command: kubectl apply -k manifests/
  - Exit code: 1
  - Output: error validating data: unknown field "replicas"
  
  Suggestions:
  - Check manifest syntax in controlplane/manifests/
  - Verify Kubernetes API version compatibility
  - Run with --debug for detailed logs
```

## Consequences

### Positive

1. **Improved User Experience**: Clear visibility into long-running operations
2. **Better Debugging**: Enhanced error messages reduce troubleshooting time
3. **Professional Appearance**: Modern CLI output matches user expectations
4. **Reduced Support Burden**: Clearer errors mean fewer support questions
5. **Minimal Dependencies**: Only two well-maintained libraries added

### Negative

1. **Additional Dependencies**: Two new Go libraries to manage
2. **Terminal Compatibility**: Some terminals may not support all formatting
3. **Output Complexity**: More complex output handling in code

### Mitigation Strategies

1. **Graceful Degradation**: Detect terminal capabilities and fall back to simple output
2. **Verbose Mode**: Add `--simple-output` flag for CI/CD environments
3. **Testing**: Test on various terminal emulators and CI environments

## Implementation Plan

### Phase 1 Deliverables

1. **Progress Utility Package** (`internal/progress/`)
   - Progress bar manager
   - Parallel progress tracker
   - Error formatter

2. **Update start_v2.go**
   - Replace `fmt.Print(".")` with progress bars
   - Add parallel progress tracking
   - Implement structured error handling

3. **Update stop_v2.go**
   - Add deletion progress
   - Optional confirmation prompt

4. **Documentation**
   - Update user guide with new output examples
   - Add troubleshooting guide based on enhanced errors

### Success Metrics

1. **User Feedback**: Positive response to improved visibility
2. **Reduced Support**: Fewer questions about "stuck" deployments
3. **Error Resolution**: Faster problem resolution due to better error messages

## Future Considerations

This implementation lays the groundwork for:
- Integration with existing TUI (see ADR-0016 update)
- Structured logging output (JSON mode for automation)
- Progress webhooks for external monitoring

## References

- [schollz/progressbar](https://github.com/schollz/progressbar) - Progress bar library
- [pterm/pterm](https://github.com/pterm/pterm) - Terminal UI library
- [ADR-0016](./0016-web-ui-deprecation-and-tui-adoption.md) - TUI implementation (to be updated)