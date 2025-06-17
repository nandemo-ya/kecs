package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/servicediscovery"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/inmemory"
)

var _ = Describe("Service Discovery Integration", func() {
	var (
		server              *httptest.Server
		serviceDiscoveryMgr servicediscovery.Manager
		ecsAPI             *api.DefaultECSAPI
		storage            *inmemory.Storage
	)

	BeforeEach(func() {
		// Create storage
		storage = inmemory.NewStorage()

		// Create fake Kubernetes client
		fakeClient := fake.NewSimpleClientset()

		// Create service discovery manager
		serviceDiscoveryMgr = servicediscovery.NewManager(fakeClient, "us-east-1", "123456789012")

		// Create ECS API with service discovery
		kindManager := &kubernetes.KindManager{}
		ecsAPI = api.NewDefaultECSAPIWithConfig(storage, kindManager, "us-east-1", "123456789012").(*api.DefaultECSAPI)
		ecsAPI.SetServiceDiscoveryManager(serviceDiscoveryMgr)

		// Create test server
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := r.Header.Get("X-Amz-Target")
			if target != "" && contains(target, "ServiceDiscovery") {
				serviceDiscoveryAPI := api.NewServiceDiscoveryAPI(serviceDiscoveryMgr, "us-east-1", "123456789012")
				serviceDiscoveryAPI.HandleServiceDiscoveryRequest(w, r)
			} else {
				generated.HandleECSRequest(ecsAPI)(w, r)
			}
		})
		server = httptest.NewServer(handler)
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Service Creation with Service Discovery", func() {
		var (
			namespaceID string
			serviceID   string
		)

		BeforeEach(func() {
			// Create a namespace first
			req := api.CreatePrivateDnsNamespaceRequest{
				Name: "test.local",
				Vpc:  "vpc-123456",
			}

			body, _ := json.Marshal(req)
			resp := makeRequest(server.URL, "ServiceDiscovery.CreatePrivateDnsNamespace", body)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// For testing, we'll use a fixed namespace ID
			namespaceID = "ns-test123"

			// Create a service discovery service
			sdReq := api.CreateServiceDiscoveryServiceRequest{
				Name:        "my-service",
				NamespaceId: namespaceID,
				DnsConfig: &servicediscovery.DnsConfig{
					NamespaceId: namespaceID,
					DnsRecords: []servicediscovery.DnsRecord{
						{Type: "A", TTL: 60},
					},
				},
			}

			body, _ = json.Marshal(sdReq)
			resp = makeRequest(server.URL, "ServiceDiscovery.CreateService", body)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var sdResp api.CreateServiceDiscoveryServiceResponse
			json.NewDecoder(resp.Body).Decode(&sdResp)
			serviceID = sdResp.Service.ID
		})

		It("should create an ECS service with service registry", func() {
			// Create cluster first
			clusterReq := generated.CreateClusterRequest{
				ClusterName: strPtr("test-cluster"),
			}
			body, _ := json.Marshal(clusterReq)
			resp := makeRequest(server.URL, "AmazonEC2ContainerServiceV20141113.CreateCluster", body)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Register task definition
			taskDefReq := generated.RegisterTaskDefinitionRequest{
				Family: strPtr("test-task"),
				ContainerDefinitions: []generated.ContainerDefinition{
					{
						Name:  strPtr("web"),
						Image: strPtr("nginx:latest"),
						PortMappings: []generated.PortMapping{
							{
								ContainerPort: intPtr(80),
							},
						},
					},
				},
			}
			body, _ = json.Marshal(taskDefReq)
			resp = makeRequest(server.URL, "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition", body)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Create service with service registry
			serviceArn := "arn:aws:servicediscovery:us-east-1:123456789012:service/" + serviceID
			createServiceReq := generated.CreateServiceRequest{
				Cluster:        strPtr("test-cluster"),
				ServiceName:    strPtr("test-service"),
				TaskDefinition: strPtr("test-task"),
				DesiredCount:   intPtr(2),
				ServiceRegistries: []generated.ServiceRegistry{
					{
						RegistryArn:   &serviceArn,
						ContainerName: strPtr("web"),
						ContainerPort: intPtr(80),
					},
				},
			}

			body, _ = json.Marshal(createServiceReq)
			resp = makeRequest(server.URL, "AmazonEC2ContainerServiceV20141113.CreateService", body)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var createServiceResp generated.CreateServiceResponse
			err := json.NewDecoder(resp.Body).Decode(&createServiceResp)
			Expect(err).NotTo(HaveOccurred())
			Expect(createServiceResp.Service).NotTo(BeNil())
			Expect(createServiceResp.Service.ServiceName).To(Equal(strPtr("test-service")))
			Expect(createServiceResp.Service.ServiceRegistries).To(HaveLen(1))
		})

		It("should discover instances after task registration", func() {
			// Register an instance
			regReq := api.RegisterInstanceRequest{
				ServiceId:  serviceID,
				InstanceId: "task-123",
				Attributes: map[string]string{
					"AWS_INSTANCE_IPV4": "10.0.0.1",
					"PORT":              "80",
					"ECS_CLUSTER":       "test-cluster",
					"ECS_SERVICE":       "test-service",
					"ECS_TASK":          "task-123",
				},
			}

			body, _ := json.Marshal(regReq)
			resp := makeRequest(server.URL, "ServiceDiscovery.RegisterInstance", body)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Discover instances
			discoverReq := servicediscovery.DiscoverInstancesRequest{
				NamespaceName: "test.local",
				ServiceName:   "my-service",
			}

			body, _ = json.Marshal(discoverReq)
			resp = makeRequest(server.URL, "ServiceDiscovery.DiscoverInstances", body)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var discoverResp servicediscovery.DiscoverInstancesResponse
			err := json.NewDecoder(resp.Body).Decode(&discoverResp)
			Expect(err).NotTo(HaveOccurred())
			Expect(discoverResp.Instances).To(HaveLen(1))
			Expect(discoverResp.Instances[0].InstanceId).To(Equal("task-123"))
			Expect(discoverResp.Instances[0].Attributes["AWS_INSTANCE_IPV4"]).To(Equal("10.0.0.1"))
		})
	})
})

func makeRequest(baseURL, target string, body []byte) *http.Response {
	req, _ := http.NewRequest("POST", baseURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", target)
	
	client := &http.Client{}
	resp, _ := client.Do(req)
	return resp
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int32) *int32 {
	return &i
}

func TestServiceDiscoveryIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Discovery Integration Suite")
}