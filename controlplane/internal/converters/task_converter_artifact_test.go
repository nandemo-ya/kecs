package converters_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/artifacts"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

var _ = Describe("TaskConverter Artifact Support", func() {
	var converter *converters.TaskConverter

	BeforeEach(func() {
		converter = converters.NewTaskConverter("us-east-1", "123456789012")
	})

	Describe("SetArtifactManager", func() {
		It("should set the artifact manager", func() {
			// Create a mock artifact manager
			artifactManager := artifacts.NewManager(nil)

			// This should not panic
			converter.SetArtifactManager(artifactManager)
		})
	})

	Describe("GetArtifactScript", func() {
		It("should generate shell script for artifacts", func() {
			artifactManager := artifacts.NewManager(nil)

			artifacts := []types.Artifact{
				{
					ArtifactUrl: stringPtr("https://example.com/file.txt"),
					TargetPath:  stringPtr("config/file.txt"),
					Permissions: stringPtr("0644"),
				},
				{
					ArtifactUrl: stringPtr("s3://bucket/key"),
					TargetPath:  stringPtr("data/key"),
				},
			}

			script := artifactManager.GetArtifactScript(artifacts)
			Expect(script).To(ContainSubstring("#!/bin/sh"))
			Expect(script).To(ContainSubstring("wget -O /artifacts/config/file.txt"))
			Expect(script).To(ContainSubstring("chmod 0644"))
			Expect(script).To(ContainSubstring("S3 download placeholder"))
		})
	})
})

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
