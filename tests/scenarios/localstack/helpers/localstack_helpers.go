package helpers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
	"github.com/stretchr/testify/require"
)

// TestingT interface that matches what testify/require expects
type TestingT interface {
	Errorf(format string, args ...interface{})
	FailNow()
	Logf(format string, args ...interface{})
	Fatal(args ...interface{})
}

// LocalStackTestClient wraps AWS clients configured for LocalStack
type LocalStackTestClient struct {
	Endpoint  string
	ECSClient *ecs.Client
	IAMClient *iam.Client
	S3Client  *s3.Client
	HTTPClient *http.Client
}

// NewLocalStackTestClient creates a new test client configured for LocalStack
func NewLocalStackTestClient(kecsEndpoint string, localstackEndpoint string) *LocalStackTestClient {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				switch service {
				case ecs.ServiceID:
					return aws.Endpoint{URL: kecsEndpoint}, nil
				default:
					return aws.Endpoint{URL: localstackEndpoint}, nil
				}
			}),
		),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	return &LocalStackTestClient{
		Endpoint:   localstackEndpoint,
		ECSClient:  ecs.NewFromConfig(cfg),
		IAMClient:  iam.NewFromConfig(cfg),
		S3Client:   s3.NewFromConfig(cfg),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// WaitForLocalStackReady waits for LocalStack to be ready
func WaitForLocalStackReady(t TestingT, client *utils.ECSClient, clusterName string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for LocalStack to be ready")
		case <-ticker.C:
			// Check LocalStack status via KECS API
			status, err := client.GetLocalStackStatus(clusterName)
			if err == nil && status == "healthy" {
				return
			}
		}
	}
}

// StartLocalStack starts LocalStack via KECS CLI
func StartLocalStack(t TestingT, kecs *utils.KECSContainer, services []string) {
	args := []string{"localstack", "start"}
	if len(services) > 0 {
		args = append(args, "--services", fmt.Sprintf("%v", services))
	}

	output, err := kecs.ExecuteCommand(args...)
	require.NoError(t, err, "Failed to start LocalStack: %s", output)
}

// StopLocalStack stops LocalStack via KECS CLI
func StopLocalStack(t TestingT, kecs *utils.KECSContainer) {
	output, err := kecs.ExecuteCommand("localstack", "stop")
	require.NoError(t, err, "Failed to stop LocalStack: %s", output)
}

// EnableLocalStackService enables a LocalStack service
func EnableLocalStackService(t TestingT, kecs *utils.KECSContainer, service string) {
	output, err := kecs.ExecuteCommand("localstack", "enable", service)
	require.NoError(t, err, "Failed to enable LocalStack service %s: %s", service, output)
}

// DisableLocalStackService disables a LocalStack service
func DisableLocalStackService(t TestingT, kecs *utils.KECSContainer, service string) {
	output, err := kecs.ExecuteCommand("localstack", "disable", service)
	require.NoError(t, err, "Failed to disable LocalStack service %s: %s", service, output)
}

// GetLocalStackStatus gets LocalStack status
func GetLocalStackStatus(t TestingT, kecs *utils.KECSContainer) string {
	output, err := kecs.ExecuteCommand("localstack", "status")
	require.NoError(t, err, "Failed to get LocalStack status: %s", output)
	return output
}

// RestartLocalStack restarts LocalStack
func RestartLocalStack(t TestingT, kecs *utils.KECSContainer) {
	output, err := kecs.ExecuteCommand("localstack", "restart")
	require.NoError(t, err, "Failed to restart LocalStack: %s", output)
}

// CheckAWSServiceAccessible checks if an AWS service is accessible via LocalStack
func CheckAWSServiceAccessible(t TestingT, endpoint string, service string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	
	// LocalStack health endpoint
	healthURL := fmt.Sprintf("%s/_localstack/health", endpoint)
	resp, err := client.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// CreateS3BucketViaLocalStack creates an S3 bucket using LocalStack
func CreateS3BucketViaLocalStack(t TestingT, client *LocalStackTestClient, bucketName string) {
	ctx := context.Background()
	_, err := client.S3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	require.NoError(t, err, "Failed to create S3 bucket via LocalStack")
}

// CreateIAMRoleViaLocalStack creates an IAM role using LocalStack
func CreateIAMRoleViaLocalStack(t TestingT, client *LocalStackTestClient, roleName string) {
	ctx := context.Background()
	trustPolicy := `{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": {"Service": "ecs-tasks.amazonaws.com"},
			"Action": "sts:AssumeRole"
		}]
	}`
	
	_, err := client.IAMClient.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
	})
	require.NoError(t, err, "Failed to create IAM role via LocalStack")
}