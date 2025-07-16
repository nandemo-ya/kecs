package cmd_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
			if os.Getenv("KECS_INTEGRATION_TEST") != "true" {
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
			if os.Getenv("KECS_INTEGRATION_TEST") != "true" {
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

func TestStartV2Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Start V2 Integration Test Suite")
}