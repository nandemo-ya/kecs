package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// integration implements the IAM Integration interface
type integration struct {
	iamClient         IAMClient
	stsClient         STSClient
	kubeClient        kubernetes.Interface
	localstackManager localstack.Manager
	config            *Config
	roleMappings      map[string]*TaskRoleMapping // roleName -> mapping
}

// NewIntegration creates a new IAM integration instance
func NewIntegration(kubeClient kubernetes.Interface, localstackManager localstack.Manager, config *Config) (Integration, error) {
	if config == nil {
		config = &Config{
			KubeNamespace: "default",
			RolePrefix:    "kecs-",
		}
	}

	// Create IAM client configured for LocalStack
	cfg, err := createLocalStackConfig(config.LocalStackEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create LocalStack config: %w", err)
	}

	iamClient := iam.NewFromConfig(cfg)
	stsClient := sts.NewFromConfig(cfg)

	return &integration{
		iamClient:         iamClient,
		stsClient:         stsClient,
		kubeClient:        kubeClient,
		localstackManager: localstackManager,
		config:            config,
		roleMappings:      make(map[string]*TaskRoleMapping),
	}, nil
}

// NewIntegrationWithClients creates a new IAM integration with custom clients (for testing)
func NewIntegrationWithClients(kubeClient kubernetes.Interface, localstackManager localstack.Manager, config *Config, iamClient IAMClient, stsClient STSClient) Integration {
	if config == nil {
		config = &Config{
			KubeNamespace: "default",
			RolePrefix:    "kecs-",
		}
	}

	return &integration{
		iamClient:         iamClient,
		stsClient:         stsClient,
		kubeClient:        kubeClient,
		localstackManager: localstackManager,
		config:            config,
		roleMappings:      make(map[string]*TaskRoleMapping),
	}
}

// CreateTaskRole creates an IAM role and corresponding ServiceAccount
func (i *integration) CreateTaskRole(taskDefArn, roleName string, trustPolicy string) error {
	ctx := context.Background()

	// Ensure role name has prefix
	if !strings.HasPrefix(roleName, i.config.RolePrefix) {
		roleName = i.config.RolePrefix + roleName
	}

	// Create IAM role in LocalStack
	createRoleOutput, err := i.iamClient.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		Description:              aws.String(fmt.Sprintf("Task role for %s", taskDefArn)),
		Tags: []iamTypes.Tag{
			{
				Key:   aws.String("kecs:task-definition"),
				Value: aws.String(taskDefArn),
			},
			{
				Key:   aws.String("kecs:managed"),
				Value: aws.String("true"),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create IAM role: %w", err)
	}

	roleArn := aws.ToString(createRoleOutput.Role.Arn)

	// Create ServiceAccount in Kubernetes
	serviceAccountName := fmt.Sprintf("%s-sa", roleName)
	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: i.config.KubeNamespace,
			Annotations: map[string]string{
				ServiceAccountAnnotations.RoleArn:          roleArn,
				ServiceAccountAnnotations.RoleName:         roleName,
				ServiceAccountAnnotations.TaskDefinitionArn: taskDefArn,
			},
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kecs",
				"kecs.io/iam-role":            roleName,
			},
		},
	}

	_, err = i.kubeClient.CoreV1().ServiceAccounts(i.config.KubeNamespace).Create(ctx, serviceAccount, metav1.CreateOptions{})
	if err != nil {
		// Rollback IAM role creation
		i.iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
			RoleName: aws.String(roleName),
		})
		return fmt.Errorf("failed to create ServiceAccount: %w", err)
	}

	// Store mapping
	i.roleMappings[roleName] = &TaskRoleMapping{
		RoleName:           roleName,
		RoleArn:            roleArn,
		ServiceAccountName: serviceAccountName,
		Namespace:          i.config.KubeNamespace,
		TaskDefinitionArn:  taskDefArn,
	}

	klog.Infof("Created IAM role %s and ServiceAccount %s for task definition %s", roleName, serviceAccountName, taskDefArn)
	return nil
}

// CreateTaskExecutionRole creates an IAM execution role with necessary permissions
func (i *integration) CreateTaskExecutionRole(roleName string) error {
	ctx := context.Background()

	// Ensure role name has prefix
	if !strings.HasPrefix(roleName, i.config.RolePrefix) {
		roleName = i.config.RolePrefix + roleName
	}

	// Trust policy for ECS tasks
	trustPolicy := `{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": {
				"Service": "ecs-tasks.amazonaws.com"
			},
			"Action": "sts:AssumeRole"
		}]
	}`

	// Create the role
	_, err := i.iamClient.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		Description:              aws.String("ECS task execution role"),
		Tags: []iamTypes.Tag{
			{
				Key:   aws.String("kecs:role-type"),
				Value: aws.String("execution"),
			},
			{
				Key:   aws.String("kecs:managed"),
				Value: aws.String("true"),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create execution role: %w", err)
	}

	// Attach AWS managed policy for ECS task execution
	_, err = i.iamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String("arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"),
	})
	if err != nil {
		klog.Warningf("Failed to attach managed policy (this may be normal in LocalStack): %v", err)
		// Create a custom policy instead
		return i.createExecutionRolePolicy(roleName)
	}

	return nil
}

