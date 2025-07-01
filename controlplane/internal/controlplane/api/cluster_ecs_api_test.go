package api

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/mocks"
)

var _ = Describe("Cluster ECS API", func() {
	var (
		server           *Server
		ctx              context.Context
		mockStorage      *mocks.MockStorage
		mockClusterStore *mocks.MockClusterStore
	)

	BeforeEach(func() {
		mockStorage = mocks.NewMockStorage()
		mockClusterStore = mocks.NewMockClusterStore()
		mockStorage.SetClusterStore(mockClusterStore)
		server = &Server{
			storage: mockStorage,
			ecsAPI:  NewDefaultECSAPI(mockStorage),
		}
		ctx = context.Background()
	})

	Describe("CreateCluster", func() {
		Context("when creating a cluster with random name", func() {
			It("should create cluster with a specific name", func() {
				clusterName := "test-cluster"
				req := &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				}

				resp, err := server.ecsAPI.CreateCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify response
				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(resp.Cluster.ClusterName).NotTo(BeNil())
				Expect(*resp.Cluster.ClusterName).To(Equal("test-cluster"))

				// Get the cluster from storage to check the k8s cluster name
				cluster, err := mockClusterStore.Get(ctx, "test-cluster")
				Expect(err).NotTo(HaveOccurred())

				// Verify that the k8s cluster name follows the expected pattern
				Expect(cluster.K8sClusterName).To(HavePrefix("kecs-"))

				// Should be kecs-<cluster-name>
				Expect(cluster.K8sClusterName).To(Equal("kecs-test-cluster"))
			})

			It("should create different random names for different clusters", func() {
				// Create first cluster
				clusterName1 := "test-cluster-1"
				req1 := &generated.CreateClusterRequest{
					ClusterName: &clusterName1,
				}
				_, err := server.ecsAPI.CreateCluster(ctx, req1)
				Expect(err).NotTo(HaveOccurred())

				cluster1, err := mockClusterStore.Get(ctx, "test-cluster-1")
				Expect(err).NotTo(HaveOccurred())

				// Create second cluster
				clusterName2 := "test-cluster-2"
				req2 := &generated.CreateClusterRequest{
					ClusterName: &clusterName2,
				}
				_, err = server.ecsAPI.CreateCluster(ctx, req2)
				Expect(err).NotTo(HaveOccurred())

				cluster2, err := mockClusterStore.Get(ctx, "test-cluster-2")
				Expect(err).NotTo(HaveOccurred())

				// Verify the two clusters have different k8s cluster names
				Expect(cluster1.K8sClusterName).NotTo(Equal(cluster2.K8sClusterName))
			})
		})

		Context("when creating a cluster with idempotency", func() {
			It("should return existing cluster when name already exists", func() {
				clusterName := "idempotent-test"
				req := &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				}

				// First call - should create the cluster
				resp1, err := server.ecsAPI.CreateCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify first response
				Expect(resp1).NotTo(BeNil())
				Expect(resp1.Cluster).NotTo(BeNil())
				Expect(resp1.Cluster.ClusterArn).NotTo(BeNil())
				Expect(resp1.Cluster.ClusterName).NotTo(BeNil())
				Expect(resp1.Cluster.Status).NotTo(BeNil())

				clusterArn1 := *resp1.Cluster.ClusterArn
				clusterName1 := *resp1.Cluster.ClusterName
				status1 := *resp1.Cluster.Status

				Expect(clusterName1).To(Equal("idempotent-test"))
				Expect(status1).To(Equal("ACTIVE"))

				// Second call - should return the existing cluster (idempotent)
				resp2, err := server.ecsAPI.CreateCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify second response
				Expect(resp2).NotTo(BeNil())
				Expect(resp2.Cluster).NotTo(BeNil())
				Expect(resp2.Cluster.ClusterArn).NotTo(BeNil())
				Expect(resp2.Cluster.ClusterName).NotTo(BeNil())
				Expect(resp2.Cluster.Status).NotTo(BeNil())

				clusterArn2 := *resp2.Cluster.ClusterArn
				clusterName2 := *resp2.Cluster.ClusterName
				status2 := *resp2.Cluster.Status

				// Verify both responses are identical
				Expect(clusterArn1).To(Equal(clusterArn2))
				Expect(clusterName1).To(Equal(clusterName2))
				Expect(status1).To(Equal(status2))

				// Verify only one cluster exists in storage
				clusters, err := mockClusterStore.List(ctx)
				Expect(err).NotTo(HaveOccurred())

				clusterCount := 0
				for _, cluster := range clusters {
					if cluster.Name == "idempotent-test" {
						clusterCount++
					}
				}

				Expect(clusterCount).To(Equal(1))
			})
		})
	})

	Describe("ListClusters", func() {
		Context("when listing clusters", func() {
			It("should return empty list when no clusters exist", func() {
				req := &generated.ListClustersRequest{}

				resp, err := server.ecsAPI.ListClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.ClusterArns).To(BeEmpty())
			})

			It("should return all cluster ARNs", func() {
				// Create test clusters
				clusterNames := []string{"cluster-1", "cluster-2", "cluster-3"}
				for _, name := range clusterNames {
					clusterName := name
					_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
						ClusterName: &clusterName,
					})
					Expect(err).NotTo(HaveOccurred())
				}

				// List clusters
				req := &generated.ListClustersRequest{}
				resp, err := server.ecsAPI.ListClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.ClusterArns).To(HaveLen(3))

				// Verify all ARNs are present
				arnMap := make(map[string]bool)
				for _, arn := range resp.ClusterArns {
					arnMap[arn] = true
				}

				for _, name := range clusterNames {
					expectedArn := "arn:aws:ecs:us-east-1:123456789012:cluster/" + name
					Expect(arnMap).To(HaveKey(expectedArn))
				}
			})
		})
	})

	Describe("DescribeClusters", func() {
		Context("when describing clusters", func() {
			BeforeEach(func() {
				// Create test clusters
				clusterNames := []string{"describe-test-1", "describe-test-2"}
				for _, name := range clusterNames {
					clusterName := name
					_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
						ClusterName: &clusterName,
					})
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should describe all clusters when no specific clusters requested", func() {
				req := &generated.DescribeClustersRequest{}

				resp, err := server.ecsAPI.DescribeClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Clusters).To(HaveLen(2))
				Expect(resp.Failures).To(BeEmpty())
			})

			It("should describe specific clusters by name", func() {
				req := &generated.DescribeClustersRequest{
					Clusters: []string{"describe-test-1"},
				}

				resp, err := server.ecsAPI.DescribeClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Clusters).To(HaveLen(1))
				Expect(*resp.Clusters[0].ClusterName).To(Equal("describe-test-1"))
				Expect(resp.Failures).To(BeEmpty())
			})

			It("should return failure for non-existent cluster", func() {
				req := &generated.DescribeClustersRequest{
					Clusters: []string{"non-existent-cluster"},
				}

				resp, err := server.ecsAPI.DescribeClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Clusters).To(BeEmpty())
				Expect(resp.Failures).To(HaveLen(1))
				Expect(*resp.Failures[0].Reason).To(Equal("MISSING"))
			})

			It("should describe clusters by ARN", func() {
				// First create the cluster
				cluster, err := mockClusterStore.Get(ctx, "describe-test-1")
				Expect(err).NotTo(HaveOccurred())

				// Use ARN to describe
				req := &generated.DescribeClustersRequest{
					Clusters: []string{cluster.ARN},
				}

				resp, err := server.ecsAPI.DescribeClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Clusters).To(HaveLen(1))
				Expect(*resp.Clusters[0].ClusterName).To(Equal("describe-test-1"))
				Expect(*resp.Clusters[0].ClusterArn).To(Equal(cluster.ARN))
				Expect(resp.Failures).To(BeEmpty())
			})
		})
	})

	Describe("DeleteCluster", func() {
		Context("when deleting a cluster", func() {
			It("should delete an existing cluster", func() {
				// Create a cluster first
				clusterName := "delete-test"
				_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				})
				Expect(err).NotTo(HaveOccurred())

				// Delete the cluster
				req := &generated.DeleteClusterRequest{
					Cluster: clusterName,
				}

				resp, err := server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(*resp.Cluster.ClusterName).To(Equal("delete-test"))
				Expect(*resp.Cluster.Status).To(Equal("INACTIVE"))

				// Verify cluster is deleted from storage
				_, err = mockClusterStore.Get(ctx, "delete-test")
				Expect(err).To(HaveOccurred())
			})

			It("should fail when cluster has active services", func() {
				// Create a cluster with active services count
				clusterName := "cluster-with-services"
				_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				})
				Expect(err).NotTo(HaveOccurred())

				// Simulate active services by updating the cluster
				cluster, err := mockClusterStore.Get(ctx, clusterName)
				Expect(err).NotTo(HaveOccurred())
				cluster.ActiveServicesCount = 1
				err = mockClusterStore.Update(ctx, cluster)
				Expect(err).NotTo(HaveOccurred())

				// Try to delete the cluster
				req := &generated.DeleteClusterRequest{
					Cluster: clusterName,
				}

				_, err = server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("The cluster cannot be deleted while services are active"))
			})

			It("should fail when cluster does not exist", func() {
				clusterName := "non-existent"
				req := &generated.DeleteClusterRequest{
					Cluster: clusterName,
				}

				_, err := server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})

			It("should delete a cluster by ARN", func() {
				// Create a cluster first
				clusterName := "delete-by-arn-test"
				createResp, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				})
				Expect(err).NotTo(HaveOccurred())

				// Get the ARN
				clusterArn := *createResp.Cluster.ClusterArn

				// Delete the cluster using ARN
				req := &generated.DeleteClusterRequest{
					Cluster: clusterArn,
				}

				resp, err := server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(*resp.Cluster.ClusterName).To(Equal("delete-by-arn-test"))
				Expect(*resp.Cluster.Status).To(Equal("INACTIVE"))

				// Verify cluster is deleted from storage
				_, err = mockClusterStore.Get(ctx, "delete-by-arn-test")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("UpdateCluster", func() {
		Context("when updating a cluster", func() {
			BeforeEach(func() {
				// Create a test cluster
				clusterName := "update-test"
				_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should update cluster settings", func() {
				clusterName := "update-test"
				settingName := generated.ClusterSettingNameCONTAINER_INSIGHTS
				settingValue := "enabled"
				settings := []generated.ClusterSetting{
					{
						Name:  &settingName,
						Value: &settingValue,
					},
				}

				req := &generated.UpdateClusterRequest{
					Cluster:  clusterName,
					Settings: settings,
				}

				resp, err := server.ecsAPI.UpdateCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(resp.Cluster.Settings).To(HaveLen(1))
				Expect(*resp.Cluster.Settings[0].Name).To(Equal(settingName))
				Expect(*resp.Cluster.Settings[0].Value).To(Equal("enabled"))
			})

			It("should update cluster configuration", func() {
				clusterName := "update-test"
				loggingValue := generated.ExecuteCommandLoggingDEFAULT
				config := &generated.ClusterConfiguration{
					ExecuteCommandConfiguration: &generated.ExecuteCommandConfiguration{
						Logging: &loggingValue,
					},
				}

				req := &generated.UpdateClusterRequest{
					Cluster:       clusterName,
					Configuration: config,
				}

				resp, err := server.ecsAPI.UpdateCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(resp.Cluster.Configuration).NotTo(BeNil())
				Expect(resp.Cluster.Configuration.ExecuteCommandConfiguration).NotTo(BeNil())
				Expect(*resp.Cluster.Configuration.ExecuteCommandConfiguration.Logging).To(Equal(generated.ExecuteCommandLoggingDEFAULT))
			})

			It("should fail when cluster does not exist", func() {
				clusterName := "non-existent"
				req := &generated.UpdateClusterRequest{
					Cluster: clusterName,
				}

				_, err := server.ecsAPI.UpdateCluster(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})
	})

	Describe("UpdateClusterSettings", func() {
		Context("when updating cluster settings", func() {
			BeforeEach(func() {
				// Create a test cluster with initial settings
				clusterName := "settings-test"
				settingName := generated.ClusterSettingNameCONTAINER_INSIGHTS
				settingValue := "disabled"
				_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
					ClusterName: &clusterName,
					Settings: []generated.ClusterSetting{
						{
							Name:  &settingName,
							Value: &settingValue,
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should update existing cluster settings", func() {
				clusterName := "settings-test"
				settingName := generated.ClusterSettingNameCONTAINER_INSIGHTS
				settingValue := "enabled"
				settings := []generated.ClusterSetting{
					{
						Name:  &settingName,
						Value: &settingValue,
					},
				}

				req := &generated.UpdateClusterSettingsRequest{
					Cluster:  clusterName,
					Settings: settings,
				}

				resp, err := server.ecsAPI.UpdateClusterSettings(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(resp.Cluster.Settings).To(HaveLen(1))
				Expect(*resp.Cluster.Settings[0].Name).To(Equal(settingName))
				Expect(*resp.Cluster.Settings[0].Value).To(Equal("enabled"))
			})

			It("should fail when settings are not provided", func() {
				clusterName := "settings-test"
				req := &generated.UpdateClusterSettingsRequest{
					Cluster: clusterName,
				}

				_, err := server.ecsAPI.UpdateClusterSettings(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("settings are required"))
			})
		})
	})

	Describe("PutClusterCapacityProviders", func() {
		Context("when updating cluster capacity providers", func() {
			BeforeEach(func() {
				// Create a test cluster
				clusterName := "capacity-test"
				_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should update capacity providers and strategy", func() {
				clusterName := "capacity-test"
				capacityProviders := []string{"FARGATE", "FARGATE_SPOT"}
				strategy := []generated.CapacityProviderStrategyItem{
					{
						CapacityProvider: "FARGATE",
						Weight:           ptr.Int32(1),
						Base:             ptr.Int32(1),
					},
					{
						CapacityProvider: "FARGATE_SPOT",
						Weight:           ptr.Int32(4),
					},
				}

				req := &generated.PutClusterCapacityProvidersRequest{
					Cluster:                         clusterName,
					CapacityProviders:               capacityProviders,
					DefaultCapacityProviderStrategy: strategy,
				}

				resp, err := server.ecsAPI.PutClusterCapacityProviders(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(resp.Cluster.CapacityProviders).To(Equal(capacityProviders))
				Expect(resp.Cluster.DefaultCapacityProviderStrategy).To(HaveLen(2))
			})

			It("should fail when required fields are missing", func() {
				clusterName := "capacity-test"
				req := &generated.PutClusterCapacityProvidersRequest{
					Cluster: clusterName,
				}

				_, err := server.ecsAPI.PutClusterCapacityProviders(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("capacityProviders is required"))
			})
		})
	})

	Describe("extractClusterNameFromARN", func() {
		It("should extract cluster name from valid ARN", func() {
			arn := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"
			name := extractClusterNameFromARN(arn)
			Expect(name).To(Equal("my-cluster"))
		})

		It("should return input for non-ARN strings", func() {
			name := extractClusterNameFromARN("my-cluster")
			Expect(name).To(Equal("my-cluster"))
		})

		It("should return input for invalid ARN format", func() {
			// Missing cluster name after slash
			arn := "arn:aws:ecs:us-east-1:123456789012:cluster/"
			name := extractClusterNameFromARN(arn)
			Expect(name).To(Equal(arn))

			// No slash
			arn2 := "arn:aws:ecs:us-east-1:123456789012:cluster"
			name2 := extractClusterNameFromARN(arn2)
			Expect(name2).To(Equal(arn2))

			// Multiple slashes
			arn3 := "arn:aws:ecs:us-east-1:123456789012:cluster/my/cluster"
			name3 := extractClusterNameFromARN(arn3)
			Expect(name3).To(Equal(arn3))
		})

		It("should handle empty string", func() {
			name := extractClusterNameFromARN("")
			Expect(name).To(Equal(""))
		})
	})

	// Validation tests moved from phase1 error scenarios
	Describe("CreateCluster Validation", func() {
		Context("when cluster name is invalid", func() {
			It("should reject cluster names that are too long", func() {
				// AWS ECS cluster names must be 1-255 characters
				longName := strings.Repeat("a", 256)
				req := &generated.CreateClusterRequest{
					ClusterName: &longName,
				}

				_, err := server.ecsAPI.CreateCluster(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cluster name must be between 1 and 255 characters"))
			})

			It("should reject empty cluster names", func() {
				emptyName := ""
				req := &generated.CreateClusterRequest{
					ClusterName: &emptyName,
				}

				_, err := server.ecsAPI.CreateCluster(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cluster name must be between 1 and 255 characters"))
			})

			It("should reject cluster names with invalid characters", func() {
				invalidNames := []string{
					"cluster@name",  // @ symbol
					"cluster name",  // space
					"cluster/name",  // slash
					"cluster\\name", // backslash
					"cluster:name",  // colon
					"cluster*name",  // asterisk
					"cluster?name",  // question mark
					"cluster#name",  // hash
					"cluster%name",  // percent
				}

				for _, name := range invalidNames {
					clusterName := name
					req := &generated.CreateClusterRequest{
						ClusterName: &clusterName,
					}

					_, err := server.ecsAPI.CreateCluster(ctx, req)
					Expect(err).To(HaveOccurred(), "Should fail for cluster name: %s", name)
					Expect(err.Error()).To(ContainSubstring("cluster name can only contain alphanumeric characters, dashes, and underscores"))
				}
			})
		})
	})

	Describe("DeleteCluster Validation", func() {
		Context("when cluster identifier is invalid", func() {
			It("should reject empty cluster identifier", func() {
				req := &generated.DeleteClusterRequest{
					Cluster: "",
				}

				_, err := server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cluster identifier is required"))
			})

			It("should reject malformed ARN", func() {
				req := &generated.DeleteClusterRequest{
					Cluster: "arn:invalid:format",
				}

				_, err := server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid ARN format"))
			})
		})
	})

	Describe("DescribeClusters Validation", func() {
		Context("when cluster identifiers are invalid", func() {
			It("should handle empty cluster identifier", func() {
				req := &generated.DescribeClustersRequest{
					Clusters: []string{""},
				}

				resp, err := server.ecsAPI.DescribeClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred()) // DescribeClusters should not error, but return failure
				Expect(resp.Failures).To(HaveLen(1))
				Expect(*resp.Failures[0].Reason).To(Equal("MISSING"))
			})

			It("should handle malformed ARNs", func() {
				invalidArns := []string{
					"not-an-arn",
					"arn:aws:ecs",                        // incomplete ARN
					"arn:aws:ecs:us-east-1",              // missing account and resource
					"arn:aws:ecs:us-east-1:123456789012", // missing resource
					"arn:aws:ecs:us-east-1:123456789012:wrongtype/cluster-name", // wrong resource type
				}

				req := &generated.DescribeClustersRequest{
					Clusters: invalidArns,
				}

				resp, err := server.ecsAPI.DescribeClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred()) // DescribeClusters should not error, but return failures
				Expect(resp.Failures).To(HaveLen(len(invalidArns)))
				for _, failure := range resp.Failures {
					Expect(*failure.Reason).To(Equal("MISSING"))
				}
			})
		})
	})

	Describe("UpdateCluster Validation", func() {
		Context("when cluster identifier is invalid", func() {
			It("should reject empty cluster identifier", func() {
				req := &generated.UpdateClusterRequest{
					Cluster: "",
				}

				_, err := server.ecsAPI.UpdateCluster(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cluster identifier is required"))
			})
		})
	})

	Describe("UpdateClusterSettings Validation", func() {
		Context("when settings are invalid", func() {
			BeforeEach(func() {
				// Create a test cluster
				clusterName := "settings-validation-test"
				_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject invalid setting names", func() {
				clusterName := "settings-validation-test"
				invalidSettingName := "invalidSettingName"
				settingValue := "enabled"
				settings := []generated.ClusterSetting{
					{
						Name:  (*generated.ClusterSettingName)(&invalidSettingName),
						Value: &settingValue,
					},
				}

				req := &generated.UpdateClusterSettingsRequest{
					Cluster:  clusterName,
					Settings: settings,
				}

				_, err := server.ecsAPI.UpdateClusterSettings(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid setting name"))
			})

			It("should reject invalid setting values", func() {
				clusterName := "settings-validation-test"
				settingName := generated.ClusterSettingNameCONTAINER_INSIGHTS
				invalidValue := "invalid-value" // Should be "enabled" or "disabled"
				settings := []generated.ClusterSetting{
					{
						Name:  &settingName,
						Value: &invalidValue,
					},
				}

				req := &generated.UpdateClusterSettingsRequest{
					Cluster:  clusterName,
					Settings: settings,
				}

				_, err := server.ecsAPI.UpdateClusterSettings(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid value for containerInsights"))
			})
		})
	})

	Describe("PutClusterCapacityProviders Validation", func() {
		Context("when capacity providers are invalid", func() {
			BeforeEach(func() {
				// Create a test cluster
				clusterName := "capacity-validation-test"
				_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject invalid capacity provider names", func() {
				clusterName := "capacity-validation-test"
				invalidProviders := []string{"INVALID_PROVIDER", "ANOTHER_INVALID"}

				req := &generated.PutClusterCapacityProvidersRequest{
					Cluster:           clusterName,
					CapacityProviders: invalidProviders,
				}

				_, err := server.ecsAPI.PutClusterCapacityProviders(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid capacity provider"))
			})

			It("should reject invalid capacity provider strategy", func() {
				clusterName := "capacity-validation-test"
				providers := []string{"FARGATE"}
				strategy := []generated.CapacityProviderStrategyItem{
					{
						CapacityProvider: "FARGATE",
						Weight:           ptr.Int32(-1), // Invalid: negative weight
						Base:             ptr.Int32(-5), // Invalid: negative base
					},
				}

				req := &generated.PutClusterCapacityProvidersRequest{
					Cluster:                         clusterName,
					CapacityProviders:               providers,
					DefaultCapacityProviderStrategy: strategy,
				}

				_, err := server.ecsAPI.PutClusterCapacityProviders(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("weight must be between 0 and 1000"))
			})

			It("should reject strategy weight greater than 1000", func() {
				clusterName := "capacity-validation-test"
				providers := []string{"FARGATE"}
				strategy := []generated.CapacityProviderStrategyItem{
					{
						CapacityProvider: "FARGATE",
						Weight:           ptr.Int32(1001), // Invalid: > 1000
					},
				}

				req := &generated.PutClusterCapacityProvidersRequest{
					Cluster:                         clusterName,
					CapacityProviders:               providers,
					DefaultCapacityProviderStrategy: strategy,
				}

				_, err := server.ecsAPI.PutClusterCapacityProviders(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("weight must be between 0 and 1000"))
			})
		})
	})

	Describe("TagResource/UntagResource Validation", func() {
		Context("when resource ARN is invalid", func() {
			It("should reject empty resource ARN for TagResource", func() {
				tags := []generated.Tag{
					{
						Key:   ptr.String("Environment"),
						Value: ptr.String("test"),
					},
				}

				req := &generated.TagResourceRequest{
					ResourceArn: "",
					Tags:        tags,
				}

				_, err := server.ecsAPI.TagResource(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("resource ARN is required"))
			})

			It("should reject empty resource ARN for UntagResource", func() {
				tagKeys := []string{"Environment"}

				req := &generated.UntagResourceRequest{
					ResourceArn: "",
					TagKeys:     tagKeys,
				}

				_, err := server.ecsAPI.UntagResource(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("resource ARN is required"))
			})

			It("should reject empty resource ARN for ListTagsForResource", func() {
				req := &generated.ListTagsForResourceRequest{
					ResourceArn: "",
				}

				_, err := server.ecsAPI.ListTagsForResource(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("resource ARN is required"))
			})
		})
	})
})
