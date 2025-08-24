package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ClusterFormField represents which field is focused
type ClusterFormField int

const (
	FieldClusterName ClusterFormField = iota
	FieldRegion
	FieldCreateButton
	FieldCancelButton
)

// AWS Regions list (commonly used regions)
var awsRegions = []string{
	"us-east-1",      // N. Virginia
	"us-east-2",      // Ohio
	"us-west-1",      // N. California
	"us-west-2",      // Oregon
	"eu-west-1",      // Ireland
	"eu-central-1",   // Frankfurt
	"eu-north-1",     // Stockholm
	"ap-northeast-1", // Tokyo
	"ap-northeast-2", // Seoul
	"ap-southeast-1", // Singapore
	"ap-southeast-2", // Sydney
	"ap-south-1",     // Mumbai
}

// ClusterForm represents the cluster creation form state
type ClusterForm struct {
	// Form inputs
	clusterName textinput.Model
	regionIndex int // Index in awsRegions array

	// Form state
	focusedField ClusterFormField
	nameError    string
	errorMsg     string
	successMsg   string

	// Creation state
	isCreating      bool
	creationSteps   []CreationStep
	creationElapsed string
	creationStart   time.Time
}

// NewClusterForm creates a new cluster creation form
func NewClusterForm() *ClusterForm {
	clusterNameInput := textinput.New()
	clusterNameInput.Placeholder = "Enter cluster name (e.g., my-cluster)"
	clusterNameInput.CharLimit = 255
	clusterNameInput.Width = 35
	clusterNameInput.Focus()

	return &ClusterForm{
		clusterName:  clusterNameInput,
		regionIndex:  0, // Default to us-east-1
		focusedField: FieldClusterName,
	}
}

// Update handles messages for the cluster form
func (f *ClusterForm) Update(msg tea.Msg) (*ClusterForm, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if f.isCreating {
			// Only allow ESC during creation
			if msg.String() == "esc" {
				return nil, nil // Close form
			}
			return f, nil
		}

		switch msg.String() {
		case "esc":
			// Close form without creating
			return nil, nil // Close form

		case "tab":
			// Navigate forward
			f.focusedField = (f.focusedField + 1) % 4
			f.updateFocus()

		case "shift+tab":
			// Navigate backward
			f.focusedField = (f.focusedField - 1 + 4) % 4
			f.updateFocus()

		case "up", "down":
			// Handle region selection
			if f.focusedField == FieldRegion {
				if msg.String() == "up" && f.regionIndex > 0 {
					f.regionIndex--
				} else if msg.String() == "down" && f.regionIndex < len(awsRegions)-1 {
					f.regionIndex++
				}
			}

		case "enter":
			switch f.focusedField {
			case FieldCreateButton:
				// Validate and create
				if f.validate() {
					f.isCreating = true
					f.creationStart = time.Now()
					cmds = append(cmds, f.createCluster())
				}
			case FieldCancelButton:
				return nil, nil // Close form
			}

		default:
			// Pass other keys to text input if focused
			if f.focusedField == FieldClusterName {
				var cmd tea.Cmd
				f.clusterName, cmd = f.clusterName.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case clusterCreatingMsg:
		f.creationSteps = append(f.creationSteps, CreationStep{
			Name:   msg.step,
			Status: "running",
		})
		f.creationElapsed = time.Since(f.creationStart).Round(time.Second).String()

	case clusterCreatedMsg:
		f.isCreating = false
		if msg.err != nil {
			f.errorMsg = msg.err.Error()
			// Mark last step as failed
			if len(f.creationSteps) > 0 {
				f.creationSteps[len(f.creationSteps)-1].Status = "failed"
			}
		} else {
			f.successMsg = fmt.Sprintf("Cluster '%s' created successfully in %s", msg.clusterName, msg.region)
			// Mark all steps as done
			for i := range f.creationSteps {
				if f.creationSteps[i].Status == "running" {
					f.creationSteps[i].Status = "done"
				}
			}
			// Auto-close after success
			cmds = append(cmds, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return clusterFormCloseMsg{}
			}))
		}

	case clusterFormCloseMsg:
		return nil, nil
	}

	return f, tea.Batch(cmds...)
}

// updateFocus updates which input is focused
func (f *ClusterForm) updateFocus() {
	f.clusterName.Blur()
	if f.focusedField == FieldClusterName {
		f.clusterName.Focus()
	}
}

// validate validates the form inputs
func (f *ClusterForm) validate() bool {
	f.nameError = ""
	f.errorMsg = ""

	// Validate cluster name
	name := f.clusterName.Value()
	if name == "" {
		f.nameError = "Cluster name is required"
		return false
	}

	// AWS ECS cluster name validation (simplified)
	// Must be 1-255 characters, can contain letters, numbers, underscores, and hyphens
	if len(name) > 255 {
		f.nameError = "Cluster name must be 255 characters or less"
		return false
	}

	return true
}

// createCluster creates the ECS cluster
func (f *ClusterForm) createCluster() tea.Cmd {
	clusterName := f.clusterName.Value()
	region := awsRegions[f.regionIndex]

	return func() tea.Msg {
		// For now, we'll simulate the creation
		// The actual API call will be made from app.go with access to the API client
		return clusterCreatingMsg{
			step:        "Creating ECS cluster",
			clusterName: clusterName,
			region:      region,
		}
	}
}

// GetClusterName returns the cluster name
func (f *ClusterForm) GetClusterName() string {
	return f.clusterName.Value()
}

// GetRegion returns the selected region
func (f *ClusterForm) GetRegion() string {
	return awsRegions[f.regionIndex]
}

// Message types for cluster creation
type clusterCreatingMsg struct {
	step        string
	clusterName string
	region      string
}

type clusterCreatedMsg struct {
	clusterName string
	region      string
	err         error
}

type clusterFormCloseMsg struct{}
