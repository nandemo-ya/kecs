package resources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// Traefik constants
	TraefikName           = "traefik"
	TraefikServiceAccount = "traefik"
	TraefikConfigMap      = "traefik-config"
	TraefikService        = "traefik"
	TraefikDashboard      = "traefik-dashboard"
)

// TraefikResources contains all resources needed for Traefik
type TraefikResources struct {
	ServiceAccount     *corev1.ServiceAccount
	ClusterRole        *rbacv1.ClusterRole
	ClusterRoleBinding *rbacv1.ClusterRoleBinding
	ConfigMap          *corev1.ConfigMap
	DynamicConfigMap   *corev1.ConfigMap  // Dynamic routing configuration
	Services           []*corev1.Service
	Deployment         *appsv1.Deployment
}

// TraefikConfig contains configuration for Traefik resources
type TraefikConfig struct {
	// Image configuration
	Image           string
	ImagePullPolicy corev1.PullPolicy
	
	// Resource limits
	CPURequest    string
	MemoryRequest string
	CPULimit      string
	MemoryLimit   string
	
	// Ports
	WebPort      int32  // HTTP port
	WebNodePort  int32  // NodePort for HTTP (optional)
	AWSPort      int32  // AWS API port (4566)
	AWSNodePort  int32  // NodePort for AWS API
	
	// Features
	Debug        bool
	LogLevel     string
	AccessLog    bool
	Metrics      bool
}

// DefaultTraefikConfig returns default configuration
func DefaultTraefikConfig() *TraefikConfig {
	return &TraefikConfig{
		Image:           "traefik:v3.2",
		ImagePullPolicy: corev1.PullIfNotPresent,
		CPURequest:      "100m",
		MemoryRequest:   "128Mi",
		CPULimit:        "500m",
		MemoryLimit:     "512Mi",
		WebPort:         80,
		WebNodePort:     30080,
		AWSPort:         4566,
		AWSNodePort:     30890,
		LogLevel:        "INFO",
		AccessLog:       true,
		Metrics:         false,
	}
}

// CreateTraefikResources creates all resources for Traefik
func CreateTraefikResources(config *TraefikConfig) *TraefikResources {
	if config == nil {
		config = DefaultTraefikConfig()
	}

	return &TraefikResources{
		ServiceAccount:     createTraefikServiceAccount(),
		ClusterRole:        createTraefikClusterRole(),
		ClusterRoleBinding: createTraefikClusterRoleBinding(),
		ConfigMap:          createTraefikConfigMap(config),
		DynamicConfigMap:   createTraefikDynamicConfigMap(),
		Services:           createTraefikServices(config),
		Deployment:         createTraefikDeployment(config),
	}
}

// createTraefikServiceAccount creates the service account for Traefik
func createTraefikServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TraefikServiceAccount,
			Namespace: ControlPlaneNamespace,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "gateway",
			},
		},
	}
}

// createTraefikClusterRole creates the cluster role for Traefik
func createTraefikClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: TraefikName,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "gateway",
			},
		},
		Rules: []rbacv1.PolicyRule{
			// Core resources
			{
				APIGroups: []string{""},
				Resources: []string{"services", "endpoints", "secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			// Extensions
			{
				APIGroups: []string{"extensions", "networking.k8s.io"},
				Resources: []string{"ingresses", "ingressclasses"},
				Verbs:     []string{"get", "list", "watch"},
			},
			// Traefik CRDs - removed as we use file-based configuration
		},
	}
}

// createTraefikClusterRoleBinding creates the cluster role binding
func createTraefikClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: TraefikName,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "gateway",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     TraefikName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      TraefikServiceAccount,
				Namespace: ControlPlaneNamespace,
			},
		},
	}
}

// createTraefikConfigMap creates the configuration ConfigMap
func createTraefikConfigMap(config *TraefikConfig) *corev1.ConfigMap {
	accessLogConfig := ""
	if config.AccessLog {
		accessLogConfig = `
accessLog:
  format: json
  fields:
    defaultMode: keep
    headers:
      defaultMode: keep
      names:
        X-Amz-Target: keep
        Authorization: redact`
	}

	// Metrics are disabled for security
	metricsConfig := ""

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TraefikConfigMap,
			Namespace: ControlPlaneNamespace,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "gateway",
			},
		},
		Data: map[string]string{
			"traefik.yaml": fmt.Sprintf(`api:
  dashboard: false
  debug: %v

entryPoints:
  web:
    address: ":80"
  aws:
    address: ":%d"

providers:
  file:
    filename: /dynamic/dynamic.yaml
    watch: true
  kubernetesIngress:
    allowExternalNameServices: true

log:
  level: %s
  format: json
%s
%s

ping:
  entryPoint: traefik
`, config.Debug, config.AWSPort, config.LogLevel, accessLogConfig, metricsConfig),
		},
	}
}

