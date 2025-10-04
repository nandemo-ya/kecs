package servicediscovery

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CoreDNS", func() {
	Describe("escapeRegex", func() {
		Context("when escaping basic domain names", func() {
			It("should escape dots in domain names", func() {
				input := "demo.local"
				expected := "demo\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should escape dots in subdomain names", func() {
				input := "api.demo.local"
				expected := "api\\.demo\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})
		})

		Context("when escaping regex special characters", func() {
			It("should escape square brackets", func() {
				input := "test[special].local"
				expected := "test\\[special\\]\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should escape parentheses", func() {
				input := "app(prod).local"
				expected := "app\\(prod\\)\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should escape asterisks", func() {
				input := "service*.local"
				expected := "service\\*\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should escape plus signs", func() {
				input := "env+dev.local"
				expected := "env\\+dev\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should escape question marks", func() {
				input := "test?.local"
				expected := "test\\?\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should escape pipes", func() {
				input := "test|prod.local"
				expected := "test\\|prod\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should escape carets", func() {
				input := "api^v1.local"
				expected := "api\\^v1\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should escape dollar signs", func() {
				input := "end$.local"
				expected := "end\\$\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should escape curly braces", func() {
				input := "test{prod}.local"
				expected := "test\\{prod\\}\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should escape backslashes", func() {
				input := "test\\escape.local"
				expected := "test\\\\escape\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})
		})

		Context("when escaping multiple special characters", func() {
			It("should escape all special characters in complex strings", func() {
				input := "test[prod]*.api(v1).local"
				expected := "test\\[prod\\]\\*\\.api\\(v1\\)\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should handle strings with multiple dots and brackets", func() {
				input := "a.b[c].d.local"
				expected := "a\\.b\\[c\\]\\.d\\.local"
				Expect(escapeRegex(input)).To(Equal(expected))
			})
		})

		Context("when handling edge cases", func() {
			It("should return empty string for empty input", func() {
				Expect(escapeRegex("")).To(Equal(""))
			})

			It("should handle strings with no special characters", func() {
				input := "simple"
				expected := "simple"
				Expect(escapeRegex(input)).To(Equal(expected))
			})

			It("should handle strings with only special characters", func() {
				input := ".*+?[](){}^$|\\"
				expected := "\\.\\*\\+\\?\\[\\]\\(\\)\\{\\}\\^\\$\\|\\\\"
				Expect(escapeRegex(input)).To(Equal(expected))
			})
		})
	})
})