// AttachPolicyToRole attaches a policy to a role
func (i *integration) AttachPolicyToRole(roleName, policyArn string) error {
	ctx := context.Background()

	_, err := i.iamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyArn),
	})
	if err != nil {
		return fmt.Errorf("failed to attach policy: %w", err)
	}

	klog.Infof("Attached policy %s to role %s", policyArn, roleName)
	return nil
}

// CreateInlinePolicy creates an inline policy for a role
func (i *integration) CreateInlinePolicy(roleName, policyName, policyDocument string) error {
	ctx := context.Background()

	_, err := i.iamClient.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(policyDocument),
	})
	if err != nil {
		return fmt.Errorf("failed to create inline policy: %w", err)
	}

	klog.Infof("Created inline policy %s for role %s", policyName, roleName)
	return nil
}

// DeleteRole deletes an IAM role and its ServiceAccount
func (i *integration) DeleteRole(roleName string) error {
	ctx := context.Background()

	// Get mapping
	mapping, exists := i.roleMappings[roleName]
	if !exists {
		// Try to find ServiceAccount by annotation
		sa, err := i.findServiceAccountByRole(roleName)
		if err == nil && sa != nil {
			mapping = &TaskRoleMapping{
				RoleName:           roleName,
				ServiceAccountName: sa.Name,
				Namespace:          sa.Namespace,
			}
		}
	}

	// Delete ServiceAccount
	if mapping != nil {
		err := i.kubeClient.CoreV1().ServiceAccounts(mapping.Namespace).Delete(ctx, mapping.ServiceAccountName, metav1.DeleteOptions{})
		if err != nil {
			klog.Warningf("Failed to delete ServiceAccount: %v", err)
		}

		delete(i.roleMappings, roleName)
	}

	// Detach all policies
	policies, err := i.iamClient.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, policy := range policies.AttachedPolicies {
			i.iamClient.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
				RoleName:  aws.String(roleName),
				PolicyArn: policy.PolicyArn,
			})
		}
	}

	// Delete inline policies
	inlinePolicies, err := i.iamClient.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, policyName := range inlinePolicies.PolicyNames {
			i.iamClient.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
				RoleName:   aws.String(roleName),
				PolicyName: aws.String(policyName),
			})
		}
	}

	// Delete IAM role
	_, err = i.iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete IAM role: %w", err)
	}

	return nil
}

// GetServiceAccountForRole returns the ServiceAccount name for a role
func (i *integration) GetServiceAccountForRole(roleName string) (string, error) {
	if mapping, exists := i.roleMappings[roleName]; exists {
		return mapping.ServiceAccountName, nil
	}

	// Try to find by annotation
	sa, err := i.findServiceAccountByRole(roleName)
	if err != nil {
		return "", err
	}
	if sa == nil {
		return "", fmt.Errorf("no ServiceAccount found for role %s", roleName)
	}

	return sa.Name, nil
}

// GetRoleCredentials gets temporary credentials for a role (if using STS)
func (i *integration) GetRoleCredentials(roleName string) (*Credentials, error) {
	// For LocalStack, we use static test credentials
	// In a real implementation, this could use STS AssumeRole
	return &DefaultLocalStackCredentials, nil
}

// Helper functions

func createLocalStackConfig(endpoint string) (aws.Config, error) {
	if endpoint == "" {
		return aws.Config{}, fmt.Errorf("no endpoint configured")
	}
	
	return config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithBaseEndpoint(endpoint),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
}

func (i *integration) createExecutionRolePolicy(roleName string) error {
	// Custom policy for task execution
	policyDocument := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Action": [
					"ecr:GetAuthorizationToken",
					"ecr:BatchCheckLayerAvailability",
					"ecr:GetDownloadUrlForLayer",
					"ecr:BatchGetImage",
					"logs:CreateLogStream",
					"logs:PutLogEvents"
				],
				"Resource": "*"
			}
		]
	}`

	return i.CreateInlinePolicy(roleName, "TaskExecutionPolicy", policyDocument)
}


func (i *integration) findServiceAccountByRole(roleName string) (*v1.ServiceAccount, error) {
	ctx := context.Background()

	// List all ServiceAccounts and find by annotation
	saList, err := i.kubeClient.CoreV1().ServiceAccounts(i.config.KubeNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/managed-by=kecs",
	})
	if err != nil {
		return nil, err
	}

	for _, sa := range saList.Items {
		if sa.Annotations[ServiceAccountAnnotations.RoleName] == roleName {
			return &sa, nil
		}
	}

	return nil, nil
}

