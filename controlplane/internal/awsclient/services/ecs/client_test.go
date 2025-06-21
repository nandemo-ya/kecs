package ecs_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/awsclient"
	"github.com/nandemo-ya/kecs/controlplane/internal/awsclient/services/ecs"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

var _ = Describe("ECS Client", func() {
	var (
		server *httptest.Server
		client *ecs.Client
	)

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
	})

	Describe("ListClusters", func() {
		Context("when the request is successful", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify request
					Expect(r.Method).To(Equal("POST"))
					Expect(r.Header.Get("Content-Type")).To(Equal("application/x-amz-json-1.1"))
					Expect(r.Header.Get("X-Amz-Target")).To(Equal("AmazonEC2ContainerServiceV20141113.ListClusters"))

					// Send response
					response := generated.ListClustersResponse{
						ClusterArns: []string{
							"arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-1",
							"arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-2",
						},
					}

					w.Header().Set("Content-Type", "application/x-amz-json-1.1")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
				}))

				client = ecs.NewClient(awsclient.Config{
					Endpoint: server.URL,
					Credentials: awsclient.Credentials{
						AccessKeyID:     "test-key",
						SecretAccessKey: "test-secret",
					},
					Region: "us-east-1",
				})
			})

			It("should return a list of cluster ARNs", func() {
				input := &generated.ListClustersRequest{}
				output, err := client.ListClusters(context.Background(), input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(output.ClusterArns).To(HaveLen(2))
				Expect(output.ClusterArns[0]).To(ContainSubstring("test-cluster-1"))
				Expect(output.ClusterArns[1]).To(ContainSubstring("test-cluster-2"))
			})
		})
	})

	Describe("CreateCluster", func() {
		Context("when creating a cluster", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify request
					Expect(r.Method).To(Equal("POST"))
					Expect(r.Header.Get("X-Amz-Target")).To(Equal("AmazonEC2ContainerServiceV20141113.CreateCluster"))

					// Parse request
					var req generated.CreateClusterRequest
					err := json.NewDecoder(r.Body).Decode(&req)
					Expect(err).NotTo(HaveOccurred())
					Expect(*req.ClusterName).To(Equal("test-cluster"))

					// Send response
					clusterName := "test-cluster"
					clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster"
					status := "ACTIVE"
					
					response := generated.CreateClusterResponse{
						Cluster: &generated.Cluster{
							ClusterName: &clusterName,
							ClusterArn:  &clusterArn,
							Status:      &status,
						},
					}

					w.Header().Set("Content-Type", "application/x-amz-json-1.1")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
				}))

				client = ecs.NewClient(awsclient.Config{
					Endpoint: server.URL,
					Credentials: awsclient.Credentials{
						AccessKeyID:     "test-key",
						SecretAccessKey: "test-secret",
					},
					Region: "us-east-1",
				})
			})

			It("should create a cluster successfully", func() {
				clusterName := "test-cluster"
				input := &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				}
				output, err := client.CreateCluster(context.Background(), input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(output.Cluster).NotTo(BeNil())
				Expect(*output.Cluster.ClusterName).To(Equal("test-cluster"))
				Expect(*output.Cluster.Status).To(Equal("ACTIVE"))
			})
		})
	})

	Describe("Error Handling", func() {
		Context("when the server returns an error", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Send error response
					errorResponse := map[string]string{
						"__type": "ClusterNotFoundException",
						"message": "The referenced cluster was not found",
					}

					w.Header().Set("Content-Type", "application/x-amz-json-1.1")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(errorResponse)
				}))

				client = ecs.NewClient(awsclient.Config{
					Endpoint: server.URL,
					Credentials: awsclient.Credentials{
						AccessKeyID:     "test-key",
						SecretAccessKey: "test-secret",
					},
					Region: "us-east-1",
				})
			})

			It("should return an appropriate error", func() {
				clusterName := "non-existent-cluster"
				input := &generated.DeleteClusterRequest{
					Cluster: &clusterName,
				}
				_, err := client.DeleteCluster(context.Background(), input)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("ClusterNotFoundException"))
				Expect(err.Error()).To(ContainSubstring("The referenced cluster was not found"))
			})
		})
	})
})