package s3_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	s3api "github.com/nandemo-ya/kecs/controlplane/internal/s3/generated"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/fake"

	kecsS3 "github.com/nandemo-ya/kecs/controlplane/internal/integrations/s3"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// Mock S3 client
type mockS3Client struct {
	getObjectFunc    func(ctx context.Context, params *s3api.GetObjectRequest) (*s3api.GetObjectOutput, error)
	putObjectFunc    func(ctx context.Context, params *s3api.PutObjectRequest) (*s3api.PutObjectOutput, error)
	headObjectFunc   func(ctx context.Context, params *s3api.HeadObjectRequest) (*s3api.HeadObjectOutput, error)
	createBucketFunc func(ctx context.Context, params *s3api.CreateBucketRequest) (*s3api.CreateBucketOutput, error)
	deleteObjectFunc func(ctx context.Context, params *s3api.DeleteObjectRequest) (*s3api.DeleteObjectOutput, error)
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3api.GetObjectRequest) (*s3api.GetObjectOutput, error) {
	if m.getObjectFunc != nil {
		return m.getObjectFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3api.PutObjectRequest) (*s3api.PutObjectOutput, error) {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3api.HeadObjectRequest) (*s3api.HeadObjectOutput, error) {
	if m.headObjectFunc != nil {
		return m.headObjectFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockS3Client) CreateBucket(ctx context.Context, params *s3api.CreateBucketRequest) (*s3api.CreateBucketOutput, error) {
	if m.createBucketFunc != nil {
		return m.createBucketFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockS3Client) DeleteObject(ctx context.Context, params *s3api.DeleteObjectRequest) (*s3api.DeleteObjectOutput, error) {
	if m.deleteObjectFunc != nil {
		return m.deleteObjectFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

// Mock LocalStack manager
type mockLocalStackManager struct{}

func (m *mockLocalStackManager) Start(ctx context.Context) error {
	return nil
}

func (m *mockLocalStackManager) Stop(ctx context.Context) error {
	return nil
}

func (m *mockLocalStackManager) Restart(ctx context.Context) error {
	return nil
}

func (m *mockLocalStackManager) GetStatus() (*localstack.Status, error) {
	return &localstack.Status{Running: true, Healthy: true}, nil
}

func (m *mockLocalStackManager) UpdateServices(services []string) error {
	return nil
}

func (m *mockLocalStackManager) GetEnabledServices() ([]string, error) {
	return []string{"s3", "iam", "logs"}, nil
}

func (m *mockLocalStackManager) GetEndpoint() (string, error) {
	return "http://localstack:4566", nil
}

func (m *mockLocalStackManager) GetServiceEndpoint(service string) (string, error) {
	return "http://localstack:4566", nil
}

func (m *mockLocalStackManager) IsHealthy() bool {
	return true
}

func (m *mockLocalStackManager) IsRunning() bool {
	return true
}

func (m *mockLocalStackManager) WaitForReady(ctx context.Context, timeout time.Duration) error {
	return nil
}

func (m *mockLocalStackManager) CheckServiceHealth(service string) error {
	return nil
}

func (m *mockLocalStackManager) GetConfig() *localstack.Config {
	return &localstack.Config{
		Enabled: true,
	}
}

func (m *mockLocalStackManager) GetContainer() *localstack.LocalStackContainer {
	return nil
}

func (m *mockLocalStackManager) EnableService(service string) error {
	return nil
}

func (m *mockLocalStackManager) DisableService(service string) error {
	return nil
}

var _ = Describe("S3 Integration", func() {
	var (
		integration kecsS3.Integration
		mockClient  *mockS3Client
		kubeClient  *fake.Clientset
		lsManager   localstack.Manager
		config      *kecsS3.Config
	)

	BeforeEach(func() {
		mockClient = &mockS3Client{}
		kubeClient = fake.NewSimpleClientset()
		lsManager = &mockLocalStackManager{}
		config = &kecsS3.Config{
			LocalStackEndpoint: "http://localstack:4566",
			Region:            "us-east-1",
			ForcePathStyle:    true,
		}
		
		integration = kecsS3.NewIntegrationWithClient(kubeClient, lsManager, config, mockClient)
	})

	Describe("DownloadFile", func() {
		It("should download a file from S3", func() {
			content := "test file content"
			mockClient.getObjectFunc = func(ctx context.Context, params *s3api.GetObjectRequest) (*s3api.GetObjectOutput, error) {
				Expect(params.Bucket).To(Equal("test-bucket"))
				Expect(params.Key).To(Equal("test-key"))
				return &s3api.GetObjectOutput{
					Body: []byte(content),
				}, nil
			}

			reader, err := integration.DownloadFile(context.Background(), "test-bucket", "test-key")
			Expect(err).NotTo(HaveOccurred())
			defer reader.Close()

			data, err := io.ReadAll(reader)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal(content))
		})

		It("should return error when download fails", func() {
			mockClient.getObjectFunc = func(ctx context.Context, params *s3api.GetObjectRequest) (*s3api.GetObjectOutput, error) {
				return nil, errors.New("download failed")
			}

			_, err := integration.DownloadFile(context.Background(), "test-bucket", "test-key")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("download failed"))
		})
	})

	Describe("UploadFile", func() {
		It("should upload a file to S3", func() {
			content := "test upload content"
			mockClient.putObjectFunc = func(ctx context.Context, params *s3api.PutObjectRequest) (*s3api.PutObjectOutput, error) {
				Expect(params.Bucket).To(Equal("test-bucket"))
				Expect(params.Key).To(Equal("test-key"))
				
				// Verify body content
				Expect(string(params.Body)).To(Equal(content))
				
				return &s3api.PutObjectOutput{}, nil
			}

			err := integration.UploadFile(context.Background(), "test-bucket", "test-key", strings.NewReader(content))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when upload fails", func() {
			mockClient.putObjectFunc = func(ctx context.Context, params *s3api.PutObjectRequest) (*s3api.PutObjectOutput, error) {
				return nil, errors.New("upload failed")
			}

			err := integration.UploadFile(context.Background(), "test-bucket", "test-key", strings.NewReader("content"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("upload failed"))
		})
	})

	Describe("HeadObject", func() {
		It("should get object metadata", func() {
			etag := "123456789"
			contentType := "text/plain"
			contentLength := int64(100)
			
			mockClient.headObjectFunc = func(ctx context.Context, params *s3api.HeadObjectRequest) (*s3api.HeadObjectOutput, error) {
				Expect(params.Bucket).To(Equal("test-bucket"))
				Expect(params.Key).To(Equal("test-key"))
				return &s3api.HeadObjectOutput{
					ContentLength: &contentLength,
					ContentType:   &contentType,
					ETag:          &etag,
				}, nil
			}

			metadata, err := integration.HeadObject(context.Background(), "test-bucket", "test-key")
			Expect(err).NotTo(HaveOccurred())
			Expect(metadata.Size).To(Equal(int64(100)))
			Expect(metadata.ContentType).To(Equal("text/plain"))
			Expect(metadata.ETag).To(Equal("123456789"))
		})

		It("should return error when head fails", func() {
			mockClient.headObjectFunc = func(ctx context.Context, params *s3api.HeadObjectRequest) (*s3api.HeadObjectOutput, error) {
				return nil, errors.New("head failed")
			}

			_, err := integration.HeadObject(context.Background(), "test-bucket", "test-key")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("head failed"))
		})
	})

	Describe("CreateBucket", func() {
		It("should create a bucket", func() {
			mockClient.createBucketFunc = func(ctx context.Context, params *s3api.CreateBucketRequest) (*s3api.CreateBucketOutput, error) {
				Expect(params.Bucket).To(Equal("new-bucket"))
				return &s3api.CreateBucketOutput{}, nil
			}

			err := integration.CreateBucket(context.Background(), "new-bucket")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not return error if bucket already exists", func() {
			mockClient.createBucketFunc = func(ctx context.Context, params *s3api.CreateBucketRequest) (*s3api.CreateBucketOutput, error) {
				return nil, errors.New("BucketAlreadyExists")
			}

			err := integration.CreateBucket(context.Background(), "existing-bucket")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should add location constraint for non-us-east-1 regions", func() {
			// Create integration with different region
			regionalConfig := &kecsS3.Config{
				LocalStackEndpoint: "http://localstack:4566",
				Region:            "eu-west-1",
				ForcePathStyle:    true,
			}
			regionalIntegration := kecsS3.NewIntegrationWithClient(kubeClient, lsManager, regionalConfig, mockClient)

			mockClient.createBucketFunc = func(ctx context.Context, params *s3api.CreateBucketRequest) (*s3api.CreateBucketOutput, error) {
				Expect(params.CreateBucketConfiguration).NotTo(BeNil())
				Expect(params.CreateBucketConfiguration.LocationConstraint).NotTo(BeNil())
				constraint, ok := (*params.CreateBucketConfiguration.LocationConstraint).(string)
				Expect(ok).To(BeTrue())
				Expect(constraint).To(Equal("eu-west-1"))
				return &s3api.CreateBucketOutput{}, nil
			}

			err := regionalIntegration.CreateBucket(context.Background(), "regional-bucket")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("DeleteObject", func() {
		It("should delete an object", func() {
			mockClient.deleteObjectFunc = func(ctx context.Context, params *s3api.DeleteObjectRequest) (*s3api.DeleteObjectOutput, error) {
				Expect(params.Bucket).To(Equal("test-bucket"))
				Expect(params.Key).To(Equal("test-key"))
				return &s3api.DeleteObjectOutput{}, nil
			}

			err := integration.DeleteObject(context.Background(), "test-bucket", "test-key")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when delete fails", func() {
			mockClient.deleteObjectFunc = func(ctx context.Context, params *s3api.DeleteObjectRequest) (*s3api.DeleteObjectOutput, error) {
				return nil, errors.New("delete failed")
			}

			err := integration.DeleteObject(context.Background(), "test-bucket", "test-key")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("delete failed"))
		})
	})
})