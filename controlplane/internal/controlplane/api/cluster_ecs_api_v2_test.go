package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

var _ = Describe("Cluster ECS API V2", func() {
	var (
		ecsAPIV2 *api.DefaultECSAPIV2
		testStorage  storage.Storage
		ctx      context.Context
	)

	BeforeEach(func() {
		// Create test storage
		var err error
		testStorage, err = duckdb.NewDuckDBStorage(":memory:")
		Expect(err).ToNot(HaveOccurred())
		
		// Initialize tables
		err = testStorage.Initialize(context.Background())
		Expect(err).ToNot(HaveOccurred())
		
		ctx = context.Background()
		
		// Initialize V2 API
		ecsAPIV2 = api.NewDefaultECSAPIV2(testStorage, nil)
	})

	AfterEach(func() {
		if testStorage != nil {
			testStorage.Close()
		}
	})

	Describe("ListClustersV2", func() {
		Context("when no clusters exist", func() {
			It("should return empty list", func() {
				req := &ecs.ListClustersInput{}
				resp, err := ecsAPIV2.ListClustersV2(ctx, req)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.ClusterArns).To(BeEmpty())
				Expect(resp.NextToken).To(BeNil())
			})
		})

		Context("when clusters exist", func() {
			BeforeEach(func() {
				// Create test clusters
				for i := 1; i <= 3; i++ {
					cluster := &storage.Cluster{
						Name:   fmt.Sprintf("cluster-%d", i),
						ARN:    fmt.Sprintf("arn:aws:ecs:region:account:cluster/cluster-%d", i),
						Status: "ACTIVE",
					}
					err := testStorage.ClusterStore().Create(ctx, cluster)
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should list all clusters", func() {
				req := &ecs.ListClustersInput{}
				resp, err := ecsAPIV2.ListClustersV2(ctx, req)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.ClusterArns).To(HaveLen(3))
				Expect(resp.ClusterArns).To(ContainElements(
					"arn:aws:ecs:region:account:cluster/cluster-1",
					"arn:aws:ecs:region:account:cluster/cluster-2",
					"arn:aws:ecs:region:account:cluster/cluster-3",
				))
			})

			It("should respect MaxResults parameter", func() {
				req := &ecs.ListClustersInput{
					MaxResults: aws.Int32(2),
				}
				resp, err := ecsAPIV2.ListClustersV2(ctx, req)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.ClusterArns).To(HaveLen(2))
				Expect(resp.NextToken).ToNot(BeNil())
			})

			It("should handle pagination with NextToken", func() {
				// First page
				req1 := &ecs.ListClustersInput{
					MaxResults: aws.Int32(2),
				}
				resp1, err := ecsAPIV2.ListClustersV2(ctx, req1)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp1.NextToken).ToNot(BeNil())

				// Second page
				req2 := &ecs.ListClustersInput{
					MaxResults: aws.Int32(2),
					NextToken:  resp1.NextToken,
				}
				resp2, err := ecsAPIV2.ListClustersV2(ctx, req2)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp2.ClusterArns).To(HaveLen(1))
				Expect(resp2.NextToken).To(BeNil())
			})
		})
	})

	Describe("CreateClusterV2", func() {
		It("should create a new cluster", func() {
			req := &ecs.CreateClusterInput{
				ClusterName: aws.String("test-cluster"),
			}
			resp, err := ecsAPIV2.CreateClusterV2(ctx, req)
			
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Cluster).ToNot(BeNil())
			Expect(*resp.Cluster.ClusterName).To(Equal("test-cluster"))
			Expect(*resp.Cluster.Status).To(Equal("ACTIVE"))
			Expect(*resp.Cluster.ClusterArn).To(ContainSubstring("test-cluster"))
		})

		It("should return existing cluster if name already exists", func() {
			// Create first cluster
			req1 := &ecs.CreateClusterInput{
				ClusterName: aws.String("test-cluster"),
			}
			resp1, err := ecsAPIV2.CreateClusterV2(ctx, req1)
			Expect(err).ToNot(HaveOccurred())

			// Try to create again with same name
			req2 := &ecs.CreateClusterInput{
				ClusterName: aws.String("test-cluster"),
			}
			resp2, err := ecsAPIV2.CreateClusterV2(ctx, req2)
			
			Expect(err).ToNot(HaveOccurred())
			Expect(*resp2.Cluster.ClusterArn).To(Equal(*resp1.Cluster.ClusterArn))
		})

		It("should use default name when not provided", func() {
			req := &ecs.CreateClusterInput{}
			resp, err := ecsAPIV2.CreateClusterV2(ctx, req)
			
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(*resp.Cluster.ClusterName).To(Equal("default"))
		})
	})
})

func TestClusterECSAPIV2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster ECS API V2 Suite")
}