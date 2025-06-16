package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Integration defines the interface for S3 integration
type Integration interface {
	// DownloadFile downloads a file from S3
	DownloadFile(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	
	// UploadFile uploads a file to S3
	UploadFile(ctx context.Context, bucket, key string, reader io.Reader) error
	
	// HeadObject gets metadata for an S3 object
	HeadObject(ctx context.Context, bucket, key string) (*ObjectMetadata, error)
	
	// CreateBucket creates an S3 bucket if it doesn't exist
	CreateBucket(ctx context.Context, bucket string) error
	
	// DeleteObject deletes an object from S3
	DeleteObject(ctx context.Context, bucket, key string) error
}

// Config holds S3 integration configuration
type Config struct {
	LocalStackEndpoint string
	Region            string
	ForcePathStyle    bool
}

// ObjectMetadata contains S3 object metadata
type ObjectMetadata struct {
	Size         int64
	ContentType  string
	ETag         string
	LastModified string
}

// S3Client interface for S3 operations (for testing)
type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}