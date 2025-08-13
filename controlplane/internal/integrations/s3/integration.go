package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	s3api "github.com/nandemo-ya/kecs/controlplane/internal/s3/generated"
	"k8s.io/client-go/kubernetes"
)

// integration implements the S3 Integration interface
type integration struct {
	s3Client          S3Client
	kubeClient        kubernetes.Interface
	localstackManager localstack.Manager
	config            *Config
}

// NewIntegration creates a new S3 integration instance
func NewIntegration(kubeClient kubernetes.Interface, localstackManager localstack.Manager, cfg *Config) (Integration, error) {
	if cfg == nil {
		cfg = &Config{
			Region:         "us-east-1",
			ForcePathStyle: true, // Required for LocalStack
		}
	}

	// TODO: Create S3 client implementation that talks to LocalStack
	// For now, return error as this requires implementing the S3Client interface
	return nil, fmt.Errorf("S3 integration with generated types not yet implemented")
}

// NewIntegrationWithClient creates a new S3 integration with custom client (for testing)
func NewIntegrationWithClient(kubeClient kubernetes.Interface, localstackManager localstack.Manager, cfg *Config, s3Client S3Client) Integration {
	if cfg == nil {
		cfg = &Config{
			Region:         "us-east-1",
			ForcePathStyle: true,
		}
	}

	return &integration{
		s3Client:          s3Client,
		kubeClient:        kubeClient,
		localstackManager: localstackManager,
		config:            cfg,
	}
}

// DownloadFile downloads a file from S3
func (i *integration) DownloadFile(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	logging.Debug("Downloading S3 object", "bucket", bucket, "key", key)

	result, err := i.s3Client.GetObject(ctx, &s3api.GetObjectRequest{
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download S3 object: %w", err)
	}

	// Convert []byte to io.ReadCloser
	return &bodyReadCloser{bytes.NewReader(result.Body)}, nil
}

// UploadFile uploads a file to S3
func (i *integration) UploadFile(ctx context.Context, bucket, key string, reader io.Reader) error {
	logging.Debug("Uploading S3 object", "bucket", bucket, "key", key)

	// Read all data from reader to []byte
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read upload data: %w", err)
	}

	_, err = i.s3Client.PutObject(ctx, &s3api.PutObjectRequest{
		Bucket: bucket,
		Key:    key,
		Body:   data,
	})
	if err != nil {
		return fmt.Errorf("failed to upload S3 object: %w", err)
	}

	logging.Info("Successfully uploaded S3 object", "bucket", bucket, "key", key)
	return nil
}

// HeadObject gets metadata for an S3 object
func (i *integration) HeadObject(ctx context.Context, bucket, key string) (*ObjectMetadata, error) {
	result, err := i.s3Client.HeadObject(ctx, &s3api.HeadObjectRequest{
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object metadata: %w", err)
	}

	metadata := &ObjectMetadata{}

	if result.ContentLength != nil {
		metadata.Size = *result.ContentLength
	}

	if result.ContentType != nil {
		metadata.ContentType = *result.ContentType
	}

	if result.ETag != nil {
		metadata.ETag = strings.Trim(*result.ETag, "\"")
	}

	if result.LastModified != nil {
		metadata.LastModified = result.LastModified.Format(time.RFC3339)
	}

	return metadata, nil
}

// CreateBucket creates an S3 bucket if it doesn't exist
func (i *integration) CreateBucket(ctx context.Context, bucket string) error {
	logging.Debug("Creating S3 bucket", "bucket", bucket)

	input := &s3api.CreateBucketRequest{
		Bucket: bucket,
	}

	// Don't set LocationConstraint for us-east-1
	if i.config.Region != "us-east-1" {
		regionConstraint := s3api.BucketLocationConstraint(i.config.Region)
		input.CreateBucketConfiguration = &s3api.CreateBucketConfiguration{
			LocationConstraint: &regionConstraint,
		}
	}

	_, err := i.s3Client.CreateBucket(ctx, input)
	if err != nil {
		// Check if bucket already exists
		if strings.Contains(err.Error(), "BucketAlreadyExists") ||
			strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
			logging.Debug("S3 bucket already exists", "bucket", bucket)
			return nil
		}
		return fmt.Errorf("failed to create S3 bucket: %w", err)
	}

	logging.Info("Successfully created S3 bucket", "bucket", bucket)
	return nil
}

// DeleteObject deletes an object from S3
func (i *integration) DeleteObject(ctx context.Context, bucket, key string) error {
	logging.Debug("Deleting S3 object", "bucket", bucket, "key", key)

	_, err := i.s3Client.DeleteObject(ctx, &s3api.DeleteObjectRequest{
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		return fmt.Errorf("failed to delete S3 object: %w", err)
	}

	logging.Info("Successfully deleted S3 object", "bucket", bucket, "key", key)
	return nil
}

// bodyReadCloser wraps a bytes.Reader to implement io.ReadCloser
type bodyReadCloser struct {
	*bytes.Reader
}

func (b *bodyReadCloser) Close() error {
	return nil
}

// Helper function to create LocalStack configuration
// TODO: Implement LocalStack HTTP client that uses generated types
func createLocalStackConfig(endpoint, region string) error {
	if endpoint == "" {
		endpoint = "http://localstack.kecs-system.svc.cluster.local:4566"
	}
	// Implementation needed for LocalStack HTTP client
	return fmt.Errorf("LocalStack configuration not implemented with generated types")
}
