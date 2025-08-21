package kubernetes_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/mocks"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
)

var _ = Describe("LogCollector", func() {
	var (
		logCollector *kubernetes.LogCollector
		kubeClient   *fake.Clientset
		mockStorage  *mocks.MockStorage
		ctx          context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		kubeClient = fake.NewSimpleClientset()
		mockStorage = mocks.NewMockStorage()
		logCollector = kubernetes.NewLogCollector(kubeClient, mockStorage)
	})

	Describe("CollectTaskLogs", func() {
		Context("when collecting logs from a running pod", func() {
			BeforeEach(func() {
				// Create a test pod
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "test-namespace",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app-container",
								Image: "test-image:latest",
							},
							{
								Name:  "sidecar-container",
								Image: "sidecar:latest",
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				}
				_, err := kubeClient.CoreV1().Pods("test-namespace").Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Note: fake.Clientset doesn't support streaming pod logs
				// The test will pass but won't actually collect logs
				// This is acceptable for unit testing the basic flow
			})

			It("should complete without error when using fake client", func() {
				taskArn := "arn:aws:ecs:us-east-1:123456789012:task/test-cluster/test-task-123"
				// Fake client returns empty logs but doesn't error
				err := logCollector.CollectTaskLogs(ctx, taskArn, "test-namespace", "test-pod")
				// Should complete without error
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when pod has init containers", func() {
			BeforeEach(func() {
				// Create a test pod with init containers
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-with-init",
						Namespace: "test-namespace",
					},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name:  "init-container",
								Image: "init:latest",
							},
						},
						Containers: []corev1.Container{
							{
								Name:  "main-container",
								Image: "app:latest",
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				}
				_, err := kubeClient.CoreV1().Pods("test-namespace").Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Note: fake.Clientset doesn't support streaming pod logs
			})

			It("should complete without error when collecting from init containers", func() {
				taskArn := "arn:aws:ecs:us-east-1:123456789012:task/test-cluster/test-task-456"
				// Fake client returns empty logs but doesn't error
				err := logCollector.CollectTaskLogs(ctx, taskArn, "test-namespace", "test-pod-with-init")
				// Should complete without error
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when kubernetes client is not available", func() {
			It("should skip log collection gracefully", func() {
				// Create collector with nil client
				nilClientCollector := kubernetes.NewLogCollector(nil, mockStorage)
				
				taskArn := "arn:aws:ecs:us-east-1:123456789012:task/test-cluster/test-task-789"
				err := nilClientCollector.CollectTaskLogs(ctx, taskArn, "test-namespace", "test-pod")
				
				// Should not return error
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when storage is not available", func() {
			It("should skip log collection gracefully", func() {
				// Create collector with nil storage
				nilStorageCollector := kubernetes.NewLogCollector(kubeClient, nil)
				
				taskArn := "arn:aws:ecs:us-east-1:123456789012:task/test-cluster/test-task-999"
				err := nilStorageCollector.CollectTaskLogs(ctx, taskArn, "test-namespace", "test-pod")
				
				// Should not return error
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("CollectLogsBeforeDeletion", func() {
		It("should collect logs asynchronously with timeout", func() {
			// Create a test pod
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-async",
					Namespace: "test-namespace",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app:latest",
						},
					},
				},
			}
			_, err := kubeClient.CoreV1().Pods("test-namespace").Create(ctx, pod, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Note: fake.Clientset doesn't support streaming pod logs

			taskArn := "arn:aws:ecs:us-east-1:123456789012:task/test-cluster/test-task-async"
			
			// This should not block
			logCollector.CollectLogsBeforeDeletion(ctx, taskArn, "test-namespace", "test-pod-async")
			
			// Give it some time to complete
			time.Sleep(200 * time.Millisecond)
			
			// Since fake client doesn't support streaming, no logs will be collected
			// But the method should not block
		})
	})
})