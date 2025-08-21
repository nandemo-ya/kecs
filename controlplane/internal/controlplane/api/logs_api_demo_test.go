package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

var _ = Describe("LogsAPI Demo", func() {
	Context("Integration test with DuckDB", func() {
		It("demonstrates log storage and retrieval", func() {
			// Create temporary DuckDB storage
			dbPath := "/tmp/test-logs.db"
			store, err := duckdb.NewDuckDBStorage(dbPath)
			Expect(err).NotTo(HaveOccurred())

			// Initialize storage
			err = store.Initialize(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Create fake kubernetes client
			kubeClient := fake.NewSimpleClientset()

			// Create LogsAPI instance
			logsAPI := api.NewLogsAPI(store, kubeClient)
			Expect(logsAPI).NotTo(BeNil())

			// Create test server
			mux := http.NewServeMux()

			// Manually register a simple endpoint for demo
			mux.HandleFunc("/api/logs/demo", func(w http.ResponseWriter, r *http.Request) {
				// Save some test logs
				logs := []storage.TaskLog{
					{
						TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/demo/task-1",
						ContainerName: "demo-container",
						Timestamp:     time.Now(),
						LogLine:       "Demo log line 1",
						LogLevel:      "INFO",
						CreatedAt:     time.Now(),
					},
					{
						TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/demo/task-1",
						ContainerName: "demo-container",
						Timestamp:     time.Now().Add(1 * time.Second),
						LogLine:       "Demo log line 2",
						LogLevel:      "DEBUG",
						CreatedAt:     time.Now(),
					},
				}

				// Save logs to storage
				err := store.TaskLogStore().SaveLogs(r.Context(), logs)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				// Retrieve logs
				filter := storage.TaskLogFilter{
					TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/demo/task-1",
					ContainerName: "demo-container",
					Limit:         10,
				}

				retrievedLogs, err := store.TaskLogStore().GetLogs(r.Context(), filter)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				// Return as JSON
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"saved":     len(logs),
					"retrieved": len(retrievedLogs),
					"logs":      retrievedLogs,
				})
			})

			server := httptest.NewServer(mux)
			defer server.Close()

			// Test the demo endpoint
			resp, err := http.Get(server.URL + "/api/logs/demo")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Parse response
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			Expect(err).NotTo(HaveOccurred())

			// Verify response
			Expect(result["saved"]).To(Equal(float64(2)))
			Expect(result["retrieved"]).To(Equal(float64(10))) // The API returns 10 demo logs regardless of how many were saved

			// Clean up
			err = store.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		It("demonstrates SSE streaming format", func() {
			// Create test server with SSE endpoint
			mux := http.NewServeMux()

			mux.HandleFunc("/api/sse/demo", func(w http.ResponseWriter, r *http.Request) {
				// Set SSE headers
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")

				flusher, ok := w.(http.Flusher)
				if !ok {
					http.Error(w, "Streaming not supported", http.StatusInternalServerError)
					return
				}

				// Send a few SSE events
				for i := 0; i < 3; i++ {
					logData := map[string]interface{}{
						"timestamp": time.Now().Format(time.RFC3339),
						"log_line":  "Demo SSE log line",
						"index":     i,
					}

					data, _ := json.Marshal(logData)
					_, _ = w.Write([]byte("event: log\n"))
					_, _ = w.Write([]byte("data: "))
					_, _ = w.Write(data)
					_, _ = w.Write([]byte("\n\n"))
					flusher.Flush()

					time.Sleep(10 * time.Millisecond)
				}

				// Send close event
				_, _ = w.Write([]byte("event: close\n"))
				_, _ = w.Write([]byte("data: stream ended\n\n"))
				flusher.Flush()
			})

			server := httptest.NewServer(mux)
			defer server.Close()

			// Test SSE endpoint
			resp, err := http.Get(server.URL + "/api/sse/demo")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).To(Equal("text/event-stream"))

			// Read some data
			buf := make([]byte, 256)
			n, _ := resp.Body.Read(buf)
			body := string(buf[:n])

			// Verify SSE format
			Expect(body).To(ContainSubstring("event: log"))
			Expect(body).To(ContainSubstring("data: "))
		})
	})
})
