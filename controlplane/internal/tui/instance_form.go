package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

// FormField represents which field is currently focused
type FormField int

const (
	FieldInstanceCloseButton FormField = iota // × button at top-right
	FieldInstanceName
	FieldAdditionalServices
	FieldSubmit
	FieldCancel
)

// CreationStep represents each step in instance creation
type CreationStep struct {
	Name   string
	Status string // "pending", "running", "done", "failed"
}

// InstanceForm represents the instance creation form state
type InstanceForm struct {
	// Form fields
	instanceName       string
	additionalServices string

	// UI state
	focusedField FormField
	errorMsg     string
	successMsg   string
	isCreating   bool

	// Creation status tracking
	creationSteps     []CreationStep
	creationElapsed   string    // Time elapsed (deprecated - use elapsedTime)
	showTimeoutPrompt bool      // Show continue/abort prompt on timeout
	startTime         time.Time // When creation started
	elapsedTime       string    // Time elapsed as string

	// Validation state
	nameError     string
	servicesError string
}

// NewInstanceForm creates a new instance form with defaults
func NewInstanceForm() *InstanceForm {
	// Generate a default name
	defaultName, _ := utils.GenerateRandomName()

	return &InstanceForm{
		instanceName:       defaultName,
		additionalServices: "",
		focusedField:       FieldInstanceName, // Start with instance name field, not close button
	}
}

// NewInstanceFormWithSuggestions creates a new instance form
func NewInstanceFormWithSuggestions(instances []Instance) *InstanceForm {
	// Generate a default name
	defaultName, _ := utils.GenerateRandomName()

	return &InstanceForm{
		instanceName:       defaultName,
		additionalServices: "",
		focusedField:       FieldInstanceName, // Start with instance name field, not close button
	}
}

// Reset resets the form to default values
func (f *InstanceForm) Reset() {
	defaultName, _ := utils.GenerateRandomName()
	f.instanceName = defaultName
	f.focusedField = FieldInstanceName
	f.errorMsg = ""
	f.successMsg = ""
	f.clearValidationErrors()
}

// GenerateNewName generates a new random instance name
func (f *InstanceForm) GenerateNewName() {
	name, _ := utils.GenerateRandomName()
	f.instanceName = name
	f.nameError = ""
}

// MoveFocusUp moves focus to the previous field
func (f *InstanceForm) MoveFocusUp() {
	if f.focusedField > FieldInstanceCloseButton {
		f.focusedField--
	} else {
		f.focusedField = FieldCancel
	}
}

// MoveFocusDown moves focus to the next field
func (f *InstanceForm) MoveFocusDown() {
	if f.focusedField < FieldCancel {
		f.focusedField++
	} else {
		f.focusedField = FieldInstanceCloseButton
	}
}

// ToggleCheckbox toggles the checkbox at the current focus
func (f *InstanceForm) ToggleCheckbox() {
	// No checkboxes anymore
}

// UpdateField updates the text field at the current focus
func (f *InstanceForm) UpdateField(value string) {
	switch f.focusedField {
	case FieldInstanceName:
		f.instanceName = value
		f.nameError = ""
	}
}

// RemoveLastChar removes the last character from the current text field
func (f *InstanceForm) RemoveLastChar() {
	switch f.focusedField {
	case FieldInstanceName:
		if len(f.instanceName) > 0 {
			f.instanceName = f.instanceName[:len(f.instanceName)-1]
		}
	case FieldAdditionalServices:
		if len(f.additionalServices) > 0 {
			f.additionalServices = f.additionalServices[:len(f.additionalServices)-1]
		}
	}
}

// Validate validates all form fields
func (f *InstanceForm) Validate() bool {
	f.clearValidationErrors()
	valid := true

	// Validate instance name
	if strings.TrimSpace(f.instanceName) == "" {
		f.nameError = "Instance name is required"
		valid = false
	} else if len(f.instanceName) > 50 {
		f.nameError = "Instance name must be 50 characters or less"
		valid = false
	}

	// Validate additional services (optional)
	if f.additionalServices != "" {
		// Simple validation: check for basic format
		services := strings.Split(f.additionalServices, ",")
		for _, service := range services {
			service = strings.TrimSpace(service)
			if service != "" && !isValidServiceName(service) {
				f.servicesError = "Invalid service name: " + service
				valid = false
				break
			}
		}
	}

	// Ports are now automatically allocated, no validation needed

	return valid
}

// GetFormData returns the form data for instance creation
func (f *InstanceForm) GetFormData() map[string]interface{} {
	// Ports will be automatically allocated
	return map[string]interface{}{
		"instanceName":       f.instanceName,
		"apiPort":            0,    // 0 means auto-allocate
		"adminPort":          0,    // 0 means auto-allocate
		"localStack":         true, // Always enabled
		"additionalServices": f.additionalServices,
	}
}

// CreateMockInstance creates a mock instance (for testing)
func (f *InstanceForm) CreateMockInstance() (*Instance, error) {
	if !f.Validate() {
		return nil, fmt.Errorf("validation failed")
	}

	data := f.GetFormData()

	// Create mock instance
	instance := &Instance{
		Name:     data["instanceName"].(string),
		Status:   "ACTIVE",
		Clusters: 0,
		Services: 0,
		Tasks:    0,
		APIPort:  5373, // Default port for mock
		Age:      0,
	}

	f.successMsg = fmt.Sprintf("Instance '%s' created successfully", instance.Name)
	return instance, nil
}

// clearValidationErrors clears all validation errors
func (f *InstanceForm) clearValidationErrors() {
	f.nameError = ""
	f.servicesError = ""
}

// GetCurrentFieldValue returns the value of the currently focused field
func (f *InstanceForm) GetCurrentFieldValue() string {
	switch f.focusedField {
	case FieldInstanceName:
		return f.instanceName
	case FieldAdditionalServices:
		return f.additionalServices
	default:
		return ""
	}
}

// isValidServiceName checks if a service name is valid
func isValidServiceName(service string) bool {
	// Allow alphanumeric characters, hyphens, and underscores
	// This is a basic validation - LocalStack will do more thorough validation
	if len(service) == 0 || len(service) > 50 {
		return false
	}
	for _, ch := range service {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}
