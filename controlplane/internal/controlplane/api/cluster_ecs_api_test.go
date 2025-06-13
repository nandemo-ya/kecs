package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// MockStorage implements a simple in-memory storage for testing
type MockStorage struct {
	clusters        map[string]*storage.Cluster
	taskDefinitions map[string]*storage.TaskDefinition
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		clusters:        make(map[string]*storage.Cluster),
		taskDefinitions: make(map[string]*storage.TaskDefinition),
	}
}

func (m *MockStorage) ClusterStore() storage.ClusterStore {
	return &MockClusterStore{storage: m}
}

func (m *MockStorage) ServiceStore() storage.ServiceStore {
	return nil // Not needed for this test
}

func (m *MockStorage) TaskDefinitionStore() storage.TaskDefinitionStore {
	return &MockTaskDefinitionStore{storage: m}
}

func (m *MockStorage) TaskStore() storage.TaskStore {
	return nil // Not needed for this test
}

func (m *MockStorage) AccountSettingStore() storage.AccountSettingStore {
	return nil // Not needed for this test
}

func (m *MockStorage) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockStorage) Close() error {
	return nil
}

func (m *MockStorage) BeginTx(ctx context.Context) (storage.Transaction, error) {
	return &MockTransaction{}, nil
}

// MockTransaction implements storage.Transaction
type MockTransaction struct{}

func (t *MockTransaction) Commit() error   { return nil }
func (t *MockTransaction) Rollback() error { return nil }

type MockClusterStore struct {
	storage *MockStorage
}

func (m *MockClusterStore) Create(ctx context.Context, cluster *storage.Cluster) error {
	if _, exists := m.storage.clusters[cluster.Name]; exists {
		return errors.New("cluster already exists")
	}
	m.storage.clusters[cluster.Name] = cluster
	return nil
}

func (m *MockClusterStore) Get(ctx context.Context, clusterName string) (*storage.Cluster, error) {
	cluster, exists := m.storage.clusters[clusterName]
	if !exists {
		return nil, errors.New("cluster not found")
	}
	return cluster, nil
}

func (m *MockClusterStore) List(ctx context.Context) ([]*storage.Cluster, error) {
	clusters := make([]*storage.Cluster, 0, len(m.storage.clusters))
	for _, cluster := range m.storage.clusters {
		clusters = append(clusters, cluster)
	}
	return clusters, nil
}

func (m *MockClusterStore) Update(ctx context.Context, cluster *storage.Cluster) error {
	m.storage.clusters[cluster.Name] = cluster
	return nil
}

func (m *MockClusterStore) Delete(ctx context.Context, clusterName string) error {
	delete(m.storage.clusters, clusterName)
	return nil
}

