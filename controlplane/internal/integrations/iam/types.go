package iam

import (
	"context"

	iamapi "github.com/nandemo-ya/kecs/controlplane/internal/iam/generated"
	stsapi "github.com/nandemo-ya/kecs/controlplane/internal/sts/generated"
)

// Integration represents the IAM-Kubernetes integration
type Integration interface {
	// CreateTaskRole creates an IAM role in LocalStack and corresponding ServiceAccount in Kubernetes
	CreateTaskRole(taskDefArn, roleName string, trustPolicy string) error

	// CreateTaskExecutionRole creates an IAM execution role for ECS tasks
	CreateTaskExecutionRole(roleName string) error

	// AttachPolicyToRole attaches an IAM policy to a role
	AttachPolicyToRole(roleName, policyArn string) error

	// CreateInlinePolicy creates an inline policy for a role
	CreateInlinePolicy(roleName, policyName, policyDocument string) error

	// DeleteRole deletes an IAM role and its corresponding ServiceAccount
	DeleteRole(roleName string) error

	// GetServiceAccountForRole returns the ServiceAccount name for a given IAM role
	GetServiceAccountForRole(roleName string) (string, error)

	// GetRoleCredentials gets temporary credentials for a role (if using STS)
	GetRoleCredentials(roleName string) (*Credentials, error)
}

// TaskRoleMapping represents the mapping between IAM role and Kubernetes resources
type TaskRoleMapping struct {
	RoleName           string
	RoleArn            string
	ServiceAccountName string
	Namespace          string
	TaskDefinitionArn  string
}

// Credentials represents AWS credentials for a role
type Credentials struct {
	AccessKeyId     string
	SecretAccessKey string
	SessionToken    string
	Expiration      string
}

// Config represents IAM integration configuration
type Config struct {
	LocalStackEndpoint string
	KubeNamespace      string
	RolePrefix         string // Prefix for created roles (e.g., "kecs-")
}

// ServiceAccountAnnotations defines annotations added to ServiceAccounts
var ServiceAccountAnnotations = struct {
	RoleArn           string
	RoleName          string
	TaskDefinitionArn string
}{
	RoleArn:           "kecs.io/iam-role-arn",
	RoleName:          "kecs.io/iam-role-name",
	TaskDefinitionArn: "kecs.io/task-definition-arn",
}

// Default AWS credentials for LocalStack
var DefaultLocalStackCredentials = Credentials{
	AccessKeyId:     "test",
	SecretAccessKey: "test",
	SessionToken:    "",
}

// IAMClient interface for IAM operations (for testing)
type IAMClient interface {
	CreateRole(ctx context.Context, params *iamapi.CreateRoleRequest) (*iamapi.CreateRoleResponse, error)
	DeleteRole(ctx context.Context, params *iamapi.DeleteRoleRequest) (*iamapi.Unit, error)
	AttachRolePolicy(ctx context.Context, params *iamapi.AttachRolePolicyRequest) (*iamapi.Unit, error)
	DetachRolePolicy(ctx context.Context, params *iamapi.DetachRolePolicyRequest) (*iamapi.Unit, error)
	PutRolePolicy(ctx context.Context, params *iamapi.PutRolePolicyRequest) (*iamapi.Unit, error)
	DeleteRolePolicy(ctx context.Context, params *iamapi.DeleteRolePolicyRequest) (*iamapi.Unit, error)
	ListAttachedRolePolicies(ctx context.Context, params *iamapi.ListAttachedRolePoliciesRequest) (*iamapi.ListAttachedRolePoliciesResponse, error)
	ListRolePolicies(ctx context.Context, params *iamapi.ListRolePoliciesRequest) (*iamapi.ListRolePoliciesResponse, error)
}

// STSClient interface for STS operations (for testing)
type STSClient interface {
	AssumeRole(ctx context.Context, params *stsapi.AssumeRoleRequest) (*stsapi.AssumeRoleResponse, error)
}
