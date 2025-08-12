package instance_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/instance"
)

func TestManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instance Manager Suite")
}

var _ = Describe("Manager", func() {
	var (
		manager *instance.Manager
		ctx     context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		manager, err = instance.NewManager()
		Expect(err).NotTo(HaveOccurred())
		Expect(manager).NotTo(BeNil())
	})

	Describe("List", func() {
		It("should return list of instances", func() {
			instances, err := manager.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			// Test should pass whether instances exist or not
			Expect(instances).NotTo(BeNil())
		})
	})

	Describe("Stop", func() {
		It("should return error when stopping non-existent instance", func() {
			err := manager.Stop(ctx, "non-existent-instance")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})

	Describe("Destroy", func() {
		It("should return error when destroying non-existent instance", func() {
			err := manager.Destroy(ctx, "non-existent-instance", false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})

	Describe("IsRunning", func() {
		It("should return false for non-existent instance", func() {
			running, err := manager.IsRunning(ctx, "non-existent-instance")
			Expect(err).NotTo(HaveOccurred())
			Expect(running).To(BeFalse())
		})
	})

	Describe("Restart", func() {
		It("should return error when restarting non-existent instance", func() {
			err := manager.Restart(ctx, "non-existent-instance")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})

	Describe("GetCreationStatus", func() {
		It("should return nil for non-tracked instance", func() {
			status := manager.GetCreationStatus("non-existent-instance")
			Expect(status).To(BeNil())
		})
	})
})
