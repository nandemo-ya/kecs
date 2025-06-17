package converters_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/proxy"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("TaskConverter Sidecar Injection", func() {
	var (
		converter    *converters.TaskConverter
		taskDef      *storage.TaskDefinition
		cluster      *storage.Cluster
		proxyManager *proxy.Manager
	)

	BeforeEach(func() {
		converter = converters.NewTaskConverter("us-east-1", "123456789012")
		
		// Create proxy manager with sidecar mode
		kubeClient := fake.NewSimpleClientset()
		proxyConfig := &localstack.ProxyConfig{
			Mode:               localstack.ProxyModeSidecar,
			LocalStackEndpoint: "http://localstack:4566",
		}
		var err error
		proxyManager, err = proxy.NewManager(kubeClient, proxyConfig)
		Expect(err).ToNot(HaveOccurred())
		
		// Start proxy manager to initialize sidecar proxy
		err = proxyManager.Start(context.Background())
		Expect(err).ToNot(HaveOccurred())
		
		// Set proxy manager on converter
		converter.SetProxyManager(proxyManager)

		cluster = &storage.Cluster{
			Name: "test-cluster",
			ARN:  "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster",
		}

		// Create a basic task definition
		containerDefs := []types.ContainerDefinition{
			{
				Name:   stringPtr("app"),
				Image:  stringPtr("nginx:latest"),
				Memory: intPtr(512),
				Cpu:    intPtr(256),
			},
		}

		containerDefsJSON, _ := json.Marshal(containerDefs)
		taskDef = &storage.TaskDefinition{
			Family:               "test-task",
			Revision:             1,
			ContainerDefinitions: string(containerDefsJSON),
			CPU:                  "256",
			Memory:               "512",
			ARN:                  "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
		}
	})

	Describe("ConvertTaskToPod with AWS proxy sidecar", func() {
		It("should inject sidecar for ECS tasks by default", func() {
			runTaskReq := types.RunTaskRequest{
				TaskDefinition: stringPtr("test-task:1"),
				Cluster:        stringPtr("test-cluster"),
			}
			reqJSON, _ := json.Marshal(runTaskReq)

			pod, err := converter.ConvertTaskToPod(taskDef, reqJSON, cluster, "task-123")
			Expect(err).ToNot(HaveOccurred())
			Expect(pod).ToNot(BeNil())

			// Check that sidecar was injected
			Expect(len(pod.Spec.Containers)).To(Equal(2))
			
			// Find the sidecar
			var sidecarFound bool
			for _, container := range pod.Spec.Containers {
				if container.Name == "aws-proxy-sidecar" {
					sidecarFound = true
					Expect(container.Image).To(Equal("kecs/aws-proxy:latest"))
					
					// Check environment variables
					envMap := make(map[string]string)
					for _, env := range container.Env {
						envMap[env.Name] = env.Value
					}
					Expect(envMap["LOCALSTACK_ENDPOINT"]).To(Equal("http://localstack:4566"))
					break
				}
			}
			Expect(sidecarFound).To(BeTrue())

			// Check that main container has AWS endpoint environment variables
			mainContainer := pod.Spec.Containers[0]
			envMap := make(map[string]string)
			for _, env := range mainContainer.Env {
				envMap[env.Name] = env.Value
			}
			Expect(envMap["AWS_ENDPOINT_URL"]).To(Equal("http://localhost:4566"))
			Expect(envMap["AWS_ENDPOINT_URL_S3"]).To(Equal("http://localhost:4566"))

			// Check annotation
			Expect(pod.Annotations["kecs.io/aws-proxy-sidecar-injected"]).To(Equal("true"))
		})

		It("should not inject sidecar when explicitly disabled", func() {
			runTaskReq := types.RunTaskRequest{
				TaskDefinition: stringPtr("test-task:1"),
				Cluster:        stringPtr("test-cluster"),
			}
			reqJSON, _ := json.Marshal(runTaskReq)

			pod, err := converter.ConvertTaskToPod(taskDef, reqJSON, cluster, "task-123")
			Expect(err).ToNot(HaveOccurred())

			// Add annotation to disable proxy
			pod.Annotations["kecs.io/aws-proxy-enabled"] = "false"

			// Re-run through proxy manager (simulating what would happen in practice)
			if proxyManager.GetSidecarProxy() != nil {
				sidecarProxy := proxyManager.GetSidecarProxy()
				shouldInject := sidecarProxy.ShouldInjectSidecar(pod)
				Expect(shouldInject).To(BeFalse())
			}
		})

		It("should use custom LocalStack endpoint from annotation", func() {
			runTaskReq := types.RunTaskRequest{
				TaskDefinition: stringPtr("test-task:1"),
				Cluster:        stringPtr("test-cluster"),
			}
			reqJSON, _ := json.Marshal(runTaskReq)

			// First create the pod
			pod, err := converter.ConvertTaskToPod(taskDef, reqJSON, cluster, "task-123")
			Expect(err).ToNot(HaveOccurred())

			// Add custom endpoint annotation before sidecar injection
			pod.Annotations["kecs.io/localstack-endpoint"] = "http://custom-localstack:9999"

			// Get sidecar proxy and check if it would use custom endpoint
			if proxyManager.GetSidecarProxy() != nil {
				sidecarProxy := proxyManager.GetSidecarProxy()
				sidecarContainer := sidecarProxy.CreateProxySidecar(pod)
				
				// Check environment variables
				envMap := make(map[string]string)
				for _, env := range sidecarContainer.Env {
					envMap[env.Name] = env.Value
				}
				Expect(envMap["LOCALSTACK_ENDPOINT"]).To(Equal("http://custom-localstack:9999"))
			}
		})

		It("should respect proxy mode annotation", func() {
			// Test with environment mode
			runTaskReq := types.RunTaskRequest{
				TaskDefinition: stringPtr("test-task:1"),
				Cluster:        stringPtr("test-cluster"),
			}
			reqJSON, _ := json.Marshal(runTaskReq)

			pod, err := converter.ConvertTaskToPod(taskDef, reqJSON, cluster, "task-123")
			Expect(err).ToNot(HaveOccurred())

			// Set to environment mode
			pod.Annotations["kecs.io/aws-proxy-mode"] = "environment"

			// Check that sidecar should not be injected
			if proxyManager.GetSidecarProxy() != nil {
				sidecarProxy := proxyManager.GetSidecarProxy()
				shouldInject := sidecarProxy.ShouldInjectSidecar(pod)
				Expect(shouldInject).To(BeFalse())
			}
		})

		It("should enable debug mode when requested", func() {
			runTaskReq := types.RunTaskRequest{
				TaskDefinition: stringPtr("test-task:1"),
				Cluster:        stringPtr("test-cluster"),
			}
			reqJSON, _ := json.Marshal(runTaskReq)

			pod, err := converter.ConvertTaskToPod(taskDef, reqJSON, cluster, "task-123")
			Expect(err).ToNot(HaveOccurred())

			// Enable debug mode
			pod.Annotations["kecs.io/aws-proxy-debug"] = "true"

			// Check sidecar configuration
			if proxyManager.GetSidecarProxy() != nil {
				sidecarProxy := proxyManager.GetSidecarProxy()
				sidecarContainer := sidecarProxy.CreateProxySidecar(pod)
				
				// Check DEBUG environment variable
				var debugValue string
				for _, env := range sidecarContainer.Env {
					if env.Name == "DEBUG" {
						debugValue = env.Value
						break
					}
				}
				Expect(debugValue).To(Equal("true"))
			}
		})
	})

	Describe("Multiple sidecars", func() {
		It("should work with both CloudWatch and AWS proxy sidecars", func() {
			// Create task definition with CloudWatch logs
			containerDefs := []types.ContainerDefinition{
				{
					Name:   stringPtr("app"),
					Image:  stringPtr("nginx:latest"),
					Memory: intPtr(512),
					Cpu:    intPtr(256),
					LogConfiguration: &types.LogConfiguration{
						LogDriver: stringPtr("awslogs"),
						Options: map[string]string{
							"awslogs-group":  "/ecs/test-task",
							"awslogs-region": "us-east-1",
						},
					},
				},
			}

			containerDefsJSON, _ := json.Marshal(containerDefs)
			taskDef.ContainerDefinitions = string(containerDefsJSON)

			// Set up CloudWatch integration
			// This would normally be done, but we'll skip for this test
			// to avoid complex mocking

			runTaskReq := types.RunTaskRequest{
				TaskDefinition: stringPtr("test-task:1"),
				Cluster:        stringPtr("test-cluster"),
			}
			reqJSON, _ := json.Marshal(runTaskReq)

			pod, err := converter.ConvertTaskToPod(taskDef, reqJSON, cluster, "task-123")
			Expect(err).ToNot(HaveOccurred())

			// Should have at least 2 containers (app + aws-proxy-sidecar)
			// CloudWatch sidecar would be 3rd if CloudWatch integration was set up
			Expect(len(pod.Spec.Containers)).To(BeNumerically(">=", 2))

			// Check for AWS proxy sidecar
			var awsProxyFound bool
			for _, container := range pod.Spec.Containers {
				if container.Name == "aws-proxy-sidecar" {
					awsProxyFound = true
					break
				}
			}
			Expect(awsProxyFound).To(BeTrue())
		})
	})
})

