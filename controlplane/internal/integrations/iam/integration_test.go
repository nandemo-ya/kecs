package iam_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	iamapi "github.com/nandemo-ya/kecs/controlplane/internal/iam/generated"
	stsapi "github.com/nandemo-ya/kecs/controlplane/internal/sts/generated"
	kecsIAM "github.com/nandemo-ya/kecs/controlplane/internal/integrations/iam"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// mockIAMClient is a mock implementation of IAMClient
type mockIAMClient struct{}

func (m *mockIAMClient) CreateRole(ctx context.Context, params *iamapi.CreateRoleRequest) (*iamapi.CreateRoleResponse, error) {
	arn := "arn:aws:iam::123456789012:role/" + params.RoleName
	return &iamapi.CreateRoleResponse{
		Role: iamapi.Role{
			RoleName: params.RoleName,
			Arn:      arn,
			Path:     "/",
			RoleId:   "AIDAQ3EGKSO2V4EXAMPLE",
			CreateDate: time.Now(),
		},
	}, nil
}

func (m *mockIAMClient) DeleteRole(ctx context.Context, params *iamapi.DeleteRoleRequest) (*iamapi.Unit, error) {
	return &iamapi.Unit{}, nil
}

func (m *mockIAMClient) AttachRolePolicy(ctx context.Context, params *iamapi.AttachRolePolicyRequest) (*iamapi.Unit, error) {
	return &iamapi.Unit{}, nil
}

func (m *mockIAMClient) DetachRolePolicy(ctx context.Context, params *iamapi.DetachRolePolicyRequest) (*iamapi.Unit, error) {
	return &iamapi.Unit{}, nil
}

func (m *mockIAMClient) PutRolePolicy(ctx context.Context, params *iamapi.PutRolePolicyRequest) (*iamapi.Unit, error) {
	return &iamapi.Unit{}, nil
}

func (m *mockIAMClient) DeleteRolePolicy(ctx context.Context, params *iamapi.DeleteRolePolicyRequest) (*iamapi.Unit, error) {
	return &iamapi.Unit{}, nil
}

func (m *mockIAMClient) ListAttachedRolePolicies(ctx context.Context, params *iamapi.ListAttachedRolePoliciesRequest) (*iamapi.ListAttachedRolePoliciesResponse, error) {
	return &iamapi.ListAttachedRolePoliciesResponse{
		AttachedPolicies: []iamapi.AttachedPolicy{},
	}, nil
}

func (m *mockIAMClient) ListRolePolicies(ctx context.Context, params *iamapi.ListRolePoliciesRequest) (*iamapi.ListRolePoliciesResponse, error) {
	return &iamapi.ListRolePoliciesResponse{
		PolicyNames: []string{},
	}, nil
}

// mockSTSClient is a mock implementation of STSClient
type mockSTSClient struct{}

func (m *mockSTSClient) AssumeRole(ctx context.Context, params *stsapi.AssumeRoleRequest) (*stsapi.AssumeRoleResponse, error) {
	return &stsapi.AssumeRoleResponse{}, nil
}

// mockLocalStackManager is a mock implementation of localstack.Manager
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

func (m *mockLocalStackManager) IsRunning() bool {
	return true
}

func (m *mockLocalStackManager) IsHealthy() bool {
	return true
}

func (m *mockLocalStackManager) GetEndpoint() (string, error) {
	return "http://localhost:4566", nil
}

func (m *mockLocalStackManager) GetStatus() (*localstack.Status, error) {
	return &localstack.Status{
		Running: true,
		Healthy: true,
	}, nil
}

func (m *mockLocalStackManager) EnableService(service string) error {
	return nil
}

func (m *mockLocalStackManager) DisableService(service string) error {
	return nil
}

func (m *mockLocalStackManager) GetEnabledServices() ([]string, error) {
	return []string{"iam", "s3"}, nil
}

func (m *mockLocalStackManager) GetConfig() *localstack.Config {
	return &localstack.Config{
		Enabled: true,
	}
}

func (m *mockLocalStackManager) GetContainer() *localstack.LocalStackContainer {
	return nil
}

func (m *mockLocalStackManager) UpdateServices(services []string) error {
	return nil
}

func (m *mockLocalStackManager) GetServiceEndpoint(service string) (string, error) {
	return "http://localhost:4566", nil
}

func (m *mockLocalStackManager) WaitForReady(ctx context.Context, timeout time.Duration) error {
	return nil
}

func (m *mockLocalStackManager) CheckServiceHealth(service string) error {
	return nil
}

