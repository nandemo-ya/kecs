//go:build integration
// +build integration

package cloudwatch_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/fake"

	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/cloudwatch"
)

var _ = Describe("CloudWatch LocalStack Integration", func() {
	var (
		integration cloudwatch.Integration
		kubeClient  *fake.Clientset
	)

	BeforeEach(func() {
		if testing.Short() {
			Skip("Skipping LocalStack integration test in short mode")
		}

		kubeClient = fake.NewSimpleClientset()

		// Create real LocalStack manager
		lsManager := &mockLocalStackManager{}

		config := &cloudwatch.Config{
			LocalStackEndpoint: "http://localhost:4566",
			LogGroupPrefix:     "/ecs/test/",
			RetentionDays:      1,
			KubeNamespace:      "default",
		}

		var err error
		integration, err = cloudwatch.NewIntegration(kubeClient, lsManager, config)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Real LocalStack Operations", func() {
		It("should create and delete log groups in LocalStack", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			groupName := "integration-test-" + time.Now().Format("20060102150405")

			// Create log group
			err := integration.CreateLogGroup(groupName)
			Expect(err).NotTo(HaveOccurred())

			// Create a log stream
			streamName := "test-stream"
			err = integration.CreateLogStream(groupName, streamName)
			Expect(err).NotTo(HaveOccurred())

			// Delete log group
			err = integration.DeleteLogGroup(groupName)
			Expect(err).NotTo(HaveOccurred())

			// Verify deletion - creating stream should fail
			err = integration.CreateLogStream(groupName, "another-stream")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ResourceNotFoundException"))

			_ = ctx // use context if needed
		})

		It("should handle log group already exists error", func() {
			groupName := "duplicate-test-" + time.Now().Format("20060102150405")

			// Create log group
			err := integration.CreateLogGroup(groupName)
			Expect(err).NotTo(HaveOccurred())

			// Try to create again
			err = integration.CreateLogGroup(groupName)
			Expect(err).NotTo(HaveOccurred()) // Should handle gracefully

			// Clean up
			err = integration.DeleteLogGroup(groupName)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
