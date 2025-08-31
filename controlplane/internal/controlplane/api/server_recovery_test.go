package api

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage/memory"
)

var _ = Describe("Server Recovery", func() {
	var (
		server  *Server
		ctx     context.Context
		storage *memory.MemoryStorage
	)

	BeforeEach(func() {
		ctx = context.Background()
		storage = memory.NewMemoryStorage()

		// Initialize storage
		err := storage.Initialize(ctx)
		Expect(err).To(BeNil())

		// Create server without ClusterManager
		server = &Server{
			storage: storage,
		}
	})

	Context("when recovering state", func() {
		It("should skip recovery when kube client is not available", func() {
			// Server has no kube client, so recovery should be skipped
			err := server.RecoverState(ctx)
			Expect(err).To(BeNil())
		})
	})

	// ClusterManager-related recovery tests have been removed as ClusterManager is deprecated
})
