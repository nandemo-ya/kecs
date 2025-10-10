package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
)

var _ = Describe("Start Command Unit Tests", func() {
	Describe("determineInstanceToStart", func() {
		Context("when no instance name is provided", func() {
			BeforeEach(func() {
				startInstanceName = ""
			})

			It("should call selectOrCreateInstance", func() {
				// This test would require mocking the K3dClusterManager
				// which would be implemented with a proper testing interface
				Skip("Requires mock implementation")
			})
		})

		Context("when instance name is provided", func() {
			BeforeEach(func() {
				startInstanceName = "test-instance"
			})

			It("should check if the instance exists", func() {
				// This test would require mocking the K3dClusterManager
				Skip("Requires mock implementation")
			})
		})
	})

	Describe("showStartCompletionMessage", func() {
		It("should display completion message with correct ports", func() {
			// Capture stdout for testing
			// This would be better tested with a writer interface
			Skip("Requires output capture implementation")
		})
	})

	Describe("getInstanceStatus", func() {
		It("should return 'running' for running instances", func() {
			// This test would require mocking the K3dClusterManager
			Skip("Requires mock implementation")
		})

		It("should return 'stopped' for stopped instances", func() {
			// This test would require mocking the K3dClusterManager
			Skip("Requires mock implementation")
		})
	})

	Describe("getInstanceDataInfo", func() {
		Context("when data directory exists", func() {
			It("should return ', has data'", func() {
				// This would require setting up a temporary directory
				Skip("Requires filesystem setup")
			})
		})

		Context("when data directory does not exist", func() {
			It("should return empty string", func() {
				result := getInstanceDataInfo("non-existent-instance")
				Expect(result).To(Equal(""))
			})
		})
	})

	Describe("createNewInstance", func() {
		It("should generate a random instance name", func() {
			name, isNew, err := createNewInstance()
			Expect(err).NotTo(HaveOccurred())
			Expect(name).NotTo(BeEmpty())
			Expect(isNew).To(BeTrue())
		})

		It("should return different names on subsequent calls", func() {
			name1, _, err1 := createNewInstance()
			Expect(err1).NotTo(HaveOccurred())

			name2, _, err2 := createNewInstance()
			Expect(err2).NotTo(HaveOccurred())

			Expect(name1).NotTo(Equal(name2))
		})
	})
})

var _ = Describe("Start V2 Command Integration Tests", func() {
	var (
		clusterName string
		ctx         context.Context
		cancel      context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
		clusterName = fmt.Sprintf("kecs-test-%d", time.Now().Unix())
		DeferCleanup(func() {
			cancel()
		})
	})

	AfterEach(func() {
		cancel()
		// Clean up the test cluster
		cmd := exec.Command("kecs", "stop-v2", "--name", clusterName, "--delete-data")
		cmd.Run()
	})

	Describe("New Architecture Deployment", func() {
		It("should successfully deploy all components", func() {
			if !config.GetBool("features.integrationTest") {
				Skip("Skipping integration test (set KECS_INTEGRATION_TEST=true to run)")
			}

			By("Starting KECS with new architecture")
			cmd := exec.CommandContext(ctx, "kecs", "start-v2", "--name", clusterName, "--timeout", "10m")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Failed to start KECS: %s", string(output))

			By("Checking control plane health")
			Eventually(func() error {
				resp, err := http.Get("http://localhost:8081/health")
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("health check returned status %d", resp.StatusCode)
				}
				return nil
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			By("Testing ECS API through Traefik")
			cmd = exec.Command("aws", "ecs", "list-clusters",
				"--endpoint-url", "http://localhost:4566",
				"--region", "us-east-1",
				"--no-cli-pager")
			output, err = cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Failed to list clusters: %s", string(output))

			By("Testing LocalStack through Traefik")
			cmd = exec.Command("aws", "s3", "ls",
				"--endpoint-url", "http://localhost:4566",
				"--region", "us-east-1",
				"--no-cli-pager")
			output, err = cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Failed to list S3 buckets: %s", string(output))
		})
	})

	Describe("Component Health Checks", func() {
		It("should report all components as healthy", func() {
			if !config.GetBool("features.integrationTest") {
				Skip("Skipping integration test (set KECS_INTEGRATION_TEST=true to run)")
			}

			// Start the cluster first
			cmd := exec.Command("kecs", "start-v2", "--name", clusterName)
			err := cmd.Run()
			Expect(err).NotTo(HaveOccurred())

			By("Checking detailed health status")
			resp, err := http.Get("http://localhost:8081/health/detailed")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})
})
