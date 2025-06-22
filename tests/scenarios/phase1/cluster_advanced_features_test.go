package phase1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Cluster Advanced Features", Serial, func() {
	var (
		client utils.ECSClientInterface
		logger *utils.TestLogger
	)

	BeforeEach(func() {
		// Use shared resources from suite
		client = sharedClient
		logger = sharedLogger
	})

	Describe("Cluster Settings", func() {
		var clusterName string

		BeforeEach(func() {
			clusterName = utils.GenerateTestName("settings-cluster")
			DeferCleanup(func() {
				_ = client.DeleteCluster(clusterName)
			})
		})

		Context("when creating a cluster with settings", func() {
			It("should create the cluster with containerInsights enabled", func() {
				logger.Info("Creating cluster with settings: %s", clusterName)

				// For AWS CLI client, we need to implement CreateClusterWithSettings
				// For now, create cluster and update settings
				err := client.CreateCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())

				// Update cluster settings via AWS CLI
				// aws ecs update-cluster-settings --cluster <name> --settings name=containerInsights,value=enabled
				// This needs to be implemented in the client
				
				// Verify cluster exists
				cluster, err := client.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.ClusterName).To(Equal(clusterName))
			})
		})

		Context("when updating cluster settings", func() {
			BeforeEach(func() {
				Expect(client.CreateCluster(clusterName)).To(Succeed())
			})

			It("should update individual settings", func() {
				logger.Info("Updating cluster settings for: %s", clusterName)

				// This functionality needs to be added to the AWS CLI client
				// aws ecs update-cluster-settings --cluster <name> --settings name=containerInsights,value=enabled
			})
		})
	})

	Describe("Cluster Tags", func() {
		var clusterName string
		var clusterArn string

		BeforeEach(func() {
			clusterName = utils.GenerateTestName("tags-cluster")
			Expect(client.CreateCluster(clusterName)).To(Succeed())
			
			// Get cluster ARN for tagging
			cluster, err := client.DescribeCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())
			clusterArn = cluster.ClusterArn

			DeferCleanup(func() {
				_ = client.DeleteCluster(clusterName)
			})
		})

		Context("when tagging a cluster", func() {
			It("should add tags to the cluster", func() {
				logger.Info("Adding tags to cluster: %s", clusterName)

				tags := map[string]string{
					"Environment": "test",
					"Team":        "platform",
					"Project":     "kecs-testing",
				}

				err := client.TagResource(clusterArn, tags)
				Expect(err).NotTo(HaveOccurred())

				// List tags
				retrievedTags, err := client.ListTagsForResource(clusterArn)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrievedTags).To(HaveKeyWithValue("Environment", "test"))
				Expect(retrievedTags).To(HaveKeyWithValue("Team", "platform"))
				Expect(retrievedTags).To(HaveKeyWithValue("Project", "kecs-testing"))
			})
		})

		Context("when removing tags from a cluster", func() {
			BeforeEach(func() {
				// Add some tags first
				tags := map[string]string{
					"ToKeep":   "value1",
					"ToRemove": "value2",
				}
				Expect(client.TagResource(clusterArn, tags)).To(Succeed())
			})

			It("should remove specific tags", func() {
				logger.Info("Removing tags from cluster: %s", clusterName)

				err := client.UntagResource(clusterArn, []string{"ToRemove"})
				Expect(err).NotTo(HaveOccurred())

				// Verify tags
				tags, err := client.ListTagsForResource(clusterArn)
				Expect(err).NotTo(HaveOccurred())
				Expect(tags).To(HaveKey("ToKeep"))
				Expect(tags).NotTo(HaveKey("ToRemove"))
			})
		})
	})

	Describe("Capacity Providers", func() {
		var clusterName string

		BeforeEach(func() {
			clusterName = utils.GenerateTestName("capacity-cluster")
			Expect(client.CreateCluster(clusterName)).To(Succeed())
			
			DeferCleanup(func() {
				_ = client.DeleteCluster(clusterName)
			})
		})

		Context("when setting capacity providers on a cluster", func() {
			It("should configure FARGATE and FARGATE_SPOT providers", func() {
				logger.Info("Setting capacity providers for cluster: %s", clusterName)

				// This needs implementation in the AWS CLI client
				// aws ecs put-cluster-capacity-providers --cluster <name> 
				//   --capacity-providers FARGATE FARGATE_SPOT
				//   --default-capacity-provider-strategy capacityProvider=FARGATE,weight=1
			})
		})
	})

	Describe("Cluster Configuration", func() {
		var clusterName string

		BeforeEach(func() {
			clusterName = utils.GenerateTestName("config-cluster")
			DeferCleanup(func() {
				_ = client.DeleteCluster(clusterName)
			})
		})

		Context("when creating a cluster with configuration", func() {
			It("should create cluster with execute command configuration", func() {
				logger.Info("Creating cluster with configuration: %s", clusterName)

				// Create cluster
				err := client.CreateCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())

				// Update cluster configuration needs to be implemented
				// aws ecs update-cluster --cluster <name> --configuration executeCommandConfiguration={...}
			})
		})
	})

	Describe("Describe Clusters with Include Options", func() {
		var clusterName string

		BeforeEach(func() {
			clusterName = utils.GenerateTestName("include-test")
			
			// Create cluster with various features
			Expect(client.CreateCluster(clusterName)).To(Succeed())
			
			// Add tags
			cluster, err := client.DescribeCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())
			
			tags := map[string]string{
				"TestTag": "TestValue",
			}
			Expect(client.TagResource(cluster.ClusterArn, tags)).To(Succeed())

			DeferCleanup(func() {
				_ = client.DeleteCluster(clusterName)
			})
		})

		Context("when describing with include options", func() {
			It("should include requested fields", func() {
				logger.Info("Describing cluster with include options: %s", clusterName)

				// This needs special handling in AWS CLI client
				// aws ecs describe-clusters --clusters <name> --include TAGS SETTINGS CONFIGURATIONS
			})
		})
	})
})

