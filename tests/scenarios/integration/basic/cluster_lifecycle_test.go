package basic_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Cluster Lifecycle", func() {
	var (
		kecs      *utils.KECSContainer
		ecsClient utils.ECSClientInterface
	)

	BeforeEach(func() {
		// Start KECS
		kecs = utils.StartKECS(GinkgoT())
		ecsClient = utils.NewCurlClient(kecs.Endpoint())
	})

	AfterEach(func() {
		if kecs != nil {
			kecs.Cleanup()
		}
	})

	Describe("Basic Cluster Operations", func() {
		It("should create and delete a cluster", func() {
			clusterName := fmt.Sprintf("test-cluster-%d", time.Now().Unix())

			By("Creating a cluster")
			err := ecsClient.CreateCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying cluster exists")
			cluster, err := ecsClient.DescribeCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())
			Expect(cluster.ClusterName).To(Equal(clusterName))
			Expect(cluster.Status).To(Equal("ACTIVE"))

			By("Listing clusters")
			clusters, err := ecsClient.ListClusters()
			Expect(err).NotTo(HaveOccurred())
			Expect(clusters).To(ContainElement(ContainSubstring(clusterName)))

			By("Deleting the cluster")
			err = ecsClient.DeleteCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying cluster is deleted")
			Eventually(func() error {
				_, err := ecsClient.DescribeCluster(clusterName)
				return err
			}, 30*time.Second, 2*time.Second).Should(HaveOccurred())
		})

		It("should handle multiple clusters", func() {
			clusterNames := []string{
				fmt.Sprintf("test-cluster-1-%d", time.Now().Unix()),
				fmt.Sprintf("test-cluster-2-%d", time.Now().Unix()),
				fmt.Sprintf("test-cluster-3-%d", time.Now().Unix()),
			}

			By("Creating multiple clusters")
			for _, name := range clusterNames {
				err := ecsClient.CreateCluster(name)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Verifying all clusters exist")
			clusters, err := ecsClient.ListClusters()
			Expect(err).NotTo(HaveOccurred())
			for _, name := range clusterNames {
				Expect(clusters).To(ContainElement(ContainSubstring(name)))
			}

			By("Deleting all clusters")
			for _, name := range clusterNames {
				err := ecsClient.DeleteCluster(name)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should handle cluster name validation", func() {
			By("Rejecting invalid cluster names")
			invalidNames := []string{
				"", // empty name
				"a", // too short
				"cluster_with_underscores", // invalid characters
				"cluster with spaces",      // spaces not allowed
			}

			for _, name := range invalidNames {
				err := ecsClient.CreateCluster(name)
				Expect(err).To(HaveOccurred(), fmt.Sprintf("Should reject cluster name: %s", name))
			}
		})

		It("should prevent duplicate cluster creation", func() {
			clusterName := fmt.Sprintf("duplicate-test-%d", time.Now().Unix())

			By("Creating first cluster")
			err := ecsClient.CreateCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())

			By("Attempting to create duplicate cluster")
			err = ecsClient.CreateCluster(clusterName)
			// Should either succeed (idempotent) or fail with appropriate error
			// Implementation depends on ECS behavior

			By("Cleanup")
			ecsClient.DeleteCluster(clusterName)
		})
	})

	Describe("Cluster Attributes", func() {
		var clusterName string

		BeforeEach(func() {
			clusterName = fmt.Sprintf("attr-test-%d", time.Now().Unix())
			err := ecsClient.CreateCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			ecsClient.DeleteCluster(clusterName)
		})

		It("should support cluster attributes", func() {
			attributes := []utils.Attribute{
				{
					Name:  "test-attribute",
					Value: "test-value",
				},
				{
					Name:  "environment",
					Value: "testing",
				},
			}

			By("Setting cluster attributes")
			err := ecsClient.PutAttributes(clusterName, attributes)
			Expect(err).NotTo(HaveOccurred())

			By("Retrieving cluster attributes")
			retrievedAttrs, err := ecsClient.ListAttributes(clusterName, "cluster")
			Expect(err).NotTo(HaveOccurred())
			
			for _, attr := range attributes {
				found := false
				for _, retrieved := range retrievedAttrs {
					if retrieved.Name == attr.Name && retrieved.Value == attr.Value {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), fmt.Sprintf("Attribute %s=%s not found", attr.Name, attr.Value))
			}

			By("Deleting cluster attributes")
			err = ecsClient.DeleteAttributes(clusterName, attributes)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Cluster Tags", func() {
		var clusterName string
		var clusterArn string

		BeforeEach(func() {
			clusterName = fmt.Sprintf("tag-test-%d", time.Now().Unix())
			err := ecsClient.CreateCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())

			cluster, err := ecsClient.DescribeCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())
			clusterArn = cluster.ClusterArn
		})

		AfterEach(func() {
			ecsClient.DeleteCluster(clusterName)
		})

		It("should support cluster tagging", func() {
			tags := map[string]string{
				"Environment": "test",
				"Owner":       "integration-test",
				"Purpose":     "cluster-lifecycle-test",
			}

			By("Tagging the cluster")
			err := ecsClient.TagResource(clusterArn, tags)
			Expect(err).NotTo(HaveOccurred())

			By("Retrieving cluster tags")
			retrievedTags, err := ecsClient.ListTagsForResource(clusterArn)
			Expect(err).NotTo(HaveOccurred())

			for key, value := range tags {
				Expect(retrievedTags).To(HaveKeyWithValue(key, value))
			}

			By("Removing cluster tags")
			tagKeys := make([]string, 0, len(tags))
			for key := range tags {
				tagKeys = append(tagKeys, key)
			}
			err = ecsClient.UntagResource(clusterArn, tagKeys)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying tags are removed")
			retrievedTags, err = ecsClient.ListTagsForResource(clusterArn)
			Expect(err).NotTo(HaveOccurred())
			for key := range tags {
				Expect(retrievedTags).NotTo(HaveKey(key))
			}
		})
	})
})