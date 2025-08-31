package api

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server Shutdown", func() {
	var (
		server *Server
		ctx    context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create server without ClusterManager
		server = &Server{
			httpServer: &http.Server{
				Addr: ":8080",
			},
		}
	})

	Context("when shutting down", func() {
		It("should shutdown gracefully", func() {
			// Create a context with timeout for shutdown
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			// Server should be able to stop even without storage or cluster manager
			err := server.Stop(shutdownCtx)
			Expect(err).To(BeNil())
		})
	})

	// ClusterManager-related tests have been removed as ClusterManager is deprecated
})
