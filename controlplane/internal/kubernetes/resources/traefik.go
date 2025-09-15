package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GetTraefikResources returns all Kubernetes resources needed for the global Traefik deployment
func GetTraefikResources(namespace string) []interface{} {
	return []interface{}{
		GetTraefikServiceAccount(namespace),
		GetTraefikClusterRole(),
		GetTraefikClusterRoleBinding(namespace),
		GetTraefikConfigMap(namespace),
		GetTraefikDeployment(namespace),
		GetTraefikService(namespace),
	}
}

// GetTraefikServiceAccount returns the ServiceAccount for Traefik
func GetTraefikServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "traefik",
			Namespace: namespace,
		},
	}
}

// GetTraefikClusterRole returns the ClusterRole for Traefik
func GetTraefikClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
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
					"ingressroutes",
					"ingressroutetcps",
					"ingressrouteudps",
					"middlewares",
					"middlewaretcps",
					"serverstransports",
					"tlsoptions",
					"tlsstores",
					"traefikservices",
				},
				Verbs: []string{"get", "list", "watch"},
			},
		},
	}
}

// GetTraefikClusterRoleBinding returns the ClusterRoleBinding for Traefik
func GetTraefikClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
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
				Namespace: namespace,
			},
		},
	}
}

// GetTraefikConfigMap returns the ConfigMap for Traefik configuration
func GetTraefikConfigMap(namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "traefik-config",
			Namespace: namespace,
		},
		Data: map[string]string{
			"traefik.yaml": `global:
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

accessLog: {}`,
		},
	}
}

// GetTraefikDeployment returns the Deployment for Traefik
func GetTraefikDeployment(namespace string) *appsv1.Deployment {
	replicas := int32(1)

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "traefik",
			Namespace: namespace,
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
}

// GetTraefikService returns the Service for Traefik with NodePort configuration
func GetTraefikService(namespace string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "traefik",
			Namespace: namespace,
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
}
