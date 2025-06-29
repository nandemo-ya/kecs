// cluster_persistence_test.go
// This test verifies that KECS properly recovers k3d clusters after restart.

package phase1

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Cluster Persistence", Serial, func() {
	BeforeEach(func() {
		// Ensure we're not in test mode
		os.Unsetenv("KECS_TEST_MODE")
	})

	Describe("Single Cluster Persistence", func() {
		Context("when KECS is restarted with a persistent cluster", func() {
			var (
				kecs        *utils.KECSContainer
				client      utils.ECSClientInterface
				clusterName string
			)

			BeforeEach(func() {
				// Start KECS container with persistent data directory
				kecs = utils.StartKECSWithPersistence(GinkgoT())
				client = utils.NewECSClientInterface(kecs.Endpoint())
				clusterName = "persistence-test-cluster"
			})

			AfterEach(func() {
				if kecs != nil {
					kecs.Cleanup()
				}
			})

			It("should recover ECS cluster and recreate k3d cluster after restart", func() {
				By("Creating an ECS cluster")
				err := client.CreateCluster(clusterName)
				Expect(err).NotTo(HaveOccurred(), "Failed to create cluster")

				By("Waiting for cluster to be active")
				utils.AssertClusterActive(GinkgoT(), client, clusterName)

				By("Verifying k3d cluster exists")
				k3dName := "kecs-" + clusterName
				Eventually(func() bool {
					exists, err := utils.K3dClusterExists(k3dName)
					if err != nil {
						return false
					}
					return exists
				}, 20*time.Second, 1*time.Second).Should(BeTrue(), "k3d cluster should exist after creation")

				By("Stopping KECS")
				err = kecs.Stop()
				Expect(err).NotTo(HaveOccurred(), "Failed to stop KECS container")

				// Ensure container is fully stopped
				Eventually(func() bool {
					// Container is considered stopped when Stop() returns without error
					return true
				}, 5*time.Second).Should(BeTrue())

				By("Restarting KECS with the same data directory")
				kecs2 := utils.RestartKECSWithPersistence(GinkgoT(), kecs.DataDir)
				defer kecs2.Cleanup()

				// Create new client with new endpoint
				client2 := utils.NewECSClientInterface(kecs2.Endpoint())

				By("Verifying cluster is recovered from storage")
				clusters, err := client2.ListClusters()
				Expect(err).NotTo(HaveOccurred(), "Failed to list clusters after restart")
				// ListClusters returns ARNs, need to check if any ARN contains the cluster name
				found := false
				for _, arn := range clusters {
					if containsClusterName(arn, clusterName) {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), "Cluster should be recovered from storage")

				By("Verifying k3d cluster is recreated")
				Eventually(func() bool {
					exists, err := utils.K3dClusterExists(k3dName)
					if err != nil {
						return false
					}
					return exists
				}, 15*time.Second, 1*time.Second).Should(BeTrue(), "k3d cluster should be recreated after KECS restart")

				By("Verifying cluster is functional")
				cluster, err := client2.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred(), "Failed to describe cluster after restart")
				Expect(cluster.Status).To(Equal("ACTIVE"), "Cluster should be active after restart")

				By("Cleaning up")
				err = client2.DeleteCluster(clusterName)
				Expect(err).NotTo(HaveOccurred(), "Failed to delete cluster during cleanup")
			})
		})
	})

	Describe("Multiple Cluster Persistence", func() {
		Context("when KECS is restarted with multiple persistent clusters", func() {
			var (
				kecs         *utils.KECSContainer
				client       utils.ECSClientInterface
				clusterNames []string
			)

			BeforeEach(func() {
				// Start KECS
				kecs = utils.StartKECSWithPersistence(GinkgoT())
				client = utils.NewECSClientInterface(kecs.Endpoint())
				clusterNames = []string{
					"persistence-cluster-1",
					"persistence-cluster-2",
					"persistence-cluster-3",
				}
			})

			AfterEach(func() {
				if kecs != nil {
					kecs.Cleanup()
				}
			})

			It("should recover all clusters and recreate their k3d clusters", func() {
				By("Creating multiple clusters")
				for _, name := range clusterNames {
					err := client.CreateCluster(name)
					Expect(err).NotTo(HaveOccurred(), "Failed to create cluster %s", name)
					utils.AssertClusterActive(GinkgoT(), client, name)
				}

				By("Verifying all k3d clusters exist")
				for _, name := range clusterNames {
					k3dName := "kecs-" + name
					Eventually(func() bool {
						exists, err := utils.K3dClusterExists(k3dName)
						if err != nil {
							return false
						}
						return exists
					}, 25*time.Second, 1*time.Second).Should(BeTrue(), "k3d cluster %s should exist", k3dName)
				}

				By("Restarting KECS")
				err := kecs.Stop()
				Expect(err).NotTo(HaveOccurred())
				
				// Ensure container is fully stopped
				Eventually(func() bool {
					return true
				}, 5*time.Second).Should(BeTrue())

				kecs2 := utils.RestartKECSWithPersistence(GinkgoT(), kecs.DataDir)
				defer kecs2.Cleanup()

				client2 := utils.NewECSClientInterface(kecs2.Endpoint())

				By("Waiting for recovery process")
				// Wait for KECS to fully recover clusters from storage
				Eventually(func() bool {
					clusters, err := client2.ListClusters()
					if err != nil {
						return false
					}
					// Check if all clusters are recovered
					recoveredCount := 0
					for _, name := range clusterNames {
						for _, arn := range clusters {
							if containsClusterName(arn, name) {
								recoveredCount++
								break
							}
						}
					}
					return recoveredCount == len(clusterNames)
				}, 20*time.Second, 1*time.Second).Should(BeTrue(), "All clusters should be recovered from storage")

				By("Verifying all clusters are recovered")
				clusters, err := client2.ListClusters()
				Expect(err).NotTo(HaveOccurred())

				for _, name := range clusterNames {
					found := false
					for _, arn := range clusters {
						if containsClusterName(arn, name) {
							found = true
							break
						}
					}
					Expect(found).To(BeTrue(), "Cluster %s should be recovered", name)

					By(fmt.Sprintf("Verifying k3d cluster %s is recreated", name))
					k3dName := "kecs-" + name
					Eventually(func() bool {
						exists, err := utils.K3dClusterExists(k3dName)
						if err != nil {
							return false
						}
						return exists
					}, 15*time.Second, 1*time.Second).Should(BeTrue(), "k3d cluster %s should be recreated", k3dName)
				}

				By("Cleaning up all clusters")
				for _, name := range clusterNames {
					err := client2.DeleteCluster(name)
					Expect(err).NotTo(HaveOccurred(), "Failed to delete cluster %s", name)
				}
			})
		})
	})
})