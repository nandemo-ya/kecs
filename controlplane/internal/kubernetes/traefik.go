package kubernetes

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes/resources"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// TraefikManager manages the global Traefik deployment
type TraefikManager struct {
	client kubernetes.Interface
}

// NewTraefikManager creates a new TraefikManager
func NewTraefikManager(client kubernetes.Interface) *TraefikManager {
	return &TraefikManager{
		client: client,
	}
}

// DeployGlobalTraefik deploys the global Traefik instance for all ALBs
func (m *TraefikManager) DeployGlobalTraefik(ctx context.Context) error {
	logging.Info("Deploying global Traefik instance for ALB support")

	// Ensure namespace exists
	if err := m.ensureNamespace(ctx); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Create ServiceAccount
	if err := m.createServiceAccount(ctx); err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}

	// Create ClusterRole
	if err := m.createClusterRole(ctx); err != nil {
		return fmt.Errorf("failed to create cluster role: %w", err)
	}

	// Create ClusterRoleBinding
	if err := m.createClusterRoleBinding(ctx); err != nil {
		return fmt.Errorf("failed to create cluster role binding: %w", err)
	}

	// Create ConfigMap
	if err := m.createConfigMap(ctx); err != nil {
		return fmt.Errorf("failed to create config map: %w", err)
	}

	// Create Deployment
	if err := m.createDeployment(ctx); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create Service
	if err := m.createService(ctx); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Wait for deployment to be ready
	if err := m.waitForDeployment(ctx); err != nil {
		return fmt.Errorf("deployment not ready: %w", err)
	}

	logging.Info("Global Traefik deployment completed successfully")
	return nil
}

func (m *TraefikManager) ensureNamespace(ctx context.Context) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kecs-system",
		},
	}

	_, err := m.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) createServiceAccount(ctx context.Context) error {
	sa := resources.GetTraefikServiceAccount("kecs-system")

	_, err := m.client.CoreV1().ServiceAccounts("kecs-system").Create(ctx, sa, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) createClusterRole(ctx context.Context) error {
	cr := resources.GetTraefikClusterRole()

	_, err := m.client.RbacV1().ClusterRoles().Create(ctx, cr, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) createClusterRoleBinding(ctx context.Context) error {
	crb := resources.GetTraefikClusterRoleBinding("kecs-system")

	_, err := m.client.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) createConfigMap(ctx context.Context) error {
	cm := resources.GetTraefikConfigMap("kecs-system")

	_, err := m.client.CoreV1().ConfigMaps("kecs-system").Create(ctx, cm, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) createDeployment(ctx context.Context) error {
	deployment := resources.GetTraefikDeployment("kecs-system")

	_, err := m.client.AppsV1().Deployments("kecs-system").Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) createService(ctx context.Context) error {
	service := resources.GetTraefikService("kecs-system")

	_, err := m.client.CoreV1().Services("kecs-system").Create(ctx, service, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) waitForDeployment(ctx context.Context) error {
	logging.Info("Waiting for Traefik deployment to be ready...")

	return wait.PollImmediate(2*time.Second, 60*time.Second, func() (bool, error) {
		deployment, err := m.client.AppsV1().Deployments("kecs-system").Get(ctx, "traefik", metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Check if deployment is ready
		if deployment.Status.ReadyReplicas >= 1 {
			logging.Info("Traefik deployment is ready",
				"readyReplicas", deployment.Status.ReadyReplicas,
				"replicas", deployment.Status.Replicas)
			return true, nil
		}

		logging.Debug("Waiting for Traefik deployment",
			"readyReplicas", deployment.Status.ReadyReplicas,
			"replicas", deployment.Status.Replicas)
		return false, nil
	})
}

// IsDeployed checks if the global Traefik is already deployed
func (m *TraefikManager) IsDeployed(ctx context.Context) bool {
	deployment, err := m.client.AppsV1().Deployments("kecs-system").Get(ctx, "traefik", metav1.GetOptions{})
	if err != nil {
		return false
	}
	return deployment.Status.ReadyReplicas >= 1
}
