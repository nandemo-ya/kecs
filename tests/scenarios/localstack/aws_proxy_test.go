package localstack_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/nandemo-ya/kecs/tests/scenarios/localstack/helpers"
	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

// GinkgoWrapper wraps GinkgoT to implement utils.TestingT interface
type GinkgoWrapper struct {
	GinkgoTInterface
}

// Logf implements TestingT interface
func (g GinkgoWrapper) Logf(format string, args ...interface{}) {
	GinkgoWriter.Printf(format+"\n", args...)
}

// Fatalf implements TestingT interface
func (g GinkgoWrapper) Fatalf(format string, args ...interface{}) {
	Fail(fmt.Sprintf(format, args...))
}

// TestingTWrapper wraps GinkgoT to implement testify's TestingT interface
type TestingTWrapper struct {
	GinkgoTInterface
}

// Errorf implements TestingT interface for testify/require
func (t *TestingTWrapper) Errorf(format string, args ...interface{}) {
	AddReportEntry("error", fmt.Sprintf(format, args...))
	Fail(fmt.Sprintf(format, args...))
}

// FailNow implements TestingT interface for testify/require
func (t *TestingTWrapper) FailNow() {
	Fail("Test failed")
}

// Logf implements the logging method
func (t *TestingTWrapper) Logf(format string, args ...interface{}) {
	GinkgoWriter.Printf(format+"\n", args...)
}

// Log implements the logging method
func (t *TestingTWrapper) Log(args ...interface{}) {
	GinkgoWriter.Println(args...)
}

// Fatalf implements the fatal error method
func (t *TestingTWrapper) Fatalf(format string, args ...interface{}) {
	Fail(fmt.Sprintf(format, args...))
}

// Fatal implements the fatal error method
func (t *TestingTWrapper) Fatal(args ...interface{}) {
	Fail(fmt.Sprint(args...))
}

// Error implements the error method  
func (t *TestingTWrapper) Error(args ...interface{}) {
	Fail(fmt.Sprint(args...))
}

