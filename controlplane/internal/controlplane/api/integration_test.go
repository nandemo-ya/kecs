package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

var _ = Describe("Integration Test - V1 to V2 Migration", func() {
	var (
		server     *api.Server
		testServer *httptest.Server
		storage    storage.Storage
	)

	BeforeEach(func() {
		// Create test storage
		var err error
		storage, err = duckdb.NewDuckDBStorage(":memory:")
		Expect(err).ToNot(HaveOccurred())
		
		err = storage.Initialize(context.Background())
		Expect(err).ToNot(HaveOccurred())

		// Create API server
		server, err = api.NewServer(8080, "", storage, nil)
		Expect(err).ToNot(HaveOccurred())

		// Create test HTTP server
		handler := server.SetupRoutes()
		testServer = httptest.NewServer(handler)
	})

	AfterEach(func() {
		testServer.Close()
		if storage != nil {
			storage.Close()
		}
	})

	Describe("ECS API Requests", func() {
		It("should handle ListClusters request via X-Amz-Target header", func() {
			// Create request
			reqBody := &ecs.ListClustersInput{}
			body, err := json.Marshal(reqBody)
			Expect(err).ToNot(HaveOccurred())

			req, err := http.NewRequest("POST", testServer.URL+"/", bytes.NewReader(body))
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set("Content-Type", "application/x-amz-json-1.1")
			req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.ListClusters")

			// Send request
			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()

			// Check response
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var result ecs.ListClustersOutput
			err = json.NewDecoder(resp.Body).Decode(&result)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ClusterArns).To(BeEmpty())
		})

		It("should handle CreateCluster request via X-Amz-Target header", func() {
			// Create request
			reqBody := &ecs.CreateClusterInput{
				ClusterName: aws.String("test-cluster"),
			}
			body, err := json.Marshal(reqBody)
			Expect(err).ToNot(HaveOccurred())

			req, err := http.NewRequest("POST", testServer.URL+"/", bytes.NewReader(body))
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set("Content-Type", "application/x-amz-json-1.1")
			req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.CreateCluster")

			// Send request
			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()

			// Check response
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var result ecs.CreateClusterOutput
			err = json.NewDecoder(resp.Body).Decode(&result)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Cluster).ToNot(BeNil())
			Expect(*result.Cluster.ClusterName).To(Equal("test-cluster"))
			Expect(*result.Cluster.Status).To(Equal("ACTIVE"))
		})

		It("should handle end-to-end: create and list clusters", func() {
			// Create cluster
			createReq := &ecs.CreateClusterInput{
				ClusterName: aws.String("integration-test"),
			}
			body, err := json.Marshal(createReq)
			Expect(err).ToNot(HaveOccurred())

			req, err := http.NewRequest("POST", testServer.URL+"/", bytes.NewReader(body))
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set("Content-Type", "application/x-amz-json-1.1")
			req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.CreateCluster")

			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			resp.Body.Close()

			// List clusters
			listReq := &ecs.ListClustersInput{}
			body, err = json.Marshal(listReq)
			Expect(err).ToNot(HaveOccurred())

			req, err = http.NewRequest("POST", testServer.URL+"/", bytes.NewReader(body))
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set("Content-Type", "application/x-amz-json-1.1")
			req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.ListClusters")

			resp, err = http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()

			var listResult ecs.ListClustersOutput
			err = json.NewDecoder(resp.Body).Decode(&listResult)
			Expect(err).ToNot(HaveOccurred())
			Expect(listResult.ClusterArns).To(HaveLen(1))
			Expect(listResult.ClusterArns[0]).To(ContainSubstring("integration-test"))
		})
	})
})