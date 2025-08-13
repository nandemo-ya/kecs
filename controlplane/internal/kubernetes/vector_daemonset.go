package kubernetes

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

const (
	vectorNamespace      = "kecs-system"
	vectorDaemonSet      = "vector"
	vectorServiceAccount = "vector"
	vectorConfigMap      = "vector-config"
	vectorImage          = "timberio/vector:0.34.0-alpine"
)

// EnsureVectorDaemonSet ensures Vector DaemonSet is deployed in kecs-system namespace
func EnsureVectorDaemonSet(ctx context.Context, clientset kubernetes.Interface, localstackEndpoint string, region string) error {
	logging.Info("Ensuring Vector DaemonSet in kecs-system namespace",
		"localstackEndpoint", localstackEndpoint,
		"region", region)

	// Create ServiceAccount
	if err := createVectorServiceAccount(ctx, clientset); err != nil {
		return fmt.Errorf("failed to create Vector ServiceAccount: %w", err)
	}

	// Create ClusterRole
	if err := createVectorClusterRole(ctx, clientset); err != nil {
		return fmt.Errorf("failed to create Vector ClusterRole: %w", err)
	}

	// Create ClusterRoleBinding
	if err := createVectorClusterRoleBinding(ctx, clientset); err != nil {
		return fmt.Errorf("failed to create Vector ClusterRoleBinding: %w", err)
	}

	// Create ConfigMap
	if err := createVectorConfigMap(ctx, clientset, localstackEndpoint, region); err != nil {
		return fmt.Errorf("failed to create Vector ConfigMap: %w", err)
	}

	// Create DaemonSet
	if err := createVectorDaemonSet(ctx, clientset); err != nil {
		return fmt.Errorf("failed to create Vector DaemonSet: %w", err)
	}

	logging.Info("Vector DaemonSet successfully deployed")
	return nil
}

