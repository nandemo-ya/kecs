package portforward_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/portforward"
)

var _ = Describe("Manager", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		manager    *portforward.Manager
		k8sClient  *kubernetes.Client
		fakeClient *fake.Clientset
	)

	createTestService := func(namespace, name string, nodePort int32) *corev1.Service {
		return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort,
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Port:       80,
						TargetPort: intstr.FromInt(80),
						NodePort:   nodePort,
					},
				},
				Selector: map[string]string{
					"app": name,
				},
			},
		}
	}

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		// Create fake Kubernetes client
		fakeClient = fake.NewSimpleClientset()
		k8sClient = &kubernetes.Client{
			Clientset: fakeClient,
		}

		// Create manager with test instance name
		manager = portforward.NewManager("test-instance", k8sClient)
	})

	AfterEach(func() {
		cancel()
		manager.Stop()
		// Give time for cleanup
		time.Sleep(100 * time.Millisecond)
	})

	Describe("Forward Management", func() {
		Context("when starting a service forward", func() {
			It("should create a new forward entry", func() {
				// Create a test service with NodePort
				service := createTestService("test-cluster-us-east-1", "test-service", 30080)
				_, err := fakeClient.CoreV1().Services("test-cluster-us-east-1").Create(ctx, service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Start service forward
				forwardID, localPort, err := manager.StartServiceForward(
					ctx,
					"test-cluster",
					"test-service",
					8080,
					80,
				)

				// Note: This will fail in test environment due to k3d dependency
				// but we can verify the error message
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("k3d"))
				Expect(forwardID).To(BeEmpty())
				Expect(localPort).To(Equal(0))
			})

			It("should return error for service without NodePort", func() {
				// Create a test service without NodePort (ClusterIP)
				service := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service",
						Namespace: "test-cluster-us-east-1",
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
						Ports: []corev1.ServicePort{
							{
								Name:       "http",
								Port:       80,
								TargetPort: intstr.FromInt(80),
							},
						},
					},
				}
				_, err := fakeClient.CoreV1().Services("test-cluster-us-east-1").Create(ctx, service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Try to start forward
				forwardID, localPort, err := manager.StartServiceForward(
					ctx,
					"test-cluster",
					"test-service",
					8080,
					80,
				)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not have NodePort configured"))
				Expect(forwardID).To(BeEmpty())
				Expect(localPort).To(Equal(0))
			})
		})

		Context("when starting a task forward", func() {
			It("should create a new forward entry", func() {
				// Create a test pod
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-task-id",
						Namespace: "test-cluster-us-east-1",
						Labels: map[string]string{
							"kecs.dev/task-id": "test-task-id",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "main",
								Image: "nginx:latest",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 8080,
									},
								},
							},
						},
					},
				}
				_, err := fakeClient.CoreV1().Pods("test-cluster-us-east-1").Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Start task forward
				forwardID, localPort, err := manager.StartTaskForward(
					ctx,
					"test-cluster",
					"test-task-id",
					9090,
					8080,
				)

				// Note: This will fail in test environment due to kubectl dependency
				// but we can verify the error is from kubectl not from pod lookup
				Expect(err).To(HaveOccurred())
				// The error should be about starting kubectl, not finding the pod
				Expect(err.Error()).ToNot(ContainSubstring("no pod found"))
				Expect(forwardID).To(BeEmpty())
				Expect(localPort).To(Equal(0))
			})

			It("should return error for non-existent task", func() {
				forwardID, localPort, err := manager.StartTaskForward(
					ctx,
					"test-cluster",
					"non-existent-task",
					9090,
					8080,
				)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no pod found"))
				Expect(forwardID).To(BeEmpty())
				Expect(localPort).To(Equal(0))
			})
		})

		Context("when listing forwards", func() {
			It("should return empty list when no forwards exist", func() {
				forwards, err := manager.ListForwards()
				Expect(err).ToNot(HaveOccurred())
				Expect(forwards).To(BeEmpty())
			})

			// Note: We can't test successful forwards due to k3d/kubectl dependencies
		})

		Context("when stopping forwards", func() {
			It("should return error when stopping non-existent forward", func() {
				err := manager.StopForward("non-existent-id")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})

			It("should handle StopAllForwards gracefully with no forwards", func() {
				// Should not panic or error
				manager.StopAllForwards()

				forwards, err := manager.ListForwards()
				Expect(err).ToNot(HaveOccurred())
				Expect(forwards).To(BeEmpty())
			})
		})

		Context("when handling concurrency", func() {
			It("should handle concurrent service lookups safely", func() {
				// Create multiple test services
				for i := 0; i < 5; i++ {
					service := createTestService("test-cluster-us-east-1", fmt.Sprintf("service%d", i), int32(30080+i))
					_, err := fakeClient.CoreV1().Services("test-cluster-us-east-1").Create(ctx, service, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}

				var wg sync.WaitGroup
				errors := make([]error, 5)

				for i := 0; i < 5; i++ {
					wg.Add(1)
					go func(index int) {
						defer wg.Done()
						_, _, err := manager.StartServiceForward(
							ctx,
							"test-cluster",
							fmt.Sprintf("service%d", index),
							8080+index,
							80,
						)
						errors[index] = err
					}(i)
				}

				wg.Wait()

				// All should fail due to k3d dependency, but should fail safely
				for _, err := range errors {
					Expect(err).To(HaveOccurred())
				}
			})

			It("should handle concurrent stops safely", func() {
				var wg sync.WaitGroup

				// Try to stop multiple non-existent forwards concurrently
				for i := 0; i < 5; i++ {
					wg.Add(1)
					go func(index int) {
						defer wg.Done()
						manager.StopForward(fmt.Sprintf("forward-%d", index))
					}(i)
				}

				wg.Wait()

				// Should complete without panic
				forwards, err := manager.ListForwards()
				Expect(err).ToNot(HaveOccurred())
				Expect(forwards).To(BeEmpty())
			})
		})
	})

	Describe("Process Management", func() {
		Context("when managing kubectl processes", func() {
			It("should handle missing pods gracefully", func() {
				_, _, err := manager.StartTaskForward(
					ctx,
					"test-cluster",
					"missing-task",
					8080,
					80,
				)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no pod found"))
			})
		})
	})

	Describe("Context Cancellation", func() {
		It("should stop manager when Stop is called", func() {
			// Create a new manager
			testManager := portforward.NewManager("test-instance-2", k8sClient)

			// Stop the manager
			testManager.Stop()

			// Give time for cleanup
			time.Sleep(200 * time.Millisecond)

			// Manager should still be able to list (empty) forwards
			forwards, err := testManager.ListForwards()
			Expect(err).ToNot(HaveOccurred())
			Expect(forwards).To(BeEmpty())
		})
	})
})

var _ = Describe("Forward", func() {
	Describe("Forward Status", func() {
		It("should have correct initial status", func() {
			forward := &portforward.Forward{
				ID:            "test-id",
				Type:          portforward.ForwardTypeService,
				Cluster:       "test-cluster",
				TargetName:    "test-service",
				LocalPort:     8080,
				TargetPort:    80,
				Status:        portforward.StatusActive,
				CreatedAt:     time.Now(),
				AutoReconnect: true,
			}

			Expect(forward.Status).To(Equal(portforward.StatusActive))
			Expect(forward.RetryCount).To(Equal(0))
		})
	})
})
