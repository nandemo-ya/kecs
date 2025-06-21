package api_test

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

var _ = Describe("Cluster ECS API V2", func() {
	var (
		ecsAPIV2    api.ECSAPIV2
		testStorage storage.Storage
		ctx         context.Context
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
		
		// Initialize generated API and V2 adapter
		generatedAPI := api.NewDefaultECSAPI(testStorage, nil)
		ecsAPIV2 = api.NewECSAPIv2Adapter(generatedAPI)
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

	Describe("DescribeClustersV2", func() {
		Context("when clusters exist", func() {
			BeforeEach(func() {
				// Create test clusters
				clusters := []*storage.Cluster{
					{
						Name:                "test-cluster-1",
						ARN:                 "arn:aws:ecs:region:account:cluster/test-cluster-1",
						Status:              "ACTIVE",
						ActiveServicesCount: 2,
						RunningTasksCount:   5,
						Settings:            `[{"name":"containerInsights","value":"enabled"}]`,
						Tags:                `[{"key":"Environment","value":"test"},{"key":"Team","value":"platform"}]`,
					},
					{
						Name:   "test-cluster-2",
						ARN:    "arn:aws:ecs:region:account:cluster/test-cluster-2",
						Status: "ACTIVE",
					},
				}
				for _, cluster := range clusters {
					err := testStorage.ClusterStore().Create(ctx, cluster)
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should describe specific clusters", func() {
				req := &ecs.DescribeClustersInput{
					Clusters: []string{"test-cluster-1"},
				}
				resp, err := ecsAPIV2.DescribeClustersV2(ctx, req)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.Clusters).To(HaveLen(1))
				Expect(*resp.Clusters[0].ClusterName).To(Equal("test-cluster-1"))
				Expect(resp.Clusters[0].ActiveServicesCount).To(Equal(int32(2)))
				Expect(resp.Clusters[0].RunningTasksCount).To(Equal(int32(5)))
			})

			It("should describe all clusters when none specified", func() {
				req := &ecs.DescribeClustersInput{}
				resp, err := ecsAPIV2.DescribeClustersV2(ctx, req)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.Clusters).To(HaveLen(2))
			})

			It("should include settings and tags when requested", func() {
				req := &ecs.DescribeClustersInput{
					Clusters: []string{"test-cluster-1"},
					Include: []ecstypes.ClusterField{
						ecstypes.ClusterFieldSettings,
						ecstypes.ClusterFieldTags,
					},
				}
				resp, err := ecsAPIV2.DescribeClustersV2(ctx, req)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.Clusters).To(HaveLen(1))
				Expect(resp.Clusters[0].Settings).To(HaveLen(1))
				Expect(string(resp.Clusters[0].Settings[0].Name)).To(Equal("containerInsights"))
				Expect(*resp.Clusters[0].Settings[0].Value).To(Equal("enabled"))
				Expect(resp.Clusters[0].Tags).To(HaveLen(2))
			})

			It("should handle non-existent clusters", func() {
				req := &ecs.DescribeClustersInput{
					Clusters: []string{"test-cluster-1", "non-existent"},
				}
				resp, err := ecsAPIV2.DescribeClustersV2(ctx, req)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.Clusters).To(HaveLen(1))
				Expect(resp.Failures).To(HaveLen(1))
				Expect(*resp.Failures[0].Arn).To(Equal("non-existent"))
				Expect(*resp.Failures[0].Reason).To(Equal("MISSING"))
			})
		})
	})

	Describe("DeleteClusterV2", func() {
		Context("when cluster exists", func() {
			BeforeEach(func() {
				cluster := &storage.Cluster{
					Name:   "test-cluster",
					ARN:    "arn:aws:ecs:region:account:cluster/test-cluster",
					Status: "ACTIVE",
				}
				err := testStorage.ClusterStore().Create(ctx, cluster)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should delete an empty cluster", func() {
				req := &ecs.DeleteClusterInput{
					Cluster: aws.String("test-cluster"),
				}
				resp, err := ecsAPIV2.DeleteClusterV2(ctx, req)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(*resp.Cluster.ClusterName).To(Equal("test-cluster"))
				Expect(*resp.Cluster.Status).To(Equal("INACTIVE"))
				
				// Verify cluster is deleted
				_, err = testStorage.ClusterStore().Get(ctx, "test-cluster")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when cluster has active resources", func() {
			BeforeEach(func() {
				cluster := &storage.Cluster{
					Name:                "busy-cluster",
					ARN:                 "arn:aws:ecs:region:account:cluster/busy-cluster",
					Status:              "ACTIVE",
					ActiveServicesCount: 1,
				}
				err := testStorage.ClusterStore().Create(ctx, cluster)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should fail to delete cluster with active services", func() {
				req := &ecs.DeleteClusterInput{
					Cluster: aws.String("busy-cluster"),
				}
				_, err := ecsAPIV2.DeleteClusterV2(ctx, req)
				
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("active services or tasks"))
			})
		})

		It("should fail when cluster not found", func() {
			req := &ecs.DeleteClusterInput{
				Cluster: aws.String("non-existent"),
			}
			_, err := ecsAPIV2.DeleteClusterV2(ctx, req)
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cluster not found"))
		})
	})

	Describe("UpdateClusterV2", func() {
		Context("when cluster exists", func() {
			BeforeEach(func() {
				cluster := &storage.Cluster{
					Name:   "test-cluster",
					ARN:    "arn:aws:ecs:region:account:cluster/test-cluster",
					Status: "ACTIVE",
				}
				err := testStorage.ClusterStore().Create(ctx, cluster)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should update cluster settings", func() {
				req := &ecs.UpdateClusterInput{
					Cluster: aws.String("test-cluster"),
					Settings: []ecstypes.ClusterSetting{
						{
							Name:  ecstypes.ClusterSettingNameContainerInsights,
							Value: aws.String("enabled"),
						},
					},
				}
				resp, err := ecsAPIV2.UpdateClusterV2(ctx, req)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(*resp.Cluster.ClusterName).To(Equal("test-cluster"))
				Expect(resp.Cluster.Settings).To(HaveLen(1))
				Expect(string(resp.Cluster.Settings[0].Name)).To(Equal("containerInsights"))
				Expect(*resp.Cluster.Settings[0].Value).To(Equal("enabled"))
			})

			It("should update cluster configuration", func() {
				req := &ecs.UpdateClusterInput{
					Cluster: aws.String("test-cluster"),
					Configuration: &ecstypes.ClusterConfiguration{
						ExecuteCommandConfiguration: &ecstypes.ExecuteCommandConfiguration{
							Logging: ecstypes.ExecuteCommandLoggingDefault,
						},
					},
				}
				resp, err := ecsAPIV2.UpdateClusterV2(ctx, req)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.Cluster.Configuration).ToNot(BeNil())
			})
		})

		It("should fail when cluster not found", func() {
			req := &ecs.UpdateClusterInput{
				Cluster: aws.String("non-existent"),
			}
			_, err := ecsAPIV2.UpdateClusterV2(ctx, req)
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cluster not found"))
		})
	})
})