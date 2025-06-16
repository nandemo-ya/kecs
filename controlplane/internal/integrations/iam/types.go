package iam

import (
	"github.com/aws/aws-sdk-go-v2/service/iam"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
	
	// MapIAMPolicyToRBAC converts IAM policy to Kubernetes RBAC rules
	MapIAMPolicyToRBAC(policyDocument string) ([]rbacv1.PolicyRule, error)
}

// TaskRoleMapping represents the mapping between IAM role and Kubernetes resources
type TaskRoleMapping struct {
	RoleName           string
	RoleArn            string
	ServiceAccountName string
	Namespace          string
	TaskDefinitionArn  string
}

// PolicyMapping represents IAM policy to RBAC mapping
type PolicyMapping struct {
	IAMAction   string
	IAMResource string
	RBACVerbs   []string
	RBACGroups  []string
	RBACResources []string
}

// Config represents IAM integration configuration
type Config struct {
	LocalStackEndpoint string
	KubeNamespace      string
	RolePrefix         string // Prefix for created roles (e.g., "kecs-")
}

// ServiceAccountAnnotations defines annotations added to ServiceAccounts
var ServiceAccountAnnotations = struct {
	RoleArn          string
	RoleName         string
	TaskDefinitionArn string
}{
	RoleArn:          "kecs.io/iam-role-arn",
	RoleName:         "kecs.io/iam-role-name",
	TaskDefinitionArn: "kecs.io/task-definition-arn",
}

// Common IAM to RBAC mappings
var commonPolicyMappings = []PolicyMapping{
	// S3 mappings
	{
		IAMAction:     "s3:GetObject",
		IAMResource:   "*",
		RBACVerbs:     []string{"get"},
		RBACGroups:    []string{""},
		RBACResources: []string{"configmaps", "secrets"},
	},
	{
		IAMAction:     "s3:PutObject",
		IAMResource:   "*",
		RBACVerbs:     []string{"create", "update"},
		RBACGroups:    []string{""},
		RBACResources: []string{"configmaps"},
	},
	// CloudWatch Logs mappings
	{
		IAMAction:     "logs:CreateLogGroup",
		IAMResource:   "*",
		RBACVerbs:     []string{"create"},
		RBACGroups:    []string{""},
		RBACResources: []string{"events"},
	},
	{
		IAMAction:     "logs:CreateLogStream",
		IAMResource:   "*",
		RBACVerbs:     []string{"create"},
		RBACGroups:    []string{""},
		RBACResources: []string{"events"},
	},
	{
		IAMAction:     "logs:PutLogEvents",
		IAMResource:   "*",
		RBACVerbs:     []string{"create", "patch"},
		RBACGroups:    []string{""},
		RBACResources: []string{"events"},
	},
	// SSM Parameter Store mappings
	{
		IAMAction:     "ssm:GetParameter",
		IAMResource:   "*",
		RBACVerbs:     []string{"get"},
		RBACGroups:    []string{""},
		RBACResources: []string{"secrets", "configmaps"},
	},
	// Secrets Manager mappings
	{
		IAMAction:     "secretsmanager:GetSecretValue",
		IAMResource:   "*",
		RBACVerbs:     []string{"get"},
		RBACGroups:    []string{""},
		RBACResources: []string{"secrets"},
	},
}