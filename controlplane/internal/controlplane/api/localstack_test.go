package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
)

var _ = Describe("LocalStack API", func() {
	var (
		server      *api.Server
		testServer  *httptest.Server
		originalTestMode string
	)

	BeforeEach(func() {
		// Set test mode to avoid Kubernetes initialization issues
		originalTestMode = os.Getenv("KECS_TEST_MODE")
		os.Setenv("KECS_TEST_MODE", "true")
		
		// Create server without LocalStack manager to test disabled state
		var err error
		server, err = api.NewServer(8080, "", nil, nil)
		Expect(err).NotTo(HaveOccurred())
		
		// Create a handler that properly routes the request
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Route based on path
			if r.URL.Path == "/api/localstack/status" {
				server.GetLocalStackStatus(w, r)
			} else {
				http.NotFound(w, r)
			}
		})
		testServer = httptest.NewServer(handler)
	})

	AfterEach(func() {
		testServer.Close()
		
		// Restore original test mode setting
		if originalTestMode == "" {
			os.Unsetenv("KECS_TEST_MODE")
		} else {
			os.Setenv("KECS_TEST_MODE", originalTestMode)
		}
	})

	Describe("LocalStack Status Endpoint", func() {
		It("should return disabled status when LocalStack is not configured", func() {
			resp, err := http.Get(testServer.URL + "/api/localstack/status")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var status api.LocalStackStatus
			err = json.NewDecoder(resp.Body).Decode(&status)
			Expect(err).NotTo(HaveOccurred())

			Expect(status.Enabled).To(BeFalse())
			Expect(status.Running).To(BeFalse())
			Expect(status.Services).To(BeEmpty())
			Expect(status.ProxyEnabled).To(BeFalse())
		})

		It("should reject non-GET requests", func() {
			resp, err := http.Post(testServer.URL+"/api/localstack/status", "application/json", nil)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
		})
	})
})