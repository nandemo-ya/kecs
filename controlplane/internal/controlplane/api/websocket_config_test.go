package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WebSocketConfig", func() {
	Context("when checking origin", func() {
		DescribeTable("origin validation scenarios",
			func(config *WebSocketConfig, origin, referer, host string, expectedResult bool) {
				// Create a test request
				req := httptest.NewRequest("GET", "/ws", nil)

				// Set headers
				if origin != "" {
					req.Header.Set("Origin", origin)
				}
				if referer != "" {
					req.Header.Set("Referer", referer)
				}
				if host != "" {
					req.Host = host
				}

				// Check origin
				result := config.CheckOrigin(req)
				Expect(result).To(Equal(expectedResult))
			},
			Entry("empty allowed origins allows all",
				&WebSocketConfig{
					AllowedOrigins: []string{},
				},
				"http://example.com", "", "",
				true,
			),
			Entry("wildcard allows all origins",
				&WebSocketConfig{
					AllowedOrigins: []string{"*"},
				},
				"http://malicious.com", "", "",
				true,
			),
			Entry("exact match allowed",
				&WebSocketConfig{
					AllowedOrigins: []string{"http://localhost:3000", "https://app.example.com"},
				},
				"http://localhost:3000", "", "",
				true,
			),
			Entry("exact match denied",
				&WebSocketConfig{
					AllowedOrigins: []string{"http://localhost:3000"},
				},
				"http://localhost:3001", "", "",
				false,
			),
			Entry("scheme mismatch denied",
				&WebSocketConfig{
					AllowedOrigins: []string{"https://example.com"},
				},
				"http://example.com", "", "",
				false,
			),
			Entry("wildcard subdomain match",
				&WebSocketConfig{
					AllowedOrigins: []string{"*.example.com"},
				},
				"https://app.example.com", "", "",
				true,
			),
			Entry("wildcard subdomain with nested subdomain",
				&WebSocketConfig{
					AllowedOrigins: []string{"*.example.com"},
				},
				"https://api.app.example.com", "", "",
				true,
			),
			Entry("wildcard subdomain root domain denied",
				&WebSocketConfig{
					AllowedOrigins: []string{"*.example.com"},
				},
				"https://example.com", "", "",
				false,
			),
			Entry("scheme-less match",
				&WebSocketConfig{
					AllowedOrigins: []string{"localhost:3000"},
				},
				"http://localhost:3000", "", "",
				true,
			),
			Entry("same origin with no origin header",
				&WebSocketConfig{
					AllowedOrigins: []string{"http://example.com"},
				},
				"", "http://localhost:8080/page", "localhost:8080",
				true,
			),
			Entry("different origin with no origin header",
				&WebSocketConfig{
					AllowedOrigins: []string{"http://example.com"},
				},
				"", "http://example.com/page", "localhost:8080",
				false,
			),
			Entry("custom check function overrides",
				&WebSocketConfig{
					AllowedOrigins: []string{"http://denied.com"},
					CheckOriginFunc: func(r *http.Request) bool {
						// Custom function that always allows
						return true
					},
				},
				"http://custom.com", "", "",
				true,
			),
			Entry("multiple allowed origins",
				&WebSocketConfig{
					AllowedOrigins: []string{
						"http://localhost:3000",
						"http://localhost:8080",
						"https://production.app.com",
						"*.staging.app.com",
					},
				},
				"https://api.staging.app.com", "", "",
				true,
			),
		)
	})

	Context("when matching origin patterns", func() {
		var config *WebSocketConfig

		BeforeEach(func() {
			config = &WebSocketConfig{}
		})

		DescribeTable("origin matching scenarios",
			func(origin, allowed string, expected bool) {
				originURL, err := parseURL(origin)
				Expect(err).NotTo(HaveOccurred())

				result := config.matchOrigin(originURL, allowed)
				Expect(result).To(Equal(expected))
			},
			Entry("wildcard matches anything", "http://example.com", "*", true),
			Entry("exact match", "http://example.com", "http://example.com", true),
			Entry("port mismatch", "http://example.com:8080", "http://example.com", false),
			Entry("subdomain wildcard match", "https://app.example.com", "*.example.com", true),
			Entry("subdomain wildcard no match on root", "https://example.com", "*.example.com", false),
			Entry("scheme-less match with http", "http://localhost:3000", "localhost:3000", true),
			Entry("scheme-less match with https", "https://localhost:3000", "localhost:3000", true),
		)
	})

	Context("when using default configuration", func() {
		It("should have expected default values", func() {
			config := DefaultWebSocketConfig()

			Expect(config).NotTo(BeNil())
			Expect(config.AllowedOrigins).To(BeEmpty())
			Expect(config.AllowCredentials).To(BeTrue())
			Expect(config.CheckOriginFunc).To(BeNil())
		})

		It("should allow all origins by default", func() {
			config := DefaultWebSocketConfig()

			req := httptest.NewRequest("GET", "/ws", nil)
			req.Header.Set("Origin", "http://any-origin.com")

			Expect(config.CheckOrigin(req)).To(BeTrue())
		})
	})
})

// Helper function to parse URL for tests
func parseURL(urlStr string) (*url.URL, error) {
	return url.Parse(urlStr)
}