// createTraefikServices creates the services for Traefik
func createTraefikServices(config *TraefikConfig) []*corev1.Service {
	services := []*corev1.Service{
		// Main Traefik service
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      TraefikService,
				Namespace: ControlPlaneNamespace,
				Labels: map[string]string{
					LabelManagedBy: "true",
					LabelComponent: "gateway",
					LabelApp:       TraefikName,
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					LabelApp: TraefikName,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "web",
						Port:       config.WebPort,
						TargetPort: intstr.FromString("web"),
						Protocol:   corev1.ProtocolTCP,
					},
					{
						Name:       "aws",
						Port:       config.AWSPort,
						TargetPort: intstr.FromString("aws"),
						Protocol:   corev1.ProtocolTCP,
						NodePort:   config.AWSNodePort,
					},
				},
				Type: corev1.ServiceTypeNodePort,
			},
		},
		// Dashboard service
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      TraefikDashboard,
				Namespace: ControlPlaneNamespace,
				Labels: map[string]string{
					LabelManagedBy: "true",
					LabelComponent: "gateway",
					LabelApp:       TraefikName,
					LabelType:      "dashboard",
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					LabelApp: TraefikName,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "admin",
						Port:       8080,
						TargetPort: intstr.FromString("admin"),
						Protocol:   corev1.ProtocolTCP,
					},
				},
				Type: corev1.ServiceTypeClusterIP,
			},
		},
	}

	// Add metrics service if enabled
	if config.Metrics {
		services = append(services, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-metrics", TraefikName),
				Namespace: ControlPlaneNamespace,
				Labels: map[string]string{
					LabelManagedBy: "true",
					LabelComponent: "gateway",
					LabelApp:       TraefikName,
					LabelType:      "metrics",
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					LabelApp: TraefikName,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "metrics",
						Port:       8082,
						TargetPort: intstr.FromString("metrics"),
						Protocol:   corev1.ProtocolTCP,
					},
				},
				Type: corev1.ServiceTypeClusterIP,
			},
		})
	}

	return services
}

// createTraefikDeployment creates the Traefik deployment
func createTraefikDeployment(config *TraefikConfig) *appsv1.Deployment {
	replicas := int32(1)
	runAsUser := int64(65532)
	runAsNonRoot := true
	readOnlyRootFilesystem := true
	
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TraefikName,
			Namespace: ControlPlaneNamespace,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "gateway",
				LabelApp:       TraefikName,
				"kecs.dev/version": "v2",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					LabelApp: TraefikName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						LabelApp:       TraefikName,
						LabelManagedBy: "true",
						LabelComponent: "gateway",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: TraefikServiceAccount,
					Containers: []corev1.Container{
						{
							Name:            TraefikName,
							Image:           config.Image,
							ImagePullPolicy: config.ImagePullPolicy,
							Args:            []string{"--configfile=/config/traefik.yaml"},
							Ports: []corev1.ContainerPort{
								{
									Name:          "web",
									ContainerPort: config.WebPort,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "aws",
									ContainerPort: config.AWSPort,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "admin",
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "metrics",
									ContainerPort: 8082,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(config.CPURequest),
									corev1.ResourceMemory: resource.MustParse(config.MemoryRequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(config.CPULimit),
									corev1.ResourceMemory: resource.MustParse(config.MemoryLimit),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/ping",
										Port: intstr.FromString("admin"),
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       30,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/ping",
										Port: intstr.FromString("admin"),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
									Add:  []corev1.Capability{"NET_BIND_SERVICE"},
								},
								RunAsNonRoot:           &runAsNonRoot,
								RunAsUser:              &runAsUser,
								ReadOnlyRootFilesystem: &readOnlyRootFilesystem,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/config",
									ReadOnly:  true,
								},
								{
									Name:      "dynamic",
									MountPath: "/dynamic",
									ReadOnly:  true,
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
										Name: TraefikConfigMap,
									},
								},
							},
						},
						{
							Name: "dynamic",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "traefik-dynamic-config",
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

// createTraefikDynamicConfigMap creates the dynamic configuration for routing
func createTraefikDynamicConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "traefik-dynamic-config",
			Namespace: ControlPlaneNamespace,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "gateway",
			},
		},
		Data: map[string]string{
			"dynamic.yaml": `http:
  routers:
    # ECS API routing
    ecs-api:
      entryPoints:
        - aws
      rule: "PathPrefix(` + "`/v1`" + `)"
      service: ecs-api
      priority: 10
    # LocalStack routing (catch-all)
    localstack:
      entryPoints:
        - aws
      rule: "PathPrefix(` + "`/`" + `)"
      service: localstack
      priority: 1
  services:
    ecs-api:
      loadBalancer:
        servers:
          - url: "http://kecs-api:80"
    localstack:
      loadBalancer:
        servers:
          - url: "http://localstack:4566"`,
		},
	}
}