func createVectorServiceAccount(ctx context.Context, clientset kubernetes.Interface) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vectorServiceAccount,
			Namespace: vectorNamespace,
		},
	}

	_, err := clientset.CoreV1().ServiceAccounts(vectorNamespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func createVectorClusterRole(ctx context.Context, clientset kubernetes.Interface) error {
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: vectorServiceAccount,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "pods", "nodes"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	_, err := clientset.RbacV1().ClusterRoles().Create(ctx, cr, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func createVectorClusterRoleBinding(ctx context.Context, clientset kubernetes.Interface) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: vectorServiceAccount,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     vectorServiceAccount,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      vectorServiceAccount,
				Namespace: vectorNamespace,
			},
		},
	}

	_, err := clientset.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func createVectorConfigMap(ctx context.Context, clientset kubernetes.Interface, localstackEndpoint string, region string) error {
	if localstackEndpoint == "" {
		localstackEndpoint = "http://localstack.kecs-system.svc.cluster.local:4566"
	}
	if region == "" {
		region = "us-east-1"
	}

	vectorConfig := fmt.Sprintf(`# Vector configuration for CloudWatch Logs integration

# Input: Collect all container logs
[sources.kubernetes_logs]
type = "kubernetes_logs"

# Transform: Filter out system namespaces
[transforms.filter_namespace]
type = "filter"
inputs = ["kubernetes_logs"]
condition = '''
!includes(["kecs-system", "kube-system"], .kubernetes.pod_namespace)
'''

# Transform: Process logs for CloudWatch
[transforms.process_logs]
type = "remap"
inputs = ["filter_namespace"]
source = '''
# Extract metadata
.namespace = string!(.kubernetes.pod_namespace)
.pod_name = string!(.kubernetes.pod_name)
.container_name = string!(.kubernetes.container_name)

# Check for CloudWatch log configuration in annotations
.annotations = .kubernetes.pod_annotations

# Default values
.log_group = "/kecs/default"
.log_stream = .namespace + "/" + .pod_name + "/" + .container_name

# Check for container-specific log configuration
container_prefix = "kecs.dev/container-" + .container_name + "-logs-"

# Extract log configuration from annotations
if exists(.annotations) {
  # Check if CloudWatch is enabled
  if exists(.annotations."kecs.dev/cloudwatch-logs-enabled") {
    if .annotations."kecs.dev/cloudwatch-logs-enabled" == "true" {
      # Get log group
      group_key = container_prefix + "group"
      if exists(.annotations[group_key]) {
        .log_group = string!(.annotations[group_key])
      }
      
      # Get stream prefix
      stream_key = container_prefix + "stream-prefix"
      if exists(.annotations[stream_key]) {
        stream_prefix = string!(.annotations[stream_key])
        .log_stream = stream_prefix + "/" + .pod_name
      }
      
      # Mark as CloudWatch enabled
      .cloudwatch_enabled = true
    } else {
      .cloudwatch_enabled = false
    }
  } else {
    .cloudwatch_enabled = false
  }
} else {
  .cloudwatch_enabled = false
}
'''

# Route: Only send CloudWatch-enabled logs
[transforms.route_cloudwatch]
type = "filter"
inputs = ["process_logs"]
condition = '.cloudwatch_enabled == true'

# Output: Send to CloudWatch Logs via LocalStack
[sinks.cloudwatch]
type = "aws_cloudwatch_logs"
inputs = ["route_cloudwatch"]
endpoint = "%s"
region = "%s"
group_name = "{{ log_group }}"
stream_name = "{{ log_stream }}"
create_missing_group = true
create_missing_stream = true
encoding.codec = "text"

# Optional: Console output for debugging
[sinks.console_debug]
type = "console"
inputs = ["route_cloudwatch"]
encoding.codec = "json"
target = "stdout"

# Sink configuration for debugging (can be removed in production)
[sinks.console_debug.when]
type = "vrl"
source = '''
# Only log every 100th message to reduce noise
random_float() < 0.01
'''
`, localstackEndpoint, region)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vectorConfigMap,
			Namespace: vectorNamespace,
		},
		Data: map[string]string{
			"vector.toml": vectorConfig,
		},
	}

	_, err := clientset.CoreV1().ConfigMaps(vectorNamespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func createVectorDaemonSet(ctx context.Context, clientset kubernetes.Interface) error {
	replicas := int32(1)
	privileged := false

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vectorDaemonSet,
			Namespace: vectorNamespace,
			Labels: map[string]string{
				"app":                "vector",
				"kecs.dev/component": "logging",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "vector",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                "vector",
						"kecs.dev/component": "logging",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: vectorServiceAccount,
					Containers: []corev1.Container{
						{
							Name:  "vector",
							Image: vectorImage,
							Env: []corev1.EnvVar{
								{
									Name:  "VECTOR_CONFIG_DIR",
									Value: "/etc/vector",
								},
								{
									Name:  "VECTOR_LOG",
									Value: "info",
								},
								{
									Name: "VECTOR_SELF_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "AWS_ACCESS_KEY_ID",
									Value: "test",
								},
								{
									Name:  "AWS_SECRET_ACCESS_KEY",
									Value: "test",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/vector",
									ReadOnly:  true,
								},
								{
									Name:      "var-log",
									MountPath: "/var/log",
									ReadOnly:  true,
								},
								{
									Name:      "var-lib-docker-containers",
									MountPath: "/var/lib/docker/containers",
									ReadOnly:  true,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
									corev1.ResourceCPU:    resource.MustParse("500m"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("200m"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: vectorConfigMap,
									},
								},
							},
						},
						{
							Name: "var-log",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log",
								},
							},
						},
						{
							Name: "var-lib-docker-containers",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/docker/containers",
								},
							},
						},
					},
					HostNetwork: false,
					DNSPolicy:   corev1.DNSClusterFirst,
				},
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: replicas,
					},
				},
			},
		},
	}

	_, err := clientset.AppsV1().DaemonSets(vectorNamespace).Create(ctx, ds, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}
