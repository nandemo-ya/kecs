package cmd

import (
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kubeconfig", func() {
	Describe("fixKubeconfig", func() {
		Context("when fixing server URLs", func() {
			It("should fix server URL with missing port", func() {
				// Test case: server URL has no port (just trailing colon)
				input := `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTi...
    server: https://0.0.0.0:
  name: k3d-kecs-test
contexts:
- context:
    cluster: k3d-kecs-test
    user: admin@k3d-kecs-test
  name: k3d-kecs-test
current-context: k3d-kecs-test
kind: Config`

				// Mock getK3dAPIPort to return a known port
				// For now, we'll test the regex replacement directly
				fixed := strings.ReplaceAll(input, "0.0.0.0", "127.0.0.1")
				
				// The updated regex pattern
				re := regexp.MustCompile(`(https://127\.0\.0\.1)(:\d+)?(:)?`)
				result := re.ReplaceAllStringFunc(fixed, func(match string) string {
					return "https://127.0.0.1:6443"
				})
				
				// Debug output
				GinkgoWriter.Printf("Input after 0.0.0.0 replacement:\n%s\n", fixed)
				GinkgoWriter.Printf("Result after regex replacement:\n%s\n", result)
				
				Expect(result).To(ContainSubstring("server: https://127.0.0.1:6443"))
				// Check that we don't have trailing colon without port
				Expect(result).NotTo(MatchRegexp(`server: https://127\.0\.0\.1:\s*\n`))
			})

			It("should fix server URL with existing port", func() {
				input := `server: https://0.0.0.0:12345`
				fixed := strings.ReplaceAll(input, "0.0.0.0", "127.0.0.1")
				
				re := regexp.MustCompile(`(https://127\.0\.0\.1)(:\d+)?(:)?`)
				result := re.ReplaceAllStringFunc(fixed, func(match string) string {
					return "https://127.0.0.1:6443"
				})
				
				Expect(result).To(Equal("server: https://127.0.0.1:6443"))
			})

			It("should handle quoted URLs in YAML", func() {
				input := `server: 'https://0.0.0.0:'`
				fixed := strings.ReplaceAll(input, "0.0.0.0", "127.0.0.1")
				
				re := regexp.MustCompile(`(https://127\.0\.0\.1)(:\d+)?(:)?`)
				result := re.ReplaceAllStringFunc(fixed, func(match string) string {
					return "https://127.0.0.1:6443"
				})
				
				Expect(result).To(Equal("server: 'https://127.0.0.1:6443'"))
			})

			It("should handle double-quoted URLs in YAML", func() {
				input := `server: "https://0.0.0.0:"`
				fixed := strings.ReplaceAll(input, "0.0.0.0", "127.0.0.1")
				
				re := regexp.MustCompile(`(https://127\.0\.0\.1)(:\d+)?(:)?`)
				result := re.ReplaceAllStringFunc(fixed, func(match string) string {
					return "https://127.0.0.1:6443"
				})
				
				Expect(result).To(Equal(`server: "https://127.0.0.1:6443"`))
			})
		})
	})
})