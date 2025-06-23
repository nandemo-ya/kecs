package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// Helper functions for simple container deployment

// generateSimpleTaskID generates a simple task ID
func generateSimpleTaskID() string {
	return fmt.Sprintf("task-%d", rand.Intn(1000000))
}

// createBasicPod creates a basic Kubernetes pod from task definition
func (s *Server) createBasicPod(taskDef *storage.TaskDefinition, cluster *storage.Cluster, taskID string) (*corev1.Pod, error) {
	// Parse container definitions
	var containerDefs []map[string]interface{}
	if err := json.Unmarshal([]byte(taskDef.ContainerDefinitions), &containerDefs); err != nil {
		return nil, fmt.Errorf("failed to parse container definitions: %w", err)
	}

	if len(containerDefs) == 0 {
		return nil, fmt.Errorf("no container definitions found")
	}

	// Get the first container definition
	firstContainer := containerDefs[0]

	// Extract basic container info
	name, _ := firstContainer["name"].(string)
	image, _ := firstContainer["image"].(string)

	if name == "" || image == "" {
		return nil, fmt.Errorf("missing required container name or image")
	}

	// Create pod specification
	namespace := fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)
	podName := fmt.Sprintf("ecs-task-%s", taskID)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				"kecs.dev/cluster":     cluster.Name,
				"kecs.dev/task-id":     taskID,
				"kecs.dev/task-family": taskDef.Family,
				"kecs.dev/managed-by":  "kecs",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: image,
				},
			},
		},
	}

	// Create the pod in Kubernetes
	kubeClient, err := s.getKubeClient(cluster.K8sClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	createdPod, err := kubeClient.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create pod in kubernetes: %w", err)
	}

	log.Printf("Successfully created pod %s in namespace %s", createdPod.Name, createdPod.Namespace)
	return createdPod, nil
}

// getKubeClient gets a Kubernetes client for the specified k3d cluster
func (s *Server) getKubeClient(k8sClusterName string) (kubernetes.Interface, error) {
	if s.clusterManager == nil {
		return nil, fmt.Errorf("cluster manager not available")
	}

	return s.clusterManager.GetKubeClient(k8sClusterName)
}
