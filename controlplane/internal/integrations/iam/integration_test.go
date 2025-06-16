package iam_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/iam"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("IAM Integration", func() {
	var (
		integration       iam.Integration
		kubeClient        *fake.Clientset
		localstackManager localstack.Manager
		config            *iam.Config
	)

	BeforeEach(func() {
		kubeClient = fake.NewSimpleClientset()
		localstackManager = &mockLocalStackManager{}
		config = &iam.Config{
			LocalStackEndpoint: "http://localhost:4566",
			KubeNamespace:      "default",
			RolePrefix:         "kecs-",
		}

		var err error
		integration, err = iam.NewIntegration(kubeClient, localstackManager, config)
		Expect(err).NotTo(HaveOccurred())
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
			Expect(sa.Annotations[iam.ServiceAccountAnnotations.RoleName]).To(Equal("kecs-my-task-role"))
			Expect(sa.Annotations[iam.ServiceAccountAnnotations.TaskDefinitionArn]).To(Equal(taskDefArn))
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
			Expect(sa.Annotations[iam.ServiceAccountAnnotations.RoleName]).To(Equal("kecs-unprefixed-role"))
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

	Describe("MapIAMPolicyToRBAC", func() {
		It("should map S3 permissions to ConfigMap access", func() {
			policyDoc := `{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": ["s3:GetObject", "s3:PutObject"],
					"Resource": "*"
				}]
			}`

			rules, err := integration.MapIAMPolicyToRBAC(policyDoc)
			Expect(err).NotTo(HaveOccurred())
			Expect(rules).To(HaveLen(2))

			// Check GetObject mapping
			Expect(rules[0].Verbs).To(ContainElement("get"))
			Expect(rules[0].Resources).To(ContainElements("configmaps", "secrets"))

			// Check PutObject mapping
			Expect(rules[1].Verbs).To(ContainElements("create", "update"))
			Expect(rules[1].Resources).To(ContainElement("configmaps"))
		})

		It("should map CloudWatch Logs permissions", func() {
			policyDoc := `{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"],
					"Resource": "*"
				}]
			}`

			rules, err := integration.MapIAMPolicyToRBAC(policyDoc)
			Expect(err).NotTo(HaveOccurred())
			Expect(rules).To(HaveLen(3))

			// All should map to events
			for _, rule := range rules {
				Expect(rule.Resources).To(ContainElement("events"))
			}
		})

		It("should handle wildcard actions", func() {
			policyDoc := `{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "s3:*",
					"Resource": "*"
				}]
			}`

			rules, err := integration.MapIAMPolicyToRBAC(policyDoc)
			Expect(err).NotTo(HaveOccurred())
			// Should match s3:GetObject and s3:PutObject
			Expect(len(rules)).To(BeNumerically(">=", 2))
		})

		It("should ignore Deny statements", func() {
			policyDoc := `{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Deny",
					"Action": "s3:GetObject",
					"Resource": "*"
				}]
			}`

			rules, err := integration.MapIAMPolicyToRBAC(policyDoc)
			Expect(err).NotTo(HaveOccurred())
			Expect(rules).To(BeEmpty())
		})
	})

	Describe("CreateInlinePolicy", func() {
		It("should create inline policy and update RBAC", func() {
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

			// Verify RBAC was created
			role, err := kubeClient.RbacV1().Roles("default").Get(context.Background(), "kecs-test-role-sa", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(role.Rules).To(HaveLen(1))
			Expect(role.Rules[0].Verbs).To(ContainElement("get"))
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

func (m *mockLocalStackManager) GetEndpoint() string {
	return "http://localhost:4566"
}

func (m *mockLocalStackManager) GetStatus() *localstack.Status {
	return &localstack.Status{
		Running: true,
		Healthy: true,
	}
}

func (m *mockLocalStackManager) EnableService(service string) error {
	return nil
}

func (m *mockLocalStackManager) DisableService(service string) error {
	return nil
}

func (m *mockLocalStackManager) GetEnabledServices() []string {
	return []string{"iam", "s3"}
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