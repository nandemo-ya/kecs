package utils

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NameGen", func() {
	Describe("GenerateRandomName", func() {
		It("should generate names in format 'adjective-noun'", func() {
			names := make(map[string]bool)

			// Test multiple times to ensure randomness
			for i := 0; i < 100; i++ {
				name, err := GenerateRandomName()
				Expect(err).NotTo(HaveOccurred())

				// Check format: should be "adjective-noun"
				parts := strings.Split(name, "-")
				Expect(parts).To(HaveLen(2))

				// Check that adjective and noun are from our lists
				Expect(adjectives).To(ContainElement(parts[0]))
				Expect(nouns).To(ContainElement(parts[1]))

				names[name] = true
			}

			// Check that we got different names (at least 50% unique in 100 attempts)
			Expect(len(names)).To(BeNumerically(">=", 50))
		})
	})

	Describe("GenerateClusterName", func() {
		It("should generate names with 'kecs-' prefix", func() {
			for i := 0; i < 10; i++ {
				name, err := GenerateClusterName()
				Expect(err).NotTo(HaveOccurred())

				// Check that it starts with "kecs-"
				Expect(name).To(HavePrefix("kecs-"))

				// Check format after prefix
				withoutPrefix := strings.TrimPrefix(name, "kecs-")
				parts := strings.Split(withoutPrefix, "-")
				Expect(parts).To(HaveLen(2))
			}
		})
	})

	Describe("GenerateClusterNameWithFallback", func() {
		It("should generate random name with kecs prefix", func() {
			name := GenerateClusterNameWithFallback("fallback")

			// Should have kecs prefix
			Expect(name).To(HavePrefix("kecs-"))

			// The name should have 3 parts: kecs-adjective-noun
			parts := strings.Split(name, "-")
			Expect(parts).To(HaveLen(3))
			Expect(parts[0]).To(Equal("kecs"))
		})
	})
})

var _ = Describe("NameGen Benchmarks", Ordered, func() {
	Describe("Performance", func() {
		It("GenerateRandomName benchmark", func() {
			// Run benchmark within Ginkgo
			names := []string{}
			for i := 0; i < 1000; i++ {
				name, err := GenerateRandomName()
				Expect(err).NotTo(HaveOccurred())
				names = append(names, name)
			}
			Expect(names).To(HaveLen(1000))
		})

		It("GenerateClusterName benchmark", func() {
			// Run benchmark within Ginkgo
			names := []string{}
			for i := 0; i < 1000; i++ {
				name, err := GenerateClusterName()
				Expect(err).NotTo(HaveOccurred())
				names = append(names, name)
			}
			Expect(names).To(HaveLen(1000))
		})
	})
})
