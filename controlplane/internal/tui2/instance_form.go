package tui2

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

// FormField represents which field is currently focused
type FormField int

const (
	FieldInstanceName FormField = iota
	FieldAPIPort
	FieldAdminPort
	FieldLocalStack
	FieldTraefik
	FieldDevMode
	FieldSubmit
	FieldCancel
)

// InstanceForm represents the instance creation form state
type InstanceForm struct {
	// Form fields
	instanceName string
	apiPort      string
	adminPort    string
	localStack   bool
	traefik      bool
	devMode      bool
	
	// UI state
	focusedField FormField
	errorMsg     string
	successMsg   string
	
	// Validation state
	nameError    string
	apiPortError string
	adminPortError string
}

// NewInstanceForm creates a new instance form with defaults
func NewInstanceForm() *InstanceForm {
	// Generate a default name
	defaultName, _ := utils.GenerateRandomName()
	
	return &InstanceForm{
		instanceName: defaultName,
		apiPort:      "8080",
		adminPort:    "8081",
		localStack:   true,
		traefik:      true,
		devMode:      false,
		focusedField: FieldInstanceName,
	}
}

// Reset resets the form to default values
func (f *InstanceForm) Reset() {
	defaultName, _ := utils.GenerateRandomName()
	f.instanceName = defaultName
	f.apiPort = "8080"
	f.adminPort = "8081"
	f.localStack = true
	f.traefik = true
	f.devMode = false
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
	if f.focusedField > FieldInstanceName {
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
		f.focusedField = FieldInstanceName
	}
}

// ToggleCheckbox toggles the checkbox at the current focus
func (f *InstanceForm) ToggleCheckbox() {
	switch f.focusedField {
	case FieldLocalStack:
		f.localStack = !f.localStack
	case FieldTraefik:
		f.traefik = !f.traefik
	case FieldDevMode:
		f.devMode = !f.devMode
	}
}

// UpdateField updates the text field at the current focus
func (f *InstanceForm) UpdateField(value string) {
	switch f.focusedField {
	case FieldInstanceName:
		f.instanceName = value
		f.nameError = ""
	case FieldAPIPort:
		f.apiPort = value
		f.apiPortError = ""
	case FieldAdminPort:
		f.adminPort = value
		f.adminPortError = ""
	}
}

// RemoveLastChar removes the last character from the current text field
func (f *InstanceForm) RemoveLastChar() {
	switch f.focusedField {
	case FieldInstanceName:
		if len(f.instanceName) > 0 {
			f.instanceName = f.instanceName[:len(f.instanceName)-1]
		}
	case FieldAPIPort:
		if len(f.apiPort) > 0 {
			f.apiPort = f.apiPort[:len(f.apiPort)-1]
		}
	case FieldAdminPort:
		if len(f.adminPort) > 0 {
			f.adminPort = f.adminPort[:len(f.adminPort)-1]
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
	
	// Validate API port
	apiPort, err := strconv.Atoi(f.apiPort)
	if err != nil || apiPort < 1 || apiPort > 65535 {
		f.apiPortError = "API port must be between 1 and 65535"
		valid = false
	}
	
	// Validate Admin port
	adminPort, err := strconv.Atoi(f.adminPort)
	if err != nil || adminPort < 1 || adminPort > 65535 {
		f.adminPortError = "Admin port must be between 1 and 65535"
		valid = false
	} else if adminPort == apiPort {
		f.adminPortError = "Admin port must be different from API port"
		valid = false
	}
	
	return valid
}

// GetFormData returns the form data for instance creation
func (f *InstanceForm) GetFormData() map[string]interface{} {
	apiPort, _ := strconv.Atoi(f.apiPort)
	adminPort, _ := strconv.Atoi(f.adminPort)
	
	return map[string]interface{}{
		"instanceName": f.instanceName,
		"apiPort":      apiPort,
		"adminPort":    adminPort,
		"localStack":   f.localStack,
		"traefik":      f.traefik,
		"devMode":      f.devMode,
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
		APIPort:  data["apiPort"].(int),
		Age:      0,
	}
	
	f.successMsg = fmt.Sprintf("Instance '%s' created successfully", instance.Name)
	return instance, nil
}

// clearValidationErrors clears all validation errors
func (f *InstanceForm) clearValidationErrors() {
	f.nameError = ""
	f.apiPortError = ""
	f.adminPortError = ""
}

// GetCurrentFieldValue returns the value of the currently focused field
func (f *InstanceForm) GetCurrentFieldValue() string {
	switch f.focusedField {
	case FieldInstanceName:
		return f.instanceName
	case FieldAPIPort:
		return f.apiPort
	case FieldAdminPort:
		return f.adminPort
	default:
		return ""
	}
}