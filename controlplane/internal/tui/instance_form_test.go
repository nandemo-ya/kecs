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
			Expect(data["apiPort"]).To(Equal(0))    // Auto-allocate
			Expect(data["adminPort"]).To(Equal(0))  // Auto-allocate
			Expect(data["localStack"]).To(BeTrue()) // Always true
		})
	})

	Describe("Field Navigation", func() {
		It("should move focus down through fields", func() {
			// Start at instance name and capture value
			initialValue := form.GetCurrentFieldValue()
			Expect(initialValue).NotTo(BeEmpty()) // Should have random name

			// Navigate down
			form.MoveFocusDown() // Should move to Submit

			// Navigate back up
			form.MoveFocusUp() // Should return to instance name

			// Verify we're back at instance name
			currentValue := form.GetCurrentFieldValue()
			Expect(currentValue).To(Equal(initialValue))
		})

		It("should move focus up through fields", func() {
			// Start at instance name field
			form.MoveFocusUp() // Moves to close button (wraps to Cancel)
			// Since we removed API/Admin port fields, navigation should be simpler
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

		It("should validate instance name", func() {
			// Clear instance name by repeatedly calling RemoveLastChar
			for i := 0; i < 50; i++ {
				form.RemoveLastChar()
			}
			valid := form.Validate()
			Expect(valid).To(BeFalse())

			form.UpdateField("test-instance")
			valid = form.Validate()
			Expect(valid).To(BeTrue())
		})

		// Port validation is no longer needed as ports are auto-allocated

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
			Expect(instance.APIPort).To(Equal(5373)) // Default mock port
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

			// Reset
			form.Reset()

			// Check defaults restored
			data := form.GetFormData()
			Expect(data["instanceName"]).NotTo(Equal("custom-name"))
			Expect(data["apiPort"]).To(Equal(0)) // Auto-allocate
		})
	})
})