var _ = Describe("Cluster ECS API", func() {
	var (
		server *Server
		ctx    context.Context
	)

	BeforeEach(func() {
		mockStorage := NewMockStorage()
		server = &Server{
			storage:     mockStorage,
			kindManager: nil, // Skip actual kind cluster creation in tests
			ecsAPI:      NewDefaultECSAPI(mockStorage, nil),
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

				// Get the cluster from storage to check the kind cluster name
				cluster, err := server.storage.ClusterStore().Get(ctx, "test-cluster")
				Expect(err).NotTo(HaveOccurred())

				// Verify that the kind cluster name follows the expected pattern
				Expect(cluster.KindClusterName).To(HavePrefix("kecs-"))
				
				// Should be kecs-<cluster-name>
				Expect(cluster.KindClusterName).To(Equal("kecs-test-cluster"))
			})

			It("should create different random names for different clusters", func() {
				// Create first cluster
				clusterName1 := "test-cluster-1"
				req1 := &generated.CreateClusterRequest{
					ClusterName: &clusterName1,
				}
				_, err := server.ecsAPI.CreateCluster(ctx, req1)
				Expect(err).NotTo(HaveOccurred())

				cluster1, err := server.storage.ClusterStore().Get(ctx, "test-cluster-1")
				Expect(err).NotTo(HaveOccurred())

				// Create second cluster
				clusterName2 := "test-cluster-2"
				req2 := &generated.CreateClusterRequest{
					ClusterName: &clusterName2,
				}
				_, err = server.ecsAPI.CreateCluster(ctx, req2)
				Expect(err).NotTo(HaveOccurred())

				cluster2, err := server.storage.ClusterStore().Get(ctx, "test-cluster-2")
				Expect(err).NotTo(HaveOccurred())

				// Verify the two clusters have different kind cluster names
				Expect(cluster1.KindClusterName).NotTo(Equal(cluster2.KindClusterName))
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
				clusters, err := server.storage.ClusterStore().List(ctx)
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
					expectedArn := "arn:aws:ecs:ap-northeast-1:123456789012:cluster/" + name
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
				cluster, err := server.storage.ClusterStore().Get(ctx, "describe-test-1")
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
					Cluster: &clusterName,
				}
				
				resp, err := server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(*resp.Cluster.ClusterName).To(Equal("delete-test"))
				Expect(*resp.Cluster.Status).To(Equal("INACTIVE"))

				// Verify cluster is deleted from storage
				_, err = server.storage.ClusterStore().Get(ctx, "delete-test")
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
				cluster, err := server.storage.ClusterStore().Get(ctx, clusterName)
				Expect(err).NotTo(HaveOccurred())
				cluster.ActiveServicesCount = 1
				err = server.storage.ClusterStore().Update(ctx, cluster)
				Expect(err).NotTo(HaveOccurred())

				// Try to delete the cluster
				req := &generated.DeleteClusterRequest{
					Cluster: &clusterName,
				}
				
				_, err = server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("active services"))
			})

			It("should fail when cluster does not exist", func() {
				clusterName := "non-existent"
				req := &generated.DeleteClusterRequest{
					Cluster: &clusterName,
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
					Cluster: &clusterArn,
				}
				
				resp, err := server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(*resp.Cluster.ClusterName).To(Equal("delete-by-arn-test"))
				Expect(*resp.Cluster.Status).To(Equal("INACTIVE"))
				
				// Verify cluster is deleted from storage
				_, err = server.storage.ClusterStore().Get(ctx, "delete-by-arn-test")
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
				settingName := generated.ClusterSettingNameContainerInsights
				settingValue := "enabled"
				settings := []generated.ClusterSetting{
					{
						Name:  &settingName,
						Value: &settingValue,
					},
				}

				req := &generated.UpdateClusterRequest{
					Cluster:  &clusterName,
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
				loggingValue := generated.ExecuteCommandLoggingDefault
				config := &generated.ClusterConfiguration{
					ExecuteCommandConfiguration: &generated.ExecuteCommandConfiguration{
						Logging: &loggingValue,
					},
				}

				req := &generated.UpdateClusterRequest{
					Cluster:       &clusterName,
					Configuration: config,
				}

				resp, err := server.ecsAPI.UpdateCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(resp.Cluster.Configuration).NotTo(BeNil())
				Expect(resp.Cluster.Configuration.ExecuteCommandConfiguration).NotTo(BeNil())
				Expect(*resp.Cluster.Configuration.ExecuteCommandConfiguration.Logging).To(Equal(generated.ExecuteCommandLoggingDefault))
			})

			It("should fail when cluster does not exist", func() {
				clusterName := "non-existent"
				req := &generated.UpdateClusterRequest{
					Cluster: &clusterName,
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
				settingName := generated.ClusterSettingNameContainerInsights
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
				settingName := generated.ClusterSettingNameContainerInsights
				settingValue := "enabled"
				settings := []generated.ClusterSetting{
					{
						Name:  &settingName,
						Value: &settingValue,
					},
				}

				req := &generated.UpdateClusterSettingsRequest{
					Cluster:  &clusterName,
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
					Cluster: &clusterName,
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
				weightOne := generated.CapacityProviderStrategyItemWeight(1)
				baseOne := generated.CapacityProviderStrategyItemBase(1)
				weightFour := generated.CapacityProviderStrategyItemWeight(4)
				strategy := []generated.CapacityProviderStrategyItem{
					{
						CapacityProvider: ptr.String("FARGATE"),
						Weight:           &weightOne,
						Base:             &baseOne,
					},
					{
						CapacityProvider: ptr.String("FARGATE_SPOT"),
						Weight:           &weightFour,
					},
				}

				req := &generated.PutClusterCapacityProvidersRequest{
					Cluster:                         &clusterName,
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
					Cluster: &clusterName,
				}

				_, err := server.ecsAPI.PutClusterCapacityProviders(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("capacityProviders is required"))
			})
		})
	})

	Describe("extractClusterNameFromARN", func() {
		It("should extract cluster name from valid ARN", func() {
			arn := "arn:aws:ecs:ap-northeast-1:123456789012:cluster/my-cluster"
			name := extractClusterNameFromARN(arn)
			Expect(name).To(Equal("my-cluster"))
		})
		
		It("should return input for non-ARN strings", func() {
			name := extractClusterNameFromARN("my-cluster")
			Expect(name).To(Equal("my-cluster"))
		})
		
		It("should return input for invalid ARN format", func() {
			// Missing cluster name after slash
			arn := "arn:aws:ecs:ap-northeast-1:123456789012:cluster/"
			name := extractClusterNameFromARN(arn)
			Expect(name).To(Equal(arn))
			
			// No slash
			arn2 := "arn:aws:ecs:ap-northeast-1:123456789012:cluster"
			name2 := extractClusterNameFromARN(arn2)
			Expect(name2).To(Equal(arn2))
			
			// Multiple slashes
			arn3 := "arn:aws:ecs:ap-northeast-1:123456789012:cluster/my/cluster"
			name3 := extractClusterNameFromARN(arn3)
			Expect(name3).To(Equal(arn3))
		})
		
		It("should handle empty string", func() {
			name := extractClusterNameFromARN("")
			Expect(name).To(Equal(""))
		})
	})
})

