package s3

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
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

	// Create S3 client configured for LocalStack
	awsCfg, err := createLocalStackConfig(cfg.LocalStackEndpoint, cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to create LocalStack config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.ForcePathStyle
	})

	return &integration{
		s3Client:          s3Client,
		kubeClient:        kubeClient,
		localstackManager: localstackManager,
		config:            cfg,
	}, nil
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
	klog.V(2).Infof("Downloading S3 object: s3://%s/%s", bucket, key)

	result, err := i.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download S3 object: %w", err)
	}

	return result.Body, nil
}

// UploadFile uploads a file to S3
func (i *integration) UploadFile(ctx context.Context, bucket, key string, reader io.Reader) error {
	klog.V(2).Infof("Uploading S3 object: s3://%s/%s", bucket, key)

	_, err := i.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   reader,
	})
	if err != nil {
		return fmt.Errorf("failed to upload S3 object: %w", err)
	}

	klog.Infof("Successfully uploaded S3 object: s3://%s/%s", bucket, key)
	return nil
}

// HeadObject gets metadata for an S3 object
func (i *integration) HeadObject(ctx context.Context, bucket, key string) (*ObjectMetadata, error) {
	result, err := i.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
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
	klog.V(2).Infof("Creating S3 bucket: %s", bucket)

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	// Don't set LocationConstraint for us-east-1
	if i.config.Region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(i.config.Region),
		}
	}

	_, err := i.s3Client.CreateBucket(ctx, input)
	if err != nil {
		// Check if bucket already exists
		if strings.Contains(err.Error(), "BucketAlreadyExists") || 
		   strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
			klog.V(2).Infof("S3 bucket already exists: %s", bucket)
			return nil
		}
		return fmt.Errorf("failed to create S3 bucket: %w", err)
	}

	klog.Infof("Successfully created S3 bucket: %s", bucket)
	return nil
}

// DeleteObject deletes an object from S3
func (i *integration) DeleteObject(ctx context.Context, bucket, key string) error {
	klog.V(2).Infof("Deleting S3 object: s3://%s/%s", bucket, key)

	_, err := i.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete S3 object: %w", err)
	}

	klog.Infof("Successfully deleted S3 object: s3://%s/%s", bucket, key)
	return nil
}

// Helper function to create LocalStack configuration
func createLocalStackConfig(endpoint, region string) (aws.Config, error) {
	if endpoint == "" {
		endpoint = "http://localstack.aws-services.svc.cluster.local:4566"
	}

	return config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               endpoint,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			}),
		),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
}