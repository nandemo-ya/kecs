package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type NamespaceManager struct {
	clientset *kubernetes.Clientset
}

func NewNamespaceManager(clientset *kubernetes.Clientset) *NamespaceManager {
	return &NamespaceManager{
		clientset: clientset,
	}
}

func (n *NamespaceManager) CreateNamespace(ctx context.Context, clusterName, region string) error {
	namespaceName := fmt.Sprintf("%s-%s", clusterName, region)

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				"kecs.dev/managed": "true",
				"kecs.dev/cluster": clusterName,
				"kecs.dev/region":  region,
				"kecs.dev/type":    "ecs-cluster",
			},
			Annotations: map[string]string{
				"kecs.dev/created-by": "kecs-controlplane",
			},
		},
	}

	_, err := n.clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	_, err = n.clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", namespaceName, err)
	}

	return nil
}

func (n *NamespaceManager) DeleteNamespace(ctx context.Context, clusterName, region string) error {
	namespaceName := fmt.Sprintf("%s-%s", clusterName, region)

	err := n.clientset.CoreV1().Namespaces().Delete(ctx, namespaceName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete namespace %s: %w", namespaceName, err)
	}

	return nil
}

func (n *NamespaceManager) GetNamespace(ctx context.Context, clusterName, region string) (*corev1.Namespace, error) {
	namespaceName := fmt.Sprintf("%s-%s", clusterName, region)

	namespace, err := n.clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %s: %w", namespaceName, err)
	}

	return namespace, nil
}

func (n *NamespaceManager) ListNamespacesForCluster(ctx context.Context, clusterName string) ([]corev1.Namespace, error) {
	labelSelector := fmt.Sprintf("kecs.dev/cluster=%s", clusterName)

	namespaceList, err := n.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces for cluster %s: %w", clusterName, err)
	}

	return namespaceList.Items, nil
}