// MockTaskDefinitionStore for task definition operations
type MockTaskDefinitionStore struct {
	storage *MockStorage
}

func (m *MockTaskDefinitionStore) Register(ctx context.Context, taskDef *storage.TaskDefinition) (*storage.TaskDefinition, error) {
	if m.storage.taskDefinitions == nil {
		m.storage.taskDefinitions = make(map[string]*storage.TaskDefinition)
	}

	// Get next revision
	maxRevision := 0
	for _, td := range m.storage.taskDefinitions {
		if td.Family == taskDef.Family && td.Revision > maxRevision {
			maxRevision = td.Revision
		}
	}
	taskDef.Revision = maxRevision + 1
	taskDef.ARN = fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task-definition/%s:%d", taskDef.Family, taskDef.Revision)
	taskDef.Status = "ACTIVE"
	taskDef.RegisteredAt = time.Now()

	key := fmt.Sprintf("%s:%d", taskDef.Family, taskDef.Revision)
	m.storage.taskDefinitions[key] = taskDef
	return taskDef, nil
}

func (m *MockTaskDefinitionStore) Get(ctx context.Context, family string, revision int) (*storage.TaskDefinition, error) {
	key := fmt.Sprintf("%s:%d", family, revision)
	td, exists := m.storage.taskDefinitions[key]
	if !exists {
		return nil, errors.New("task definition not found")
	}
	return td, nil
}

func (m *MockTaskDefinitionStore) GetLatest(ctx context.Context, family string) (*storage.TaskDefinition, error) {
	var latest *storage.TaskDefinition
	maxRevision := 0
	for _, td := range m.storage.taskDefinitions {
		if td.Family == family && td.Status == "ACTIVE" && td.Revision > maxRevision {
			maxRevision = td.Revision
			latest = td
		}
	}
	if latest == nil {
		return nil, errors.New("task definition family not found")
	}
	return latest, nil
}

func (m *MockTaskDefinitionStore) ListFamilies(ctx context.Context, familyPrefix string, status string, limit int, nextToken string) ([]*storage.TaskDefinitionFamily, string, error) {
	familyMap := make(map[string]*storage.TaskDefinitionFamily)
	
	for _, td := range m.storage.taskDefinitions {
		if familyPrefix != "" && !hasPrefix(td.Family, familyPrefix) {
			continue
		}
		if status != "" && td.Status != status {
			continue
		}
		
		if family, exists := familyMap[td.Family]; exists {
			if td.Revision > family.LatestRevision {
				family.LatestRevision = td.Revision
			}
			if td.Status == "ACTIVE" {
				family.ActiveRevisions++
			}
		} else {
			familyMap[td.Family] = &storage.TaskDefinitionFamily{
				Family:          td.Family,
				LatestRevision:  td.Revision,
				ActiveRevisions: 0,
			}
			if td.Status == "ACTIVE" {
				familyMap[td.Family].ActiveRevisions = 1
			}
		}
	}
	
	families := make([]*storage.TaskDefinitionFamily, 0, len(familyMap))
	for _, family := range familyMap {
		families = append(families, family)
	}
	
	// Simple pagination
	if limit > 0 && len(families) > limit {
		return families[:limit], families[limit-1].Family, nil
	}
	
	return families, "", nil
}

func (m *MockTaskDefinitionStore) ListRevisions(ctx context.Context, family string, status string, limit int, nextToken string) ([]*storage.TaskDefinitionRevision, string, error) {
	revisions := make([]*storage.TaskDefinitionRevision, 0)
	
	for _, td := range m.storage.taskDefinitions {
		if td.Family != family {
			continue
		}
		if status != "" && td.Status != status {
			continue
		}
		
		rev := &storage.TaskDefinitionRevision{
			ARN:      td.ARN,
			Family:   td.Family,
			Revision: td.Revision,
			Status:   td.Status,
		}
		revisions = append(revisions, rev)
	}
	
	return revisions, "", nil
}

func (m *MockTaskDefinitionStore) Deregister(ctx context.Context, family string, revision int) error {
	key := fmt.Sprintf("%s:%d", family, revision)
	td, exists := m.storage.taskDefinitions[key]
	if !exists {
		return errors.New("task definition not found")
	}
	if td.Status == "INACTIVE" {
		return nil // Already inactive (idempotent)
	}
	td.Status = "INACTIVE"
	return nil
}

func (m *MockTaskDefinitionStore) GetByARN(ctx context.Context, arn string) (*storage.TaskDefinition, error) {
	for _, td := range m.storage.taskDefinitions {
		if td.ARN == arn {
			return td, nil
		}
	}
	return nil, errors.New("task definition not found")
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}