var _ = Describe("IAM Integration", func() {
	var (
		integration       kecsIAM.Integration
		kubeClient        *fake.Clientset
		// localstackManager localstack.Manager // unused in temp implementation
		config            *kecsIAM.Config
		iamClient         kecsIAM.IAMClient
		stsClient         kecsIAM.STSClient
	)

	BeforeEach(func() {
		kubeClient = fake.NewSimpleClientset()
		// localstackManager = &mockLocalStackManager{} // unused in temp implementation
		config = &kecsIAM.Config{
			LocalStackEndpoint: "http://localhost:4566",
			KubeNamespace:      "default",
			RolePrefix:         "kecs-",
		}

		iamClient = &mockIAMClient{}
		stsClient = &mockSTSClient{}
		
		// Use the test constructor with mocked clients
		integration = kecsIAM.NewIntegrationWithClient(
			kubeClient,
			iamClient,
			stsClient,
			config,
		)
	})

	Describe("CreateTaskRole", func() {
		It("should create IAM role and ServiceAccount", func() {
			taskDefArn := "arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"
			roleName := "my-task-role"
			trustPolicy := `{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Principal": {"Service": "ecs-tasks.amazonaws.com"},
					"Action": "sts:AssumeRole"
				}]
			}`

			err := integration.CreateTaskRole(taskDefArn, roleName, trustPolicy)
			Expect(err).NotTo(HaveOccurred())

			// Verify ServiceAccount was created
			serviceAccountName := "kecs-my-task-role-sa"
			sa, err := kubeClient.CoreV1().ServiceAccounts("default").Get(context.Background(), serviceAccountName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(sa.Name).To(Equal(serviceAccountName))
			Expect(sa.Annotations[kecsIAM.ServiceAccountAnnotations.RoleName]).To(Equal("kecs-my-task-role"))
			Expect(sa.Annotations[kecsIAM.ServiceAccountAnnotations.TaskDefinitionArn]).To(Equal(taskDefArn))
		})

		It("should add prefix to role name if not present", func() {
			taskDefArn := "arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"
			roleName := "unprefixed-role"
			trustPolicy := `{"Version": "2012-10-17", "Statement": []}`

			err := integration.CreateTaskRole(taskDefArn, roleName, trustPolicy)
			Expect(err).NotTo(HaveOccurred())

			// ServiceAccount should have prefixed role name
			serviceAccountName := "kecs-unprefixed-role-sa"
			sa, err := kubeClient.CoreV1().ServiceAccounts("default").Get(context.Background(), serviceAccountName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(sa.Annotations[kecsIAM.ServiceAccountAnnotations.RoleName]).To(Equal("kecs-unprefixed-role"))
		})
	})

	Describe("GetServiceAccountForRole", func() {
		It("should return ServiceAccount name for existing role", func() {
			// First create a role
			taskDefArn := "arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"
			roleName := "test-role"
			trustPolicy := `{"Version": "2012-10-17", "Statement": []}`

			err := integration.CreateTaskRole(taskDefArn, roleName, trustPolicy)
			Expect(err).NotTo(HaveOccurred())

			// Get ServiceAccount name
			saName, err := integration.GetServiceAccountForRole("kecs-test-role")
			Expect(err).NotTo(HaveOccurred())
			Expect(saName).To(Equal("kecs-test-role-sa"))
		})

		It("should return error for non-existent role", func() {
			_, err := integration.GetServiceAccountForRole("non-existent-role")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetRoleCredentials", func() {
		It("should return LocalStack test credentials", func() {
			creds, err := integration.GetRoleCredentials("test-role")
			Expect(err).NotTo(HaveOccurred())
			Expect(creds).NotTo(BeNil())
			Expect(creds.AccessKeyId).To(Equal("test"))
			Expect(creds.SecretAccessKey).To(Equal("test"))
		})
	})

	Describe("CreateInlinePolicy", func() {
		It("should create inline policy without RBAC", func() {
			// First create a role
			taskDefArn := "arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"
			roleName := "test-role"
			trustPolicy := `{"Version": "2012-10-17", "Statement": []}`

			err := integration.CreateTaskRole(taskDefArn, roleName, trustPolicy)
			Expect(err).NotTo(HaveOccurred())

			// Create inline policy
			policyName := "test-policy"
			policyDoc := `{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "s3:GetObject",
					"Resource": "*"
				}]
			}`

			err = integration.CreateInlinePolicy("kecs-test-role", policyName, policyDoc)
			Expect(err).NotTo(HaveOccurred())
			
			// Only ServiceAccount should exist, no RBAC resources
			_, err = kubeClient.CoreV1().ServiceAccounts("default").Get(context.Background(), "kecs-test-role-sa", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("DeleteRole", func() {
		It("should delete IAM role and ServiceAccount", func() {
			// First create a role
			taskDefArn := "arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"
			roleName := "test-role"
			trustPolicy := `{"Version": "2012-10-17", "Statement": []}`

			err := integration.CreateTaskRole(taskDefArn, roleName, trustPolicy)
			Expect(err).NotTo(HaveOccurred())

			// Delete the role
			err = integration.DeleteRole("kecs-test-role")
			Expect(err).NotTo(HaveOccurred())

			// Verify ServiceAccount was deleted
			_, err = kubeClient.CoreV1().ServiceAccounts("default").Get(context.Background(), "kecs-test-role-sa", metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
		})
	})
})
