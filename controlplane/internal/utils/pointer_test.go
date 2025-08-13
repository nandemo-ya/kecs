package utils_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

var _ = Describe("Pointer Utils", func() {
	Describe("Ptr", func() {
		It("should return a pointer to string", func() {
			str := "hello"
			ptr := utils.Ptr(str)
			Expect(ptr).NotTo(BeNil())
			Expect(*ptr).To(Equal("hello"))
		})

		It("should return a pointer to int", func() {
			num := 42
			ptr := utils.Ptr(num)
			Expect(ptr).NotTo(BeNil())
			Expect(*ptr).To(Equal(42))
		})

		It("should return a pointer to bool", func() {
			b := true
			ptr := utils.Ptr(b)
			Expect(ptr).NotTo(BeNil())
			Expect(*ptr).To(BeTrue())
		})

		It("should return a pointer to struct", func() {
			type TestStruct struct {
				Name  string
				Value int
			}
			s := TestStruct{Name: "test", Value: 100}
			ptr := utils.Ptr(s)
			Expect(ptr).NotTo(BeNil())
			Expect(ptr.Name).To(Equal("test"))
			Expect(ptr.Value).To(Equal(100))
		})
	})

	Describe("Deref", func() {
		It("should dereference a string pointer", func() {
			str := "hello"
			ptr := &str
			result := utils.Deref(ptr)
			Expect(result).To(Equal("hello"))
		})

		It("should return empty string for nil string pointer", func() {
			var ptr *string
			result := utils.Deref(ptr)
			Expect(result).To(Equal(""))
		})

		It("should return 0 for nil int pointer", func() {
			var ptr *int
			result := utils.Deref(ptr)
			Expect(result).To(Equal(0))
		})

		It("should return false for nil bool pointer", func() {
			var ptr *bool
			result := utils.Deref(ptr)
			Expect(result).To(BeFalse())
		})
	})

	Describe("DerefOrDefault", func() {
		It("should dereference a string pointer", func() {
			str := "hello"
			ptr := &str
			result := utils.DerefOrDefault(ptr, "default")
			Expect(result).To(Equal("hello"))
		})

		It("should return default for nil string pointer", func() {
			var ptr *string
			result := utils.DerefOrDefault(ptr, "default")
			Expect(result).To(Equal("default"))
		})

		It("should return default for nil int pointer", func() {
			var ptr *int
			result := utils.DerefOrDefault(ptr, 99)
			Expect(result).To(Equal(99))
		})

		It("should return default for nil bool pointer", func() {
			var ptr *bool
			result := utils.DerefOrDefault(ptr, true)
			Expect(result).To(BeTrue())
		})
	})
})
