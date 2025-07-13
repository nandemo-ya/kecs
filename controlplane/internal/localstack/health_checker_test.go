package localstack_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

var _ = Describe("HealthChecker", func() {
	var (
		healthChecker localstack.HealthChecker
		mockServer    *httptest.Server
		healthCalled  bool
	)

	BeforeEach(func() {
		healthCalled = false
		// Create mock LocalStack server
		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == localstack.HealthCheckPath {
				healthCalled = true
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"status": "ok"}`)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		healthChecker = localstack.NewHealthChecker(mockServer.URL)
	})

	AfterEach(func() {
		mockServer.Close()
	})

	Context("UpdateEndpoint", func() {
		It("should update the endpoint and use the new endpoint for health checks", func() {
			// Initial health check should succeed
			ctx := context.Background()
			status, err := healthChecker.CheckHealth(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Healthy).To(BeTrue())
			Expect(healthCalled).To(BeTrue())

			// Reset flag
			healthCalled = false

			// Create a new mock server to simulate different endpoint
			newMockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == localstack.HealthCheckPath {
					healthCalled = true
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, `{"status": "ok", "services": {}}`)
				}
			}))
			defer newMockServer.Close()

			// Update endpoint
			healthChecker.UpdateEndpoint(newMockServer.URL)

			// Health check should now use the new endpoint
			status, err = healthChecker.CheckHealth(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Healthy).To(BeTrue())
			Expect(healthCalled).To(BeTrue())
		})

		It("should handle connection failures gracefully", func() {
			// Update to invalid endpoint
			healthChecker.UpdateEndpoint("http://localhost:1234")

			ctx := context.Background()
			status, err := healthChecker.CheckHealth(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Healthy).To(BeFalse())
			Expect(status.Message).To(ContainSubstring("failed to connect"))
		})
	})

	Context("WaitForHealthy", func() {
		It("should wait for LocalStack to become healthy", func() {
			ctx := context.Background()
			err := healthChecker.WaitForHealthy(ctx, 5*time.Second)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should timeout if LocalStack doesn't become healthy", func() {
			// Update to endpoint that will fail
			healthChecker.UpdateEndpoint("http://localhost:1234")

			ctx := context.Background()
			err := healthChecker.WaitForHealthy(ctx, 2*time.Second)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("timeout"))
		})
	})
})