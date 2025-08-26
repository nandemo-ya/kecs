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

// Helper functions
func int32Ptr(i int32) *int32 {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}

const (
	// ControlPlane constants
	ControlPlaneNamespace      = "kecs-system"
	ControlPlaneName           = "kecs-controlplane"
	ControlPlaneServiceAccount = "kecs-controlplane"
	ControlPlaneConfigMap      = "kecs-config"
	ControlPlanePVC            = "kecs-data"
	ControlPlaneAPIService     = "kecs-api"
	ControlPlaneAdminService   = "kecs-admin"
	ControlPlaneService        = "kecs-controlplane" // Deprecated, kept for backward compatibility

	// Labels
	LabelManagedBy = "kecs.dev/managed"
	LabelComponent = "kecs.dev/component"
	LabelType      = "kecs.dev/type"
	LabelApp       = "app"
)

// ControlPlaneResources contains all resources needed for the control plane
type ControlPlaneResources struct {
	Namespace          *corev1.Namespace
	ServiceAccount     *corev1.ServiceAccount
	ClusterRole        *rbacv1.ClusterRole
	ClusterRoleBinding *rbacv1.ClusterRoleBinding
	ConfigMap          *corev1.ConfigMap
	PVC                *corev1.PersistentVolumeClaim
	Services           []*corev1.Service
	Deployment         *appsv1.Deployment
}

// ControlPlaneConfig contains configuration for control plane resources
type ControlPlaneConfig struct {
	// Image configuration
	Image           string
	ImagePullPolicy corev1.PullPolicy

	// Resource limits
	CPURequest    string
	MemoryRequest string
	CPULimit      string
	MemoryLimit   string

	// Storage
	StorageSize string

	// Ports
	APIPort   int32
	AdminPort int32

	// Features
	Debug    bool
	LogLevel string

	// Additional environment variables
	ExtraEnvVars []corev1.EnvVar
}

// DefaultControlPlaneConfig returns default configuration
func DefaultControlPlaneConfig() *ControlPlaneConfig {
	return &ControlPlaneConfig{
		Image:           "ghcr.io/nandemo-ya/kecs:latest",
		ImagePullPolicy: corev1.PullIfNotPresent,
		CPURequest:      "100m",
		MemoryRequest:   "128Mi",
		CPULimit:        "1000m",
		MemoryLimit:     "1Gi",
		StorageSize:     "10Gi",
		APIPort:         80,
		AdminPort:       5374,
		LogLevel:        "info",
	}
}

// CreateControlPlaneResources creates all resources for the control plane
func CreateControlPlaneResources(config *ControlPlaneConfig) *ControlPlaneResources {
	if config == nil {
		config = DefaultControlPlaneConfig()
	}

	return &ControlPlaneResources{
		Namespace:          createNamespace(),
		ServiceAccount:     createServiceAccount(),
		ClusterRole:        createClusterRole(),
		ClusterRoleBinding: createClusterRoleBinding(),
		ConfigMap:          createConfigMap(config),
		PVC:                createPVC(config),
		Services:           createServices(config),
		Deployment:         createDeployment(config),
	}
}

// createNamespace creates the kecs-system namespace
func createNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ControlPlaneNamespace,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelType:      "system",
			},
		},
	}
}

// createServiceAccount creates the service account for control plane
func createServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ControlPlaneServiceAccount,
			Namespace: ControlPlaneNamespace,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "controlplane",
			},
		},
	}
}

// createClusterRole creates the cluster role for control plane
func createClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: ControlPlaneName,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "controlplane",
			},
		},
		Rules: []rbacv1.PolicyRule{
			// Core resources
			{
				APIGroups: []string{""},
				Resources: []string{
					"namespaces", "pods", "services", "endpoints", "persistentvolumeclaims",
					"configmaps", "secrets", "serviceaccounts", "events", "nodes",
				},
				Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			// Apps resources
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "replicasets", "daemonsets", "statefulsets"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			// Batch resources
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs", "cronjobs"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			// RBAC resources
			{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"roles", "rolebindings", "clusterroles", "clusterrolebindings"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			// Networking resources
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"ingresses", "networkpolicies"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			// Pod logs
			{
				APIGroups: []string{""},
				Resources: []string{"pods/log", "pods/exec", "pods/attach", "pods/portforward"},
				Verbs:     []string{"get", "create"},
			},
			// Metrics
			{
				APIGroups: []string{"metrics.k8s.io"},
				Resources: []string{"pods", "nodes"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
}

// createClusterRoleBinding creates the cluster role binding
func createClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: ControlPlaneName,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "controlplane",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     ControlPlaneName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      ControlPlaneServiceAccount,
				Namespace: ControlPlaneNamespace,
			},
		},
	}
}

// createConfigMap creates the configuration ConfigMap
func createConfigMap(config *ControlPlaneConfig) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ControlPlaneConfigMap,
			Namespace: ControlPlaneNamespace,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "controlplane",
			},
		},
		Data: map[string]string{
			"config.yaml": fmt.Sprintf(`server:
  port: 5373
  adminPort: %d
  logLevel: %s

features:
  containerMode: false
  multiTenancy: true
  secretsManager: true
  serviceDiscovery: true
  elbv2: true
  traefik: true

database:
  type: duckdb
  path: /data/kecs.db
  inMemory: false

localstack:
  enabled: true
  services:
    - s3
    - iam
    - secretsmanager
    - ssm
    - logs
    - lambda
    - sts
  image: localstack/localstack
  version: latest

kubernetes:
  watchNamespaces: []
`, config.AdminPort, config.LogLevel),
		},
	}
}

// createPVC creates the persistent volume claim
func createPVC(config *ControlPlaneConfig) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ControlPlanePVC,
			Namespace: ControlPlaneNamespace,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "controlplane",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(config.StorageSize),
				},
			},
		},
	}
}

// createServices creates the services for control plane
func createServices(config *ControlPlaneConfig) []*corev1.Service {
	return []*corev1.Service{
		// API Service
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ControlPlaneAPIService,
				Namespace: ControlPlaneNamespace,
				Labels: map[string]string{
					LabelManagedBy: "true",
					LabelComponent: "controlplane",
					LabelType:      "api",
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					LabelApp: ControlPlaneName,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Port:       config.APIPort,
						TargetPort: intstr.FromInt(5373),
						Protocol:   corev1.ProtocolTCP,
						NodePort:   30080, // Fixed NodePort for external access
					},
				},
				Type: corev1.ServiceTypeNodePort,
			},
		},
		// Admin Service (NodePort for external access)
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ControlPlaneAdminService,
				Namespace: ControlPlaneNamespace,
				Labels: map[string]string{
					LabelManagedBy: "true",
					LabelComponent: "controlplane",
					LabelType:      "admin",
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					LabelApp: ControlPlaneName,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "admin",
						Port:       config.AdminPort,
						TargetPort: intstr.FromInt(int(config.AdminPort)),
						Protocol:   corev1.ProtocolTCP,
						NodePort:   30081, // Fixed NodePort for admin API
					},
				},
				Type: corev1.ServiceTypeNodePort,
			},
		},
		// Legacy Admin Service (ClusterIP for backward compatibility)
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ControlPlaneService,
				Namespace: ControlPlaneNamespace,
				Labels: map[string]string{
					LabelManagedBy: "true",
					LabelComponent: "controlplane",
					LabelType:      "admin-legacy",
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					LabelApp: ControlPlaneName,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "admin",
						Port:       config.AdminPort,
						TargetPort: intstr.FromInt(int(config.AdminPort)),
						Protocol:   corev1.ProtocolTCP,
					},
				},
				Type: corev1.ServiceTypeClusterIP,
			},
		},
	}
}

// createDeployment creates the control plane deployment
func createDeployment(config *ControlPlaneConfig) *appsv1.Deployment {
	replicas := int32(1)

	// Build environment variables
	envVars := []corev1.EnvVar{
		{
			Name:  "KECS_CONFIG_PATH",
			Value: "/etc/kecs/config.yaml",
		},
		{
			Name:  "KECS_DATA_DIR",
			Value: "/data",
		},
		{
			Name:  "KECS_LOG_LEVEL",
			Value: config.LogLevel,
		},
		{
			Name: "KECS_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: "KECS_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	}

	// Add debug environment variable if enabled
	if config.Debug {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "KECS_DEBUG",
			Value: "true",
		})
	}

	// Add extra environment variables
	envVars = append(envVars, config.ExtraEnvVars...)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ControlPlaneName,
			Namespace: ControlPlaneNamespace,
			Labels: map[string]string{
				LabelManagedBy: "true",
				LabelComponent: "controlplane",
				LabelApp:       ControlPlaneName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			ProgressDeadlineSeconds: int32Ptr(300), // 5 minutes should be enough
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					LabelApp: ControlPlaneName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						LabelApp:       ControlPlaneName,
						LabelManagedBy: "true",
						LabelComponent: "controlplane",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            ControlPlaneServiceAccount,
					TerminationGracePeriodSeconds: int64Ptr(30),
					DNSPolicy:                     corev1.DNSClusterFirstWithHostNet,
					// Add init container to verify network connectivity before main container starts
					InitContainers: []corev1.Container{
						{
							Name:    "wait-for-network",
							Image:   "busybox:1.36",
							Command: []string{"sh", "-c"},
							Args:    []string{"echo 'Checking network connectivity...'; nslookup kubernetes.default.svc.cluster.local || true; echo 'Network check complete'"},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "controlplane",
							Image:           config.Image,
							ImagePullPolicy: config.ImagePullPolicy,
							Command:         []string{"/controlplane"},
							Args:            []string{"server"},
							Env:             envVars,
							Ports: []corev1.ContainerPort{
								{
									Name:          "api",
									ContainerPort: 5373,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "admin",
									ContainerPort: int32(config.AdminPort),
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
							StartupProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/ready",
										Port: intstr.FromString("admin"),
									},
								},
								InitialDelaySeconds: 5,  // Start checking early
								PeriodSeconds:       2,  // Check frequently during startup
								FailureThreshold:    60, // Allow up to 2 minutes for startup (60 * 2s)
								SuccessThreshold:    1,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromString("admin"),
									},
								},
								InitialDelaySeconds: 0, // No delay needed with startup probe
								PeriodSeconds:       30,
								FailureThreshold:    3,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/ready",
										Port: intstr.FromString("admin"),
									},
								},
								InitialDelaySeconds: 0, // No delay needed with startup probe
								PeriodSeconds:       5, // Check more frequently for faster detection
								FailureThreshold:    3, // Reduced since startup probe handles initial startup
								SuccessThreshold:    1,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/kecs",
									ReadOnly:  true,
								},
								{
									Name:      "data",
									MountPath: "/data",
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
										Name: ControlPlaneConfigMap,
									},
								},
							},
						},
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: ControlPlanePVC,
								},
							},
						},
					},
				},
			},
		},
	}
}
