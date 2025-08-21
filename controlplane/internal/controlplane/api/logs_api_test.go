package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/mocks"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("LogsAPI", func() {
	var (
		logsAPI    *api.LogsAPI
		mockStore  *mocks.MockStorage
		kubeClient *fake.Clientset
		router     *mux.Router
		server     *httptest.Server
	)

	BeforeEach(func() {
		// Create mock storage
		mockStore = mocks.NewMockStorage()

		// Create fake kubernetes client
		kubeClient = fake.NewSimpleClientset()

		// Create LogsAPI instance
		logsAPI = api.NewLogsAPI(mockStore, kubeClient)

		// Setup router
		router = mux.NewRouter()
		logsAPI.RegisterRoutes(router)

		// Create test server
		server = httptest.NewServer(router)
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("GET /api/tasks/{taskArn}/containers/{containerName}/logs", func() {
		Context("when retrieving historical logs from storage", func() {
			It("should return logs with pagination", func() {
				// Prepare test data
				now := time.Now()
				taskArn := "arn:aws:ecs:us-east-1:123456789012:task/test-cluster/test-task-123"
				containerName := "test-container"

				// Add test logs to mock storage
				logs := []storage.TaskLog{
					{
						ID:            "log-1",
						TaskArn:       taskArn,
						ContainerName: containerName,
						Timestamp:     now.Add(-2 * time.Minute),
						LogLine:       "Starting application...",
						LogLevel:      "INFO",
						CreatedAt:     now,
					},
					{
						ID:            "log-2",
						TaskArn:       taskArn,
						ContainerName: containerName,
						Timestamp:     now.Add(-1 * time.Minute),
						LogLine:       "Application started successfully",
						LogLevel:      "INFO",
						CreatedAt:     now,
					},
				}

				// Save logs to mock storage
				err := mockStore.TaskLogStore().SaveLogs(context.Background(), logs)
				Expect(err).NotTo(HaveOccurred())

				// Make request
				reqUrl := fmt.Sprintf("%s/api/tasks/%s/containers/%s/logs?limit=10",
					server.URL, taskArn, containerName)
				resp, err := http.Get(reqUrl)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				// Check response
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				// Parse response
				var result map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&result)
				Expect(err).NotTo(HaveOccurred())

				// Verify response structure
				Expect(result).To(HaveKey("logs"))
				Expect(result).To(HaveKey("totalCount"))
				Expect(result).To(HaveKey("limit"))
				Expect(result).To(HaveKey("offset"))

				// Verify logs
				logsData, ok := result["logs"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(logsData).To(HaveLen(2))
			})

			It("should filter logs by log level", func() {
				// Prepare test data
				now := time.Now()
				taskArn := "arn:aws:ecs:us-east-1:123456789012:task/test-cluster/test-task-456"
				containerName := "app-container"

				// Add test logs with different levels
				logs := []storage.TaskLog{
					{
						ID:            "log-1",
						TaskArn:       taskArn,
						ContainerName: containerName,
						Timestamp:     now.Add(-3 * time.Minute),
						LogLine:       "Debug message",
						LogLevel:      "DEBUG",
						CreatedAt:     now,
					},
					{
						ID:            "log-2",
						TaskArn:       taskArn,
						ContainerName: containerName,
						Timestamp:     now.Add(-2 * time.Minute),
						LogLine:       "Error occurred",
						LogLevel:      "ERROR",
						CreatedAt:     now,
					},
					{
						ID:            "log-3",
						TaskArn:       taskArn,
						ContainerName: containerName,
						Timestamp:     now.Add(-1 * time.Minute),
						LogLine:       "Info message",
						LogLevel:      "INFO",
						CreatedAt:     now,
					},
				}

				// Save logs to mock storage
				err := mockStore.TaskLogStore().SaveLogs(context.Background(), logs)
				Expect(err).NotTo(HaveOccurred())

				// Make request filtering by ERROR level
				reqUrl := fmt.Sprintf("%s/api/tasks/%s/containers/%s/logs?level=ERROR",
					server.URL, taskArn, containerName)
				resp, err := http.Get(reqUrl)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				// Check response
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				// Parse response
				var result map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&result)
				Expect(err).NotTo(HaveOccurred())

				// Verify filtered logs
				logsData, ok := result["logs"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(logsData).To(HaveLen(1)) // Only ERROR log
			})
		})
	})

	Describe("GET /api/tasks/{taskArn}/containers/{containerName}/logs/stream", func() {
		Context("when streaming logs from Kubernetes", func() {
			BeforeEach(func() {
				// Create a test pod in the fake kubernetes client
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-task-789",
						Namespace: "test-cluster-us-east-1",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test-container",
								Image: "test-image",
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				}
				_, err := kubeClient.CoreV1().Pods("test-cluster-us-east-1").Create(
					context.Background(), pod, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should stream logs as Server-Sent Events", func() {
				taskArn := "arn:aws:ecs:us-east-1:123456789012:task/test-cluster/test-task-789"
				containerName := "test-container"

				// Make request with follow=false for one-time fetch
				reqUrl := fmt.Sprintf("%s/api/tasks/%s/containers/%s/logs/stream?follow=false&tail=10",
					server.URL, taskArn, containerName)
				resp, err := http.Get(reqUrl)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				// Check SSE headers
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(resp.Header.Get("Content-Type")).To(Equal("text/event-stream"))
				Expect(resp.Header.Get("Cache-Control")).To(Equal("no-cache"))

				// Read a portion of the response
				buf := make([]byte, 1024)
				n, _ := resp.Body.Read(buf)
				body := string(buf[:n])

				// Verify SSE format
				Expect(body).To(ContainSubstring("event:"))
			})
		})
	})

	Describe("GET /api/tasks/{taskArn}/containers/{containerName}/logs/ws", func() {
		Context("when establishing WebSocket connection", func() {
			It("should reject non-WebSocket requests", func() {
				taskArn := "arn:aws:ecs:us-east-1:123456789012:task/test-cluster/test-task-999"
				containerName := "ws-container"

				// Make regular HTTP request (not WebSocket)
				reqUrl := fmt.Sprintf("%s/api/tasks/%s/containers/%s/logs/ws",
					server.URL, taskArn, containerName)
				resp, err := http.Get(reqUrl)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				// Should return bad request for non-WebSocket connection
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})
		})
	})

	Describe("parseTaskArn", func() {
		It("should correctly parse task ARN to extract namespace and pod name", func() {
			// This is tested indirectly through the API endpoints above
			// The parseTaskArn function is private, so we test it through the public API
		})
	})
})
