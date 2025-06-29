# KECS TUI Developer Guide

This guide covers the architecture and development of the KECS Terminal User Interface.

## Architecture Overview

The KECS TUI is built using the following technologies:
- **Bubble Tea**: The Elm Architecture framework for terminal UIs
- **Bubbles**: Component library (tables, text inputs, etc.)
- **Lipgloss**: Styling and layout engine

### Directory Structure

```
internal/tui/
├── app/              # Main application logic
│   └── app.go       # App model, routing between views
├── api/              # API client for KECS backend
│   ├── client.go    # HTTP client implementation
│   ├── types.go     # Request/response types
│   └── interface.go # Client interface
├── components/       # Reusable UI components
│   ├── help/        # Help system
│   ├── search/      # Search component
│   └── filter/      # Filter component
├── keys/            # Keyboard shortcuts
│   └── keys.go      # Centralized key bindings
├── styles/          # Visual styling
│   └── styles.go    # Lipgloss styles
└── views/           # View implementations
    ├── clusters/    # Cluster views
    ├── dashboard/   # Dashboard view
    ├── services/    # Service views
    ├── taskdefs/    # Task definition views
    └── tasks/       # Task views
```

## Core Concepts

### The Elm Architecture

Each view follows The Elm Architecture pattern:

```go
type Model struct {
    // View state
}

func (m Model) Init() tea.Cmd {
    // Initial commands
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
    // Handle messages and update state
}

func (m Model) View() string {
    // Render the view
}
```

### Message Passing

Views communicate through messages:

```go
// Internal messages
type tickMsg time.Time
type dataLoadedMsg struct {
    data []SomeType
    err  error
}

// Navigation messages
type ShowDetailsMsg struct {
    ResourceID string
}
```

## Adding a New View

1. Create view directory:
```bash
mkdir -p internal/tui/views/newview
```

2. Implement the model:
```go
package newview

type Model struct {
    client *api.Client
    // ... other fields
}

func New(endpoint string) (*Model, error) {
    // Initialize model
}

func (m *Model) Init() tea.Cmd {
    // Return initial commands
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
    // Handle updates
}

func (m *Model) View() string {
    // Render view
}
```

3. Add to main app:
```go
// In app/app.go
type ViewType int
const (
    // ...
    ViewNewView
)

// Add to App struct
newView *newview.Model

// Initialize in New()
// Handle in Update() switch
// Render in View() switch
```

## Implementing Features

### Search Functionality

1. Add search model to your view:
```go
searchModel search.Model
showSearch  bool
```

2. Initialize in constructor:
```go
searchModel: search.New("Search placeholder..."),
```

3. Handle search key binding:
```go
case keys.Matches(msg, m.keyMap.Search):
    m.showSearch = true
    m.searchModel.SetActive(true)
    return m, nil
```

4. Apply search filter:
```go
filtered := search.Filter(items, m.searchModel.Value(), func(item ItemType) []string {
    return []string{item.Name, item.ID} // Fields to search
})
```

### Filter Functionality

1. Create filter options:
```go
filterOptions := []filter.Option{
    {Label: "Active", Value: "ACTIVE"},
    {Label: "Inactive", Value: "INACTIVE"},
}
```

2. Initialize filter model:
```go
filterModel: filter.New("Filter by Status", filterOptions),
```

3. Apply filters:
```go
filtered := filter.Apply(items, m.filterModel.SelectedValues(), 
    func(item ItemType, values []string) bool {
        // Filter logic
    })
```

### Context-Sensitive Help

1. Set context when view is activated:
```go
a.help.SetContext(help.ContextNewView)
```

2. Add context to help system:
```go
// In components/help/contextual.go
const (
    ContextNewView Context = "new-view"
)

// Add to getShortcuts(), getContextTitle(), etc.
```

## API Integration

### Creating API Methods

1. Define types in `api/types.go`:
```go
type CreateResourceRequest struct {
    Name string `json:"name"`
    // ... other fields
}

type CreateResourceResponse struct {
    Resource Resource `json:"resource"`
}
```

2. Add method to interface:
```go
// In api/interface.go
type APIClient interface {
    CreateResource(ctx context.Context, req *CreateResourceRequest) (*CreateResourceResponse, error)
}
```

3. Implement in client:
```go
func (c *Client) CreateResource(ctx context.Context, req *CreateResourceRequest) (*CreateResourceResponse, error) {
    var resp CreateResourceResponse
    err := c.makeRequest(ctx, "CreateResource", req, &resp)
    return &resp, err
}
```

## Styling Guidelines

Use the predefined styles from `styles/styles.go`:

```go
// Headers and titles
styles.Header.Render("Title")
styles.ListTitle.Render("Section")

// Status indicators
styles.GetStatusStyle("ACTIVE").Render("ACTIVE")

// Information and errors
styles.Info.Render("Information")
styles.Error.Render("Error message")
styles.Success.Render("Success!")

// Interactive elements
styles.SelectedListItem.Render("Selected")
```

## Testing

### Unit Tests

```go
package newview_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("NewView", func() {
    Context("when creating a new model", func() {
        It("should initialize correctly", func() {
            model, err := newview.New("http://localhost:8080")
            Expect(err).NotTo(HaveOccurred())
            Expect(model).NotTo(BeNil())
        })
    })
})
```

### Manual Testing

1. Build and run:
```bash
cd controlplane
go build -o bin/kecs ./cmd/controlplane
./bin/kecs tui
```

2. Test all keyboard shortcuts
3. Verify data updates correctly
4. Check error handling
5. Test with different terminal sizes

## Best Practices

1. **State Management**
   - Keep view state minimal
   - Use messages for state transitions
   - Avoid direct state mutation

2. **Performance**
   - Minimize API calls
   - Use pagination for large lists
   - Cache data when appropriate

3. **Error Handling**
   - Always handle API errors gracefully
   - Show user-friendly error messages
   - Provide recovery options

4. **Accessibility**
   - Support standard keyboard navigation
   - Provide clear visual feedback
   - Include helpful status messages

5. **Code Organization**
   - Keep views focused on single responsibility
   - Reuse components where possible
   - Follow existing patterns

## Debugging

### Enable Debug Logging

```go
// Add debug prints in Update method
fmt.Fprintf(os.Stderr, "DEBUG: msg=%T state=%+v\n", msg, m.someState)
```

### Common Issues

1. **View not updating**
   - Ensure Update() returns the modified model
   - Check if commands are being batched correctly

2. **API calls failing**
   - Verify endpoint is correct
   - Check API response format matches types

3. **Keyboard shortcuts not working**
   - Verify key bindings in keys/keys.go
   - Check for conflicts with terminal

## Contributing

1. Follow the existing code style
2. Add tests for new features
3. Update documentation
4. Run linters before committing:
```bash
go fmt ./...
go vet ./...
```

## Future Enhancements

- WebSocket support for real-time updates
- Customizable themes
- Plugin system for extensions
- Export functionality
- Advanced filtering options
- Batch operations support