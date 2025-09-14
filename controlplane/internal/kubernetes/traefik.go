package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
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
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "traefik",
			Namespace: "kecs-system",
		},
	}

	_, err := m.client.CoreV1().ServiceAccounts("kecs-system").Create(ctx, sa, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) createClusterRole(ctx context.Context) error {
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "traefik",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services", "endpoints", "secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"extensions", "networking.k8s.io"},
				Resources: []string{"ingresses", "ingressclasses"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"extensions", "networking.k8s.io"},
				Resources: []string{"ingresses/status"},
				Verbs:     []string{"update"},
			},
			{
				APIGroups: []string{"traefik.io", "traefik.containo.us"},
				Resources: []string{
					"ingressroutes", "ingressroutetcps", "ingressrouteudps",
					"middlewares", "middlewaretcps", "serverstransports",
					"tlsoptions", "tlsstores", "traefikservices",
				},
				Verbs: []string{"get", "list", "watch"},
			},
		},
	}

	_, err := m.client.RbacV1().ClusterRoles().Create(ctx, cr, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) createClusterRoleBinding(ctx context.Context) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "traefik",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "traefik",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "traefik",
				Namespace: "kecs-system",
			},
		},
	}

	_, err := m.client.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) createConfigMap(ctx context.Context) error {
	traefikConfig := `global:
  checkNewVersion: false
  sendAnonymousUsage: false

api:
  dashboard: true
  debug: true

entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

providers:
  kubernetescrd:
    allowCrossNamespace: true
  kubernetesingress:
    allowEmptyServices: true

log:
  level: INFO

accessLog: {}`

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "traefik-config",
			Namespace: "kecs-system",
		},
		Data: map[string]string{
			"traefik.yaml": traefikConfig,
		},
	}

	_, err := m.client.CoreV1().ConfigMaps("kecs-system").Create(ctx, cm, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) createDeployment(ctx context.Context) error {
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "traefik",
			Namespace: "kecs-system",
			Labels: map[string]string{
				"app":               "traefik",
				"kecs.io/component": "elbv2-proxy",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "traefik",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":               "traefik",
						"kecs.io/component": "elbv2-proxy",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "traefik",
					Containers: []corev1.Container{
						{
							Name:  "traefik",
							Image: "traefik:v3.0",
							Args: []string{
								"--configfile=/config/traefik.yaml",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "web",
									ContainerPort: 80,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "websecure",
									ContainerPort: 443,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "admin",
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/config",
									ReadOnly:  true,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "traefik-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := m.client.AppsV1().Deployments("kecs-system").Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *TraefikManager) createService(ctx context.Context) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "traefik",
			Namespace: "kecs-system",
			Labels: map[string]string{
				"app":               "traefik",
				"kecs.io/component": "elbv2-proxy",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{
				"app": "traefik",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       80,
					TargetPort: intstr.FromInt(80),
					Protocol:   corev1.ProtocolTCP,
					NodePort:   30880, // Fixed NodePort for external access
				},
				{
					Name:       "websecure",
					Port:       443,
					TargetPort: intstr.FromInt(443),
					Protocol:   corev1.ProtocolTCP,
					NodePort:   30443, // Fixed NodePort for HTTPS
				},
				{
					Name:       "admin",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
					NodePort:   30808, // Fixed NodePort for Traefik dashboard
				},
			},
		},
	}

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
