package tui_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/tui"
)

var _ = Describe("InstanceForm", func() {
	var form *tui.InstanceForm

	BeforeEach(func() {
		form = tui.NewInstanceForm()
	})

	Describe("NewInstanceForm", func() {
		It("should create form with default values", func() {
			Expect(form).NotTo(BeNil())
			// Default values should be set
			data := form.GetFormData()
			Expect(data["instanceName"]).NotTo(BeEmpty())
			Expect(data["apiPort"]).To(Equal(8080))
			Expect(data["adminPort"]).To(Equal(8081))
			Expect(data["localStack"]).To(BeTrue()) // Always true
		})
	})

	Describe("Field Navigation", func() {
		It("should move focus down through fields", func() {
			// Start at instance name
			form.MoveFocusDown() // API port
			form.MoveFocusDown() // Admin port
			form.MoveFocusDown() // Submit
			form.MoveFocusDown() // Cancel
			form.MoveFocusDown() // Back to close button
			// Should wrap around to close button (not a text field, so value is empty)
			form.MoveFocusDown() // Now at instance name
			Expect(form.GetCurrentFieldValue()).NotTo(BeEmpty())
		})

		It("should move focus up through fields", func() {
			form.MoveFocusUp() // From instance name to cancel
			form.MoveFocusUp() // Submit
			form.MoveFocusUp() // Admin port
			// Continue cycling
		})
	})

	Describe("Text Input", func() {
		It("should update instance name field", func() {
			form.UpdateField("test-instance")
			data := form.GetFormData()
			Expect(data["instanceName"]).To(Equal("test-instance"))
		})

		It("should remove last character on backspace", func() {
			form.UpdateField("test")
			form.RemoveLastChar()
			data := form.GetFormData()
			Expect(data["instanceName"]).To(Equal("tes"))
		})
	})

	// Checkbox tests removed - LocalStack is always enabled, no checkboxes in form

	Describe("Validation", func() {
		It("should validate empty instance name", func() {
			form.UpdateField("")
			valid := form.Validate()
			Expect(valid).To(BeFalse())
		})

		It("should validate invalid port numbers", func() {
			// Move to API port
			form.MoveFocusDown()
			form.UpdateField("invalid")
			valid := form.Validate()
			Expect(valid).To(BeFalse())
		})

		It("should validate port conflicts", func() {
			// Set same port for both
			form.MoveFocusDown() // API port
			form.UpdateField("8080")
			form.MoveFocusDown() // Admin port
			form.UpdateField("8080")
			valid := form.Validate()
			Expect(valid).To(BeFalse())
		})

		It("should pass validation with valid data", func() {
			valid := form.Validate()
			Expect(valid).To(BeTrue())
		})
	})

	Describe("Mock Instance Creation", func() {
		It("should create mock instance with valid data", func() {
			instance, err := form.CreateMockInstance()
			Expect(err).NotTo(HaveOccurred())
			Expect(instance).NotTo(BeNil())
			Expect(instance.Status).To(Equal("ACTIVE"))
			Expect(instance.APIPort).To(Equal(8080))
		})

		It("should fail with invalid data", func() {
			form.UpdateField("")
			instance, err := form.CreateMockInstance()
			Expect(err).To(HaveOccurred())
			Expect(instance).To(BeNil())
		})
	})

	Describe("Reset", func() {
		It("should reset form to defaults", func() {
			// Modify form
			form.UpdateField("custom-name")
			form.MoveFocusDown()
			form.UpdateField("9999")

			// Reset
			form.Reset()

			// Check defaults restored
			data := form.GetFormData()
			Expect(data["instanceName"]).NotTo(Equal("custom-name"))
			Expect(data["apiPort"]).To(Equal(8080))
		})
	})
})
