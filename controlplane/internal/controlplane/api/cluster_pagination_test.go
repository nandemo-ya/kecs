package api_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/mocks"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("Cluster Pagination", func() {
	var (
		ctx          context.Context
		mockStorage  *mocks.MockStorage
		clusterStore *mocks.MockClusterStore
		ecsAPI       generated.ECSAPIInterface
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Setup mock storage
		mockStorage = mocks.NewMockStorage()
		clusterStore = mocks.NewMockClusterStore()
		mockStorage.SetClusterStore(clusterStore)

		// Create test clusters
		for i := 0; i < 250; i++ {
			cluster := &storage.Cluster{
				ID:        fmt.Sprintf("cluster-id-%03d", i), // Use predictable IDs for consistent sorting
				ARN:       fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-%03d", i),
				Name:      fmt.Sprintf("test-cluster-%03d", i),
				Status:    "ACTIVE",
				Region:    "us-east-1",
				AccountID: "123456789012",
				CreatedAt: time.Now().Add(time.Duration(-i) * time.Hour),
				UpdatedAt: time.Now(),
			}
			err := clusterStore.Create(ctx, cluster)
			Expect(err).NotTo(HaveOccurred())
		}

		// Create ECS API instance
		ecsAPI = api.NewDefaultECSAPIWithConfig(mockStorage, "us-east-1", "123456789012")
	})

	Describe("ListClusters", func() {
		Context("with pagination", func() {
			It("should list all clusters without pagination", func() {
				req := &generated.ListClustersRequest{}

				resp, err := ecsAPI.ListClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
				Expect(resp.ClusterArns).To(HaveLen(100)) // Default limit
				Expect(resp.NextToken).NotTo(BeNil())
			})

			It("should list clusters with maxResults=5", func() {
				req := &generated.ListClustersRequest{
					MaxResults: ptr.Int32(5),
				}

				resp, err := ecsAPI.ListClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
				Expect(resp.ClusterArns).To(HaveLen(5))
				Expect(resp.NextToken).NotTo(BeNil())

				// Verify ARNs are in expected order
				for i := 0; i < 5; i++ {
					expectedARN := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-%03d", i)
					Expect(resp.ClusterArns[i]).To(Equal(expectedARN))
				}
			})

			It("should list next page", func() {
				// First page
				req1 := &generated.ListClustersRequest{
					MaxResults: ptr.Int32(10),
				}
				resp1, err := ecsAPI.ListClusters(ctx, req1)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp1.ClusterArns).To(HaveLen(10))
				Expect(resp1.NextToken).NotTo(BeNil())

				// Second page
				req2 := &generated.ListClustersRequest{
					MaxResults: ptr.Int32(10),
					NextToken:  resp1.NextToken,
				}
				resp2, err := ecsAPI.ListClusters(ctx, req2)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp2).NotTo(BeNil())
				Expect(resp2.ClusterArns).To(HaveLen(10))
				Expect(resp2.NextToken).NotTo(BeNil())

				// Verify different results
				Expect(resp2.ClusterArns[0]).NotTo(Equal(resp1.ClusterArns[0]))
				expectedARN := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-%03d", 10)
				Expect(resp2.ClusterArns[0]).To(Equal(expectedARN))
			})

			It("should handle last page", func() {
				// Navigate to near the end
				req := &generated.ListClustersRequest{
					MaxResults: ptr.Int32(100),
				}

				var nextToken *string
				for i := 0; i < 2; i++ {
					req.NextToken = nextToken
					resp, err := ecsAPI.ListClusters(ctx, req)
					Expect(err).NotTo(HaveOccurred())
					nextToken = resp.NextToken
				}

				// Last page
				req.NextToken = nextToken
				resp, err := ecsAPI.ListClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
				Expect(resp.ClusterArns).To(HaveLen(50)) // 250 total - 200 = 50
				Expect(resp.NextToken).To(BeNil())
			})

			It("should return error for invalid next token", func() {
				req := &generated.ListClustersRequest{
					NextToken: ptr.String("invalid-token"),
				}

				resp, err := ecsAPI.ListClusters(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(resp).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("invalid pagination token"))
			})

			It("should cap maxResults at 100", func() {
				req := &generated.ListClustersRequest{
					MaxResults: ptr.Int32(200),
				}

				resp, err := ecsAPI.ListClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
				Expect(resp.ClusterArns).To(HaveLen(100)) // Capped at 100
			})
		})

		Context("pagination consistency", func() {
			It("should return consistent results across pages", func() {
				// Collect all clusters using pagination
				var allClusters []string
				var nextToken *string

				for {
					req := &generated.ListClustersRequest{
						MaxResults: ptr.Int32(50),
						NextToken:  nextToken,
					}

					resp, err := ecsAPI.ListClusters(ctx, req)
					Expect(err).NotTo(HaveOccurred())

					allClusters = append(allClusters, resp.ClusterArns...)

					if resp.NextToken == nil {
						break
					}
					nextToken = resp.NextToken
				}

				// Verify we got all clusters
				Expect(allClusters).To(HaveLen(250))

				// Verify order is consistent
				for i := 0; i < 250; i++ {
					expectedARN := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-%03d", i)
					Expect(allClusters[i]).To(Equal(expectedARN))
				}

				// Verify no duplicates
				seen := make(map[string]bool)
				for _, arn := range allClusters {
					Expect(seen[arn]).To(BeFalse(), "Found duplicate ARN: %s", arn)
					seen[arn] = true
				}
			})
		})
	})
})
