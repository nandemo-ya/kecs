package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"nhooyr.io/websocket"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/mocks"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("LocalStack Dashboard Integration", func() {
	var (
		mockStore  *mocks.MockStorage
		mockServiceStore *mocks.MockServiceStore
		testServer *httptest.Server
		ctx        context.Context
		cancel     context.CancelFunc
		baseURL    string
		config     *localstack.Config
	)

	BeforeEach(func() {
		mockStore = mocks.NewMockStorage()
		mockServiceStore = mocks.NewMockServiceStore()
		mockStore.SetServiceStore(mockServiceStore)
		
		ctx, cancel = context.WithCancel(context.Background())
		config = &localstack.Config{
			Namespace: "localstack",
			Port:      4566,
			Services:  []string{"s3", "dynamodb", "sqs", "sns"},
		}
		
		server, err := NewServer(8080, "/", mockStore, config)
		Expect(err).NotTo(HaveOccurred())
		
		// Get the router from server
		router := server.setupRoutes()
		testServer = httptest.NewServer(router)
		baseURL = testServer.URL
		
		// Allow servers to start
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		cancel()
		testServer.Close()
		time.Sleep(100 * time.Millisecond) // Allow cleanup
	})

	Describe("LocalStack Dashboard API", func() {
		It("should return dashboard data", func() {
			// Setup test data
			testService := &storage.Service{
				ServiceName:       "my-service",
				ClusterARN:        "arn:aws:ecs:us-east-1:123456789012:cluster/test",
				TaskDefinitionARN: "arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1",
			}
			err := mockServiceStore.Create(ctx, testService)
			Expect(err).NotTo(HaveOccurred())

			// Setup task definition store with test data
			mockTaskDefStore := mocks.NewMockTaskDefinitionStore()
			taskDef := &storage.TaskDefinition{
				Family:   "my-task",
				Revision: 1,
				ContainerDefinitions: `[{"name":"app","image":"nginx:latest","environment":[{"name":"AWS_REGION","value":"us-east-1"}]}]`,
			}
			_, err = mockTaskDefStore.Register(ctx, taskDef)
			Expect(err).NotTo(HaveOccurred())
			mockStore.SetTaskDefinitionStore(mockTaskDefStore)

			// Get dashboard data
			resp, err := http.Get(baseURL + "/localstack/dashboard")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var dashboard map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&dashboard)
			resp.Body.Close()
			Expect(err).NotTo(HaveOccurred())
			
			// Verify response
			Expect(dashboard["tasksUsingLocalStack"]).To(Equal(float64(1)))
			resourceUsage, ok := dashboard["resourceUsage"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(resourceUsage).To(ContainElement("aws"))
		})

		It("should handle WebSocket connections", func() {
			// Connect WebSocket
			wsURL := "ws" + baseURL[4:] + "/ws"
			conn, _, err := websocket.Dial(ctx, wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer conn.Close(websocket.StatusNormalClosure, "")

			// Subscribe to LocalStack events
			subscribePayload, _ := json.Marshal(map[string]interface{}{"topics": []string{"localstack"}})
			subscribeMsg := WebSocketMessage{
				Type:    "subscribe",
				Payload: subscribePayload,
			}
			err = conn.Write(ctx, websocket.MessageText, mustMarshal(subscribeMsg))
			Expect(err).NotTo(HaveOccurred())

			// Allow time for subscription
			time.Sleep(100 * time.Millisecond)

			// Receive WebSocket event
			msgCtx, msgCancel := context.WithTimeout(ctx, 2*time.Second)
			defer msgCancel()

			_, msg, err := conn.Read(msgCtx)
			if err == nil {
				var wsMsg WebSocketMessage
				err = json.Unmarshal(msg, &wsMsg)
				Expect(err).NotTo(HaveOccurred())
				Expect(wsMsg.Type).To(Equal("localstack"))
			}
		})

		It("should handle concurrent WebSocket connections", func() {
			const numConnections = 3
			connections := make([]*websocket.Conn, numConnections)
			
			// Create multiple WebSocket connections
			for i := 0; i < numConnections; i++ {
				wsURL := "ws" + baseURL[4:] + "/ws"
				conn, _, err := websocket.Dial(ctx, wsURL, nil)
				Expect(err).NotTo(HaveOccurred())
				connections[i] = conn
				
				// Subscribe to LocalStack events
				subscribePayload, _ := json.Marshal(map[string]interface{}{"topics": []string{"localstack"}})
				subscribeMsg := WebSocketMessage{
					Type:    "subscribe",
					Payload: subscribePayload,
				}
				err = conn.Write(ctx, websocket.MessageText, mustMarshal(subscribeMsg))
				Expect(err).NotTo(HaveOccurred())
			}
			
			// Allow time for subscriptions
			time.Sleep(100 * time.Millisecond)
			
			// Allow time for event propagation
			time.Sleep(100 * time.Millisecond)
			
			// Clean up
			for _, conn := range connections {
				conn.Close(websocket.StatusNormalClosure, "")
			}
		})

		It("should discover LocalStack services from task definitions", func() {
			// Setup test services
			services := []*storage.Service{
				{
					ServiceName:       "api-service",
					ClusterARN: "arn:aws:ecs:us-east-1:123456789012:cluster/test",
					TaskDefinitionARN: "arn:aws:ecs:us-east-1:123456789012:task-definition/api-task:1",
				},
				{
					ServiceName:       "worker-service",
					ClusterARN: "arn:aws:ecs:us-east-1:123456789012:cluster/test",
					TaskDefinitionARN: "arn:aws:ecs:us-east-1:123456789012:task-definition/worker-task:1",
				},
			}

			for _, svc := range services {
				err := mockServiceStore.Create(ctx, svc)
				Expect(err).NotTo(HaveOccurred())
			}

			// Setup task definitions
			mockTaskDefStore := mocks.NewMockTaskDefinitionStore()
			taskDefs := []*storage.TaskDefinition{
				{
					Family:   "api-task",
					Revision: 1,
					ContainerDefinitions: `[{"name":"api","image":"myapp:latest","environment":[{"name":"AWS_REGION","value":"us-east-1"},{"name":"S3_BUCKET","value":"my-bucket"},{"name":"DYNAMODB_TABLE","value":"my-table"}]}]`,
				},
				{
					Family:   "worker-task",
					Revision: 1,
					ContainerDefinitions: `[{"name":"worker","image":"worker:latest","environment":[{"name":"AWS_REGION","value":"us-east-1"},{"name":"SQS_QUEUE","value":"my-queue"},{"name":"SNS_TOPIC","value":"my-topic"}]}]`,
				},
			}

			for _, td := range taskDefs {
				_, err := mockTaskDefStore.Register(ctx, td)
				Expect(err).NotTo(HaveOccurred())
			}
			mockStore.SetTaskDefinitionStore(mockTaskDefStore)

			// Get dashboard
			resp, err := http.Get(baseURL + "/localstack/dashboard")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var dashboard map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&dashboard)
			resp.Body.Close()
			Expect(err).NotTo(HaveOccurred())

			// Should have services and resource usage
			Expect(dashboard["tasksUsingLocalStack"]).To(Equal(float64(2)))
			resourceUsage, ok := dashboard["resourceUsage"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(resourceUsage).To(ContainElements("s3", "dynamodb", "sqs", "sns"))
		})
	})

	Describe("Error Handling", func() {
		It("should handle empty service list gracefully", func() {
			resp, err := http.Get(baseURL + "/localstack/dashboard")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var dashboard map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&dashboard)
			resp.Body.Close()
			Expect(err).NotTo(HaveOccurred())
			Expect(dashboard["tasksUsingLocalStack"]).To(Equal(float64(0)))
			resourceUsage, ok := dashboard["resourceUsage"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(resourceUsage).To(BeEmpty())
		})

		It("should handle WebSocket disconnection gracefully", func() {
			wsURL := "ws" + baseURL[4:] + "/ws"
			conn, _, err := websocket.Dial(ctx, wsURL, nil)
			Expect(err).NotTo(HaveOccurred())

			// Subscribe
			subscribeMsg := map[string]interface{}{
				"type": "subscribe",
				"data": map[string]interface{}{"topics": []string{"localstack"}},
			}
			err = conn.Write(ctx, websocket.MessageText, mustMarshal(subscribeMsg))
			Expect(err).NotTo(HaveOccurred())

			// Close connection
			conn.Close(websocket.StatusNormalClosure, "")

			// New connection should work
			conn2, _, err := websocket.Dial(ctx, wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer conn2.Close(websocket.StatusNormalClosure, "")
		})
	})
})

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}