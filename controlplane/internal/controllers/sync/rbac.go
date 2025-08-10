package sync

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// RBACManager manages RBAC resources for cross-namespace secret access
type RBACManager struct {
	kubeClient kubernetes.Interface
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(kubeClient kubernetes.Interface) *RBACManager {
	return &RBACManager{
		kubeClient: kubeClient,
	}
}

// SetupNamespaceRBAC sets up RBAC resources to allow a namespace to read secrets from kecs-system
func (m *RBACManager) SetupNamespaceRBAC(ctx context.Context, namespace string) error {
	// Create ServiceAccount in the user namespace
	saName := fmt.Sprintf("kecs-secret-reader-%s", namespace)
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
			Labels: map[string]string{
				"kecs.io/managed-by": "kecs",
				"kecs.io/purpose":    "secret-reader",
			},
		},
	}

	_, err := m.kubeClient.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create service account: %w", err)
	}

	// Create ClusterRole for reading secrets and configmaps in kecs-system
	clusterRoleName := "kecs-system-secret-reader"
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName,
			Labels: map[string]string{
				"kecs.io/managed-by": "kecs",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets", "configmaps"},
				Verbs:         []string{"get", "list", "watch"},
				ResourceNames: []string{}, // Allow access to all secrets/configmaps
			},
		},
	}

	_, err = m.kubeClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster role: %w", err)
	}

	// Create RoleBinding in kecs-system namespace
	roleBindingName := fmt.Sprintf("kecs-secret-reader-%s", namespace)
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: "kecs-system",
			Labels: map[string]string{
				"kecs.io/managed-by":   "kecs",
				"kecs.io/for-namespace": namespace,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
	}

	_, err = m.kubeClient.RbacV1().RoleBindings("kecs-system").Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create role binding: %w", err)
	}

	logging.Info("RBAC setup completed for namespace", "namespace", namespace)
	return nil
}

// CleanupNamespaceRBAC removes RBAC resources for a namespace
func (m *RBACManager) CleanupNamespaceRBAC(ctx context.Context, namespace string) error {
	// Delete RoleBinding in kecs-system
	roleBindingName := fmt.Sprintf("kecs-secret-reader-%s", namespace)
	err := m.kubeClient.RbacV1().RoleBindings("kecs-system").Delete(ctx, roleBindingName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		logging.Error("Failed to delete role binding", "name", roleBindingName, "error", err)
	}

	// Delete ServiceAccount in user namespace
	saName := fmt.Sprintf("kecs-secret-reader-%s", namespace)
	err = m.kubeClient.CoreV1().ServiceAccounts(namespace).Delete(ctx, saName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		logging.Error("Failed to delete service account", "name", saName, "namespace", namespace, "error", err)
	}

	logging.Info("RBAC cleanup completed for namespace", "namespace", namespace)
	return nil
}