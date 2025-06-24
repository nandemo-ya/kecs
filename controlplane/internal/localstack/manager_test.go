package localstack_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"
	"time"

	"k8s.io/client-go/kubernetes/fake"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

var _ = Describe("LocalStack Manager", func() {
	var (
		manager    localstack.Manager
		kubeClient *fake.Clientset
		config     *localstack.Config
		ctx        context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		kubeClient = fake.NewSimpleClientset()

		config = localstack.DefaultConfig()
		config.Enabled = true
		config.Services = []string{"iam", "logs", "ssm"}

		var err error
		manager, err = localstack.NewManager(config, kubeClient)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Configuration", func() {
		Context("when creating with valid config", func() {
			It("should create manager successfully", func() {
				Expect(manager).NotTo(BeNil())
			})
		})

		Context("when creating with invalid config", func() {
			It("should return error for nil config", func() {
				_, err := localstack.NewManager(nil, kubeClient)
				Expect(err).To(HaveOccurred())
			})

			It("should return error for invalid service", func() {
				config.Services = []string{"invalid-service"}
				_, err := localstack.NewManager(config, kubeClient)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Service Management", func() {
		Context("when getting enabled services", func() {
			It("should return configured services", func() {
				services, err := manager.GetEnabledServices()
				Expect(err).NotTo(HaveOccurred())
				Expect(services).To(ConsistOf("iam", "logs", "ssm"))
			})
		})

		Context("when updating services", func() {
			It("should update services successfully", func() {
				newServices := []string{"iam", "s3"}
				err := manager.UpdateServices(newServices)
				Expect(err).NotTo(HaveOccurred())

				services, err := manager.GetEnabledServices()
				Expect(err).NotTo(HaveOccurred())
				Expect(services).To(ConsistOf("iam", "s3"))
			})

			It("should reject invalid services", func() {
				err := manager.UpdateServices([]string{"invalid"})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Health Check", func() {
		Context("when LocalStack is not started", func() {
			It("should report as not healthy", func() {
				Expect(manager.IsHealthy()).To(BeFalse())
			})
		})

		Context("when checking status", func() {
			It("should return status", func() {
				status, err := manager.GetStatus()
				Expect(err).NotTo(HaveOccurred())
				Expect(status).NotTo(BeNil())
				Expect(status.Running).To(BeFalse())
				Expect(status.Healthy).To(BeFalse())
			})
		})
	})

	Describe("Endpoint Management", func() {
		Context("when LocalStack is not running", func() {
			It("should return error for endpoint", func() {
				_, err := manager.GetEndpoint()
				Expect(err).To(HaveOccurred())
			})

			It("should return error for service endpoint", func() {
				_, err := manager.GetServiceEndpoint("s3")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Lifecycle Management", func() {
		Context("when waiting for ready", func() {
			It("should timeout if not running", func() {
				err := manager.WaitForReady(ctx, 100*time.Millisecond)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
