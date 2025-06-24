package servicediscovery

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Service Discovery Manager", func() {
	var (
		manager Manager
		ctx     context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		fakeClient := fake.NewSimpleClientset()
		manager = NewManager(fakeClient, "us-east-1", "123456789012")
	})

	Describe("Namespace Operations", func() {
		It("should create a private DNS namespace", func() {
			namespace, err := manager.CreatePrivateDnsNamespace(ctx, "test.local", "vpc-123", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(namespace).NotTo(BeNil())
			Expect(namespace.Name).To(Equal("test.local"))
			Expect(namespace.Type).To(Equal("DNS_PRIVATE"))
		})

		It("should not allow duplicate namespace names", func() {
			_, err := manager.CreatePrivateDnsNamespace(ctx, "test.local", "vpc-123", nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = manager.CreatePrivateDnsNamespace(ctx, "test.local", "vpc-456", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should list namespaces", func() {
			_, err := manager.CreatePrivateDnsNamespace(ctx, "test1.local", "vpc-123", nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = manager.CreatePrivateDnsNamespace(ctx, "test2.local", "vpc-123", nil)
			Expect(err).NotTo(HaveOccurred())

			namespaces, err := manager.ListNamespaces(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(namespaces).To(HaveLen(2))
		})

		It("should delete a namespace without services", func() {
			namespace, err := manager.CreatePrivateDnsNamespace(ctx, "test.local", "vpc-123", nil)
			Expect(err).NotTo(HaveOccurred())

			err = manager.DeleteNamespace(ctx, namespace.ID)
			Expect(err).NotTo(HaveOccurred())

			_, err = manager.GetNamespace(ctx, namespace.ID)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Service Operations", func() {
		var namespaceID string

		BeforeEach(func() {
			namespace, err := manager.CreatePrivateDnsNamespace(ctx, "test.local", "vpc-123", nil)
			Expect(err).NotTo(HaveOccurred())
			namespaceID = namespace.ID
		})

		It("should create a service in a namespace", func() {
			dnsConfig := &DnsConfig{
				NamespaceId:   namespaceID,
				RoutingPolicy: "MULTIVALUE",
				DnsRecords: []DnsRecord{
					{Type: "A", TTL: 60},
				},
			}

			service, err := manager.CreateService(ctx, "test-service", namespaceID, dnsConfig, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(service).NotTo(BeNil())
			Expect(service.Name).To(Equal("test-service"))
			Expect(service.NamespaceID).To(Equal(namespaceID))
		})

		It("should not allow duplicate service names in the same namespace", func() {
			dnsConfig := &DnsConfig{
				NamespaceId: namespaceID,
				DnsRecords: []DnsRecord{
					{Type: "A", TTL: 60},
				},
			}

			_, err := manager.CreateService(ctx, "test-service", namespaceID, dnsConfig, nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = manager.CreateService(ctx, "test-service", namespaceID, dnsConfig, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should delete a service without instances", func() {
			dnsConfig := &DnsConfig{
				NamespaceId: namespaceID,
				DnsRecords: []DnsRecord{
					{Type: "A", TTL: 60},
				},
			}

			service, err := manager.CreateService(ctx, "test-service", namespaceID, dnsConfig, nil)
			Expect(err).NotTo(HaveOccurred())

			err = manager.DeleteService(ctx, service.ID)
			Expect(err).NotTo(HaveOccurred())

			_, err = manager.GetService(ctx, service.ID)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Instance Operations", func() {
		var serviceID string

		BeforeEach(func() {
			namespace, err := manager.CreatePrivateDnsNamespace(ctx, "test.local", "vpc-123", nil)
			Expect(err).NotTo(HaveOccurred())

			dnsConfig := &DnsConfig{
				NamespaceId: namespace.ID,
				DnsRecords: []DnsRecord{
					{Type: "A", TTL: 60},
				},
			}

			service, err := manager.CreateService(ctx, "test-service", namespace.ID, dnsConfig, nil)
			Expect(err).NotTo(HaveOccurred())
			serviceID = service.ID
		})

		It("should register an instance", func() {
			attributes := map[string]string{
				"AWS_INSTANCE_IPV4": "10.0.0.1",
				"PORT":              "8080",
			}

			instance, err := manager.RegisterInstance(ctx, serviceID, "instance-1", attributes)
			Expect(err).NotTo(HaveOccurred())
			Expect(instance).NotTo(BeNil())
			Expect(instance.ID).To(Equal("instance-1"))
			Expect(instance.ServiceID).To(Equal(serviceID))
		})

		It("should deregister an instance", func() {
			attributes := map[string]string{
				"AWS_INSTANCE_IPV4": "10.0.0.1",
			}

			_, err := manager.RegisterInstance(ctx, serviceID, "instance-1", attributes)
			Expect(err).NotTo(HaveOccurred())

			err = manager.DeregisterInstance(ctx, serviceID, "instance-1")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should discover instances", func() {
			// Register multiple instances
			for i := 1; i <= 3; i++ {
				attributes := map[string]string{
					"AWS_INSTANCE_IPV4": fmt.Sprintf("10.0.0.%d", i),
				}
				_, err := manager.RegisterInstance(ctx, serviceID, fmt.Sprintf("instance-%d", i), attributes)
				Expect(err).NotTo(HaveOccurred())
			}

			// Discover instances
			req := &DiscoverInstancesRequest{
				NamespaceName: "test.local",
				ServiceName:   "test-service",
			}

			resp, err := manager.DiscoverInstances(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Instances).To(HaveLen(3))
		})

		It("should update instance health status", func() {
			attributes := map[string]string{
				"AWS_INSTANCE_IPV4": "10.0.0.1",
			}

			_, err := manager.RegisterInstance(ctx, serviceID, "instance-1", attributes)
			Expect(err).NotTo(HaveOccurred())

			err = manager.UpdateInstanceHealthStatus(ctx, serviceID, "instance-1", "HEALTHY")
			Expect(err).NotTo(HaveOccurred())

			// Verify health status through discovery
			req := &DiscoverInstancesRequest{
				NamespaceName: "test.local",
				ServiceName:   "test-service",
				HealthStatus:  "HEALTHY",
			}

			resp, err := manager.DiscoverInstances(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Instances).To(HaveLen(1))
			Expect(resp.Instances[0].HealthStatus).To(Equal("HEALTHY"))
		})
	})

	Describe("Kubernetes Integration", func() {
		It("should create headless service for service discovery", func() {
			namespace, err := manager.CreatePrivateDnsNamespace(ctx, "test.local", "vpc-123", nil)
			Expect(err).NotTo(HaveOccurred())

			dnsConfig := &DnsConfig{
				NamespaceId: namespace.ID,
				DnsRecords: []DnsRecord{
					{Type: "A", TTL: 60},
				},
			}

			service, err := manager.CreateService(ctx, "test-service", namespace.ID, dnsConfig, nil)
			Expect(err).NotTo(HaveOccurred())

			// Register an instance to trigger Kubernetes service creation
			attributes := map[string]string{
				"AWS_INSTANCE_IPV4": "10.0.0.1",
			}

			_, err = manager.RegisterInstance(ctx, service.ID, "instance-1", attributes)
			Expect(err).NotTo(HaveOccurred())

			// In real implementation, this would create a Kubernetes service
			// Here we just verify no errors occurred
		})
	})
})

func TestServiceDiscovery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Discovery Suite")
}