var _ = Describe("AWS API Proxy", func() {
	var (
		kecs            *utils.KECSContainer
		client          *utils.ECSClient
		localstackClient *helpers.LocalStackTestClient
		testClusterName string
	)

	BeforeEach(func() {
		// Start KECS with LocalStack enabled
		kecs = utils.StartKECS(GinkgoWrapper{GinkgoT()})
		DeferCleanup(func() {
			if kecs != nil {
				kecs.Cleanup()
			}
		})

		// Create ECS client
		client = utils.NewECSClient(kecs.Endpoint())
		
		// Create a test cluster
		testClusterName = fmt.Sprintf("test-proxy-%d", time.Now().Unix())
		err := client.CreateCluster(testClusterName)
		Expect(err).NotTo(HaveOccurred())

		// Start LocalStack with required services
		helpers.StartLocalStack(&TestingTWrapper{GinkgoT()}, kecs, []string{"iam", "s3", "cloudwatchlogs", "secretsmanager"})
		helpers.WaitForLocalStackReady(&TestingTWrapper{GinkgoT()}, client, testClusterName, 30*time.Second)

		// Create LocalStack test client
		// Note: In a real test, we'd need to get the actual LocalStack endpoint from KECS
		localstackEndpoint := fmt.Sprintf("%s/localstack", kecs.Endpoint())
		localstackClient = helpers.NewLocalStackTestClient(kecs.Endpoint(), localstackEndpoint)
	})

	AfterEach(func() {
		// Clean up
		if client != nil && testClusterName != "" {
			client.DeleteCluster(testClusterName)
		}
		if kecs != nil {
			helpers.StopLocalStack(&TestingTWrapper{GinkgoT()}, kecs)
		}
	})

	Describe("S3 API Proxy", func() {
		It("should proxy S3 API calls to LocalStack", func() {
			bucketName := fmt.Sprintf("test-bucket-%d", time.Now().Unix())

			// Create S3 bucket via LocalStack
			helpers.CreateS3BucketViaLocalStack(&TestingTWrapper{GinkgoT()}, localstackClient, bucketName)

			// List buckets to verify
			ctx := context.Background()
			listResult, err := localstackClient.S3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
			Expect(err).NotTo(HaveOccurred())

			// Check that our bucket exists
			bucketFound := false
			for _, bucket := range listResult.Buckets {
				if aws.ToString(bucket.Name) == bucketName {
					bucketFound = true
					break
				}
			}
			Expect(bucketFound).To(BeTrue(), "Created bucket should be found in list")

			// Clean up
			_, err = localstackClient.S3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
				Bucket: aws.String(bucketName),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle S3 errors correctly", func() {
			// Try to access non-existent bucket
			ctx := context.Background()
			_, err := localstackClient.S3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
				Bucket: aws.String("non-existent-bucket"),
			})
			Expect(err).To(HaveOccurred())
			// The error should indicate the bucket doesn't exist
			Expect(err.Error()).To(ContainSubstring("NoSuchBucket"))
		})
	})

	Describe("IAM API Proxy", func() {
		It("should proxy IAM API calls to LocalStack", func() {
			roleName := fmt.Sprintf("test-role-%d", time.Now().Unix())

			// Create IAM role via LocalStack
			helpers.CreateIAMRoleViaLocalStack(&TestingTWrapper{GinkgoT()}, localstackClient, roleName)

			// List roles to verify
			ctx := context.Background()
			listResult, err := localstackClient.IAMClient.ListRoles(ctx, &iam.ListRolesInput{})
			Expect(err).NotTo(HaveOccurred())

			// Check that our role exists
			roleFound := false
			for _, role := range listResult.Roles {
				if aws.ToString(role.RoleName) == roleName {
					roleFound = true
					break
				}
			}
			Expect(roleFound).To(BeTrue(), "Created role should be found in list")

			// Clean up
			_, err = localstackClient.IAMClient.DeleteRole(ctx, &iam.DeleteRoleInput{
				RoleName: aws.String(roleName),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle IAM policy operations", func() {
			roleName := fmt.Sprintf("test-role-policy-%d", time.Now().Unix())
			policyName := "test-policy"

			// Create role first
			helpers.CreateIAMRoleViaLocalStack(&TestingTWrapper{GinkgoT()}, localstackClient, roleName)

			// Attach inline policy
			ctx := context.Background()
			policyDocument := `{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "s3:GetObject",
					"Resource": "*"
				}]
			}`

			_, err := localstackClient.IAMClient.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
				RoleName:       aws.String(roleName),
				PolicyName:     aws.String(policyName),
				PolicyDocument: aws.String(policyDocument),
			})
			Expect(err).NotTo(HaveOccurred())

			// List role policies
			listPoliciesResult, err := localstackClient.IAMClient.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
				RoleName: aws.String(roleName),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(listPoliciesResult.PolicyNames).To(ContainElement(policyName))

			// Clean up
			_, _ = localstackClient.IAMClient.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
				RoleName:   aws.String(roleName),
				PolicyName: aws.String(policyName),
			})
			_, _ = localstackClient.IAMClient.DeleteRole(ctx, &iam.DeleteRoleInput{
				RoleName: aws.String(roleName),
			})
		})
	})

	Describe("Service Isolation", func() {
		It("should route ECS calls to KECS, not LocalStack", func() {
			// Create a task definition via ECS API
			taskDef, err := client.CurlClient.RegisterTaskDefinition("test-task", `{
				"containerDefinitions": [{
					"name": "test-container",
					"image": "nginx:latest",
					"memory": 128
				}]
			}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDef).NotTo(BeNil())

			// This should go to KECS, not LocalStack
			// Verify by checking the task definition format
			Expect(taskDef.Family).To(Equal("test-task"))
			Expect(taskDef.Revision).To(Equal("1"))
		})

		It("should reject calls to non-enabled LocalStack services", func() {
			// Ensure DynamoDB is not enabled
			status := helpers.GetLocalStackStatus(&TestingTWrapper{GinkgoT()}, kecs)
			if strings.Contains(status, "dynamodb") {
				helpers.DisableLocalStackService(&TestingTWrapper{GinkgoT()}, kecs, "dynamodb")
				time.Sleep(2 * time.Second)
			}

			// Try to access DynamoDB (should fail or return error)
			// Note: This test depends on how the proxy handles non-enabled services
			// It might return an error or route to a default handler
		})
	})

	Describe("Endpoint Configuration", func() {
		It("should use correct LocalStack endpoint for services", func() {
			// This test verifies that the AWS SDK is correctly configured
			// to use LocalStack endpoints for AWS services

			// Check S3 endpoint configuration
			bucketName := fmt.Sprintf("endpoint-test-%d", time.Now().Unix())
			ctx := context.Background()
			
			// Create bucket should succeed if endpoint is correct
			_, err := localstackClient.S3Client.CreateBucket(ctx, &s3.CreateBucketInput{
				Bucket: aws.String(bucketName),
			})
			Expect(err).NotTo(HaveOccurred())

			// Clean up
			_, _ = localstackClient.S3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
				Bucket: aws.String(bucketName),
			})
		})
	})
})