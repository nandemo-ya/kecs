package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnsureNamespace creates a namespace if it doesn't exist
func EnsureNamespace(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	// Check if namespace exists
	_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		// Namespace already exists
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check namespace existence: %w", err)
	}

	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"managed-by": "kecs",
				"purpose":    "ecs-cluster",
			},
		},
	}

	_, err = clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	return nil
}