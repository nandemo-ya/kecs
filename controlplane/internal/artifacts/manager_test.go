package artifacts_test

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/artifacts"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/s3"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

// Mock S3 integration
type mockS3Integration struct {
	downloadFunc func(ctx context.Context, bucket, key string) (io.ReadCloser, error)
}

func (m *mockS3Integration) DownloadFile(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, bucket, key)
	}
	return nil, errors.New("not implemented")
}

func (m *mockS3Integration) UploadFile(ctx context.Context, bucket, key string, reader io.Reader) error {
	return errors.New("not implemented")
}

func (m *mockS3Integration) HeadObject(ctx context.Context, bucket, key string) (*s3.ObjectMetadata, error) {
	return nil, errors.New("not implemented")
}

func (m *mockS3Integration) CreateBucket(ctx context.Context, bucket string) error {
	return errors.New("not implemented")
}

func (m *mockS3Integration) DeleteObject(ctx context.Context, bucket, key string) error {
	return errors.New("not implemented")
}

var _ = Describe("Artifact Manager", func() {
	var (
		manager    *artifacts.Manager
		mockS3     *mockS3Integration
		tempDir    string
		testServer *httptest.Server
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "artifact-test-*")
		Expect(err).NotTo(HaveOccurred())

		mockS3 = &mockS3Integration{}
		manager = artifacts.NewManager(mockS3)

		// Set up test HTTP server
		testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/test-file.txt":
				w.Write([]byte("test content"))
			case "/test-with-checksum.txt":
				w.Write([]byte("checksummed content"))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		testServer.Close()
	})

	Describe("DownloadArtifact", func() {
		Context("from S3", func() {
			It("should download file from S3", func() {
				content := "s3 test content"
				mockS3.downloadFunc = func(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
					Expect(bucket).To(Equal("test-bucket"))
					Expect(key).To(Equal("test-key"))
					return io.NopCloser(strings.NewReader(content)), nil
				}

				artifact := &types.Artifact{
					ArtifactUrl: ptr.To("s3://test-bucket/test-key"),
					TargetPath:  ptr.To(filepath.Join(tempDir, "s3-file.txt")),
				}

				err := manager.DownloadArtifact(context.Background(), artifact)
				Expect(err).NotTo(HaveOccurred())

				// Verify file content
				data, err := os.ReadFile(*artifact.TargetPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal(content))
			})

			It("should handle S3 download errors", func() {
				mockS3.downloadFunc = func(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
					return nil, errors.New("s3 error")
				}

				artifact := &types.Artifact{
					ArtifactUrl: ptr.To("s3://test-bucket/test-key"),
					TargetPath:  ptr.To(filepath.Join(tempDir, "s3-file.txt")),
				}

				err := manager.DownloadArtifact(context.Background(), artifact)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("s3 error"))
			})
		})

		Context("from HTTP", func() {
			It("should download file from HTTP", func() {
				artifact := &types.Artifact{
					ArtifactUrl: ptr.To(testServer.URL + "/test-file.txt"),
					TargetPath:  ptr.To(filepath.Join(tempDir, "http-file.txt")),
				}

				err := manager.DownloadArtifact(context.Background(), artifact)
				Expect(err).NotTo(HaveOccurred())

				// Verify file content
				data, err := os.ReadFile(*artifact.TargetPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal("test content"))
			})

			It("should handle HTTP errors", func() {
				artifact := &types.Artifact{
					ArtifactUrl: ptr.To(testServer.URL + "/not-found"),
					TargetPath:  ptr.To(filepath.Join(tempDir, "http-file.txt")),
				}

				err := manager.DownloadArtifact(context.Background(), artifact)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("404"))
			})
		})

		Context("with checksum validation", func() {
			It("should validate SHA256 checksum", func() {
				content := "checksummed content"
				h := sha256.Sum256([]byte(content))
				checksum := hex.EncodeToString(h[:])

				artifact := &types.Artifact{
					ArtifactUrl:  ptr.To(testServer.URL + "/test-with-checksum.txt"),
					TargetPath:   ptr.To(filepath.Join(tempDir, "checksum-file.txt")),
					Checksum:     ptr.To(checksum),
					ChecksumType: ptr.To("sha256"),
				}

				err := manager.DownloadArtifact(context.Background(), artifact)
				Expect(err).NotTo(HaveOccurred())

				// Verify file exists
				_, err = os.Stat(*artifact.TargetPath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail on checksum mismatch", func() {
				artifact := &types.Artifact{
					ArtifactUrl:  ptr.To(testServer.URL + "/test-with-checksum.txt"),
					TargetPath:   ptr.To(filepath.Join(tempDir, "checksum-file.txt")),
					Checksum:     ptr.To("invalid-checksum"),
					ChecksumType: ptr.To("sha256"),
				}

				err := manager.DownloadArtifact(context.Background(), artifact)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("checksum mismatch"))
			})

			It("should validate MD5 checksum", func() {
				content := "checksummed content"
				h := md5.Sum([]byte(content))
				checksum := hex.EncodeToString(h[:])

				artifact := &types.Artifact{
					ArtifactUrl:  ptr.To(testServer.URL + "/test-with-checksum.txt"),
					TargetPath:   ptr.To(filepath.Join(tempDir, "checksum-file.txt")),
					Checksum:     ptr.To(checksum),
					ChecksumType: ptr.To("md5"),
				}

				err := manager.DownloadArtifact(context.Background(), artifact)
				Expect(err).NotTo(HaveOccurred())

				// Verify file exists
				_, err = os.Stat(*artifact.TargetPath)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with file permissions", func() {
			It("should set file permissions", func() {
				artifact := &types.Artifact{
					ArtifactUrl: ptr.To(testServer.URL + "/test-file.txt"),
					TargetPath:  ptr.To(filepath.Join(tempDir, "perm-file.txt")),
					Permissions: ptr.To("0600"),
				}

				err := manager.DownloadArtifact(context.Background(), artifact)
				Expect(err).NotTo(HaveOccurred())

				// Verify permissions
				info, err := os.Stat(*artifact.TargetPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(info.Mode().Perm()).To(Equal(os.FileMode(0600)))
			})
		})

		Context("edge cases", func() {
			It("should create target directory if it doesn't exist", func() {
				artifact := &types.Artifact{
					ArtifactUrl: ptr.To(testServer.URL + "/test-file.txt"),
					TargetPath:  ptr.To(filepath.Join(tempDir, "subdir", "nested", "file.txt")),
				}

				err := manager.DownloadArtifact(context.Background(), artifact)
				Expect(err).NotTo(HaveOccurred())

				// Verify file exists
				_, err = os.Stat(*artifact.TargetPath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should require artifact URL and target path", func() {
				artifact := &types.Artifact{}
				err := manager.DownloadArtifact(context.Background(), artifact)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("required"))

				artifact.ArtifactUrl = ptr.To("http://example.com")
				err = manager.DownloadArtifact(context.Background(), artifact)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("required"))
			})
		})
	})

	Describe("GetArtifactScript", func() {
		It("should generate shell script for artifacts", func() {
			artifacts := []types.Artifact{
				{
					ArtifactUrl: ptr.To("https://example.com/file.txt"),
					TargetPath:  ptr.To("config/file.txt"),
					Permissions: ptr.To("0644"),
				},
				{
					ArtifactUrl: ptr.To("s3://bucket/key"),
					TargetPath:  ptr.To("data/key"),
				},
			}

			script := manager.GetArtifactScript(artifacts)
			Expect(script).To(ContainSubstring("#!/bin/sh"))
			Expect(script).To(ContainSubstring("wget -O /artifacts/config/file.txt"))
			Expect(script).To(ContainSubstring("chmod 0644"))
			Expect(script).To(ContainSubstring("S3 download placeholder"))
		})

		It("should return empty string for no artifacts", func() {
			script := manager.GetArtifactScript([]types.Artifact{})
			Expect(script).To(Equal(""))
		})
	})
})
