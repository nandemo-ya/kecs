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
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// FluentBitManager manages FluentBit DaemonSets for log collection
type FluentBitManager struct {
	clientset         kubernetes.Interface
	localstackEndpoint string
	region            string
}

// NewFluentBitManager creates a new FluentBitManager
func NewFluentBitManager(clientset kubernetes.Interface, localstackEndpoint, region string) *FluentBitManager {
	return &FluentBitManager{
		clientset:         clientset,
		localstackEndpoint: localstackEndpoint,
		region:            region,
	}
}

// DeployFluentBitDaemonSet deploys or updates FluentBit DaemonSet in a namespace
func (m *FluentBitManager) DeployFluentBitDaemonSet(ctx context.Context, namespace string) error {
	// Create ConfigMap first
	if err := m.createOrUpdateConfigMap(ctx, namespace); err != nil {
		return fmt.Errorf("failed to create FluentBit ConfigMap: %w", err)
	}

	// Create or update DaemonSet
	if err := m.createOrUpdateDaemonSet(ctx, namespace); err != nil {
		return fmt.Errorf("failed to create FluentBit DaemonSet: %w", err)
	}

	logging.Info("FluentBit DaemonSet deployed", "namespace", namespace)
	return nil
}

// createOrUpdateConfigMap creates or updates the FluentBit ConfigMap
func (m *FluentBitManager) createOrUpdateConfigMap(ctx context.Context, namespace string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-bit-config",
			Namespace: namespace,
			Labels: map[string]string{
				"kecs.dev/component":  "fluent-bit",
				"kecs.dev/managed-by": "kecs",
			},
		},
		Data: map[string]string{
			"fluent-bit.conf": m.generateFluentBitConfig(namespace),
			"parsers.conf":    m.generateParsersConfig(),
			"parse-annotations.lua": m.generateLuaScript(),
		},
	}

	_, err := m.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMap.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = m.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update existing ConfigMap
	_, err = m.clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	return err
}

// generateFluentBitConfig generates the main FluentBit configuration
func (m *FluentBitManager) generateFluentBitConfig(namespace string) string {
	endpoint := m.localstackEndpoint
	if endpoint == "" {
		endpoint = "http://localstack.default.svc.cluster.local:4566"
	}

	return fmt.Sprintf(`[SERVICE]
    Flush        1
    Daemon       Off
    Log_Level    info
    Parsers_File parsers.conf

[INPUT]
    Name              tail
    Path              /host/var/log/containers/*.log
    Exclude_Path      /host/var/log/containers/*_kecs-system_*.log,/host/var/log/containers/*_kube-system_*.log
    Parser            docker
    Tag               kube.*
    Refresh_Interval  5
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    DB                /var/log/flb-kube.db
    DB.Sync           Off

[FILTER]
    Name                kubernetes
    Match               kube.*
    Kube_URL            https://kubernetes.default.svc:443
    Kube_CA_File        /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    Kube_Token_File     /var/run/secrets/kubernetes.io/serviceaccount/token
    Kube_Tag_Prefix     kube.var.log.containers.
    Merge_Log           On
    Keep_Log            Off
    K8S-Logging.Parser  On
    K8S-Logging.Exclude On
    Annotations         On

[FILTER]
    Name    lua
    Match   kube.*
    script  parse-annotations.lua
    call    parse_annotations

[OUTPUT]
    Name                cloudwatch_logs
    Match               kube.*
    region              %s
    endpoint            %s
    log_group_name      /ecs/default-logs
    log_stream_name     default-stream
    auto_create_group   On
    retry_limit         2
    net.keepalive       off
`, m.region, endpoint)
}

// generateParsersConfig generates the parsers configuration
func (m *FluentBitManager) generateParsersConfig() string {
	return `[PARSER]
    Name        docker
    Format      json
    Time_Key    time
    Time_Format %Y-%m-%dT%H:%M:%S.%LZ
    Time_Keep   On

[PARSER]
    Name        syslog
    Format      regex
    Regex       ^\<(?<pri>\d+)\>(?<time>[^ ]* {1,2}[^ ]* [^ ]*) (?<host>[^ ]*) (?<ident>[a-zA-Z0-9_\/\.\-]*)(?:\[(?<pid>[0-9]+)\])?(?:[^\:]*\:)? *(?<message>.*)$
    Time_Key    time
    Time_Format %b %d %H:%M:%S
`
}

// generateLuaScript generates the Lua script for parsing pod annotations
func (m *FluentBitManager) generateLuaScript() string {
	return `function parse_annotations(tag, timestamp, record)
    -- Get pod annotations from kubernetes metadata
    local k8s = record["kubernetes"]
    if not k8s then
        return 0, 0, 0
    end

    local annotations = k8s["annotations"]
    if not annotations then
        return 0, 0, 0
    end

    local container_name = k8s["container_name"]
    if not container_name then
        return 0, 0, 0
    end

    -- Look for container-specific log configuration
    local prefix = "kecs.dev/container-" .. container_name .. "-logs"
    local log_driver = annotations[prefix .. "-driver"]
    
    if log_driver ~= "awslogs" then
        -- Not configured for CloudWatch, skip
        return -1, 0, 0
    end

    -- Extract CloudWatch configuration
    local log_group = annotations[prefix .. "-group"]
    local log_stream = annotations[prefix .. "-stream"]
    local log_region = annotations[prefix .. "-region"]

    if log_group and log_stream then
        -- Set the log group and stream for CloudWatch output
        record["LOG_GROUP"] = log_group
        record["LOG_STREAM"] = log_stream
        
        if log_region then
            record["LOG_REGION"] = log_region
        end

        return 1, timestamp, record
    end

    -- No CloudWatch configuration found, skip this log
    return -1, 0, 0
end
`
}

// createOrUpdateDaemonSet creates or updates the FluentBit DaemonSet
func (m *FluentBitManager) createOrUpdateDaemonSet(ctx context.Context, namespace string) error {
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-bit",
			Namespace: namespace,
			Labels: map[string]string{
				"kecs.dev/component":  "fluent-bit",
				"kecs.dev/managed-by": "kecs",
				"app":                 "fluent-bit",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "fluent-bit",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                 "fluent-bit",
						"kecs.dev/component":  "fluent-bit",
						"kecs.dev/managed-by": "kecs",
					},
					Annotations: map[string]string{
						"kecs.dev/fluent-bit-version": "2.2.0",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "fluent-bit",
					Containers: []corev1.Container{
						{
							Name:  "fluent-bit",
							Image: "amazon/aws-for-fluent-bit:2.32.0",
							Env: []corev1.EnvVar{
								{
									Name:  "AWS_DEFAULT_REGION",
									Value: m.region,
								},
								{
									Name:  "AWS_ACCESS_KEY_ID",
									Value: "test", // LocalStack default
								},
								{
									Name:  "AWS_SECRET_ACCESS_KEY",
									Value: "test", // LocalStack default
								},
								{
									Name: "FLUENT_BIT_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("200Mi"),
									corev1.ResourceCPU:    resource.MustParse("200m"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("100Mi"),
									corev1.ResourceCPU:    resource.MustParse("100m"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "fluent-bit-config",
									MountPath: "/fluent-bit/etc/",
								},
								{
									Name:      "fluentbit-state",
									MountPath: "/var/log",
								},
								{
									Name:      "varlog",
									MountPath: "/host/var/log",
									ReadOnly:  true,
								},
								{
									Name:      "varlibdockercontainers",
									MountPath: "/var/lib/docker/containers",
									ReadOnly:  true,
								},
								{
									Name:      "runlogjournal",
									MountPath: "/run/log/journal",
									ReadOnly:  true,
								},
								{
									Name:      "dmesg",
									MountPath: "/var/log/dmesg",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "fluent-bit-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "fluent-bit-config",
									},
								},
							},
						},
						{
							Name: "fluentbit-state",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "varlog",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log",
								},
							},
						},
						{
							Name: "varlibdockercontainers",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/docker/containers",
								},
							},
						},
						{
							Name: "runlogjournal",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/log/journal",
								},
							},
						},
						{
							Name: "dmesg",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log/dmesg",
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      "node-role.kubernetes.io/master",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
				},
			},
		},
	}

	_, err := m.clientset.AppsV1().DaemonSets(namespace).Get(ctx, daemonSet.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create ServiceAccount first
			if err := m.createServiceAccount(ctx, namespace); err != nil {
				return fmt.Errorf("failed to create ServiceAccount: %w", err)
			}

			_, err = m.clientset.AppsV1().DaemonSets(namespace).Create(ctx, daemonSet, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update existing DaemonSet
	_, err = m.clientset.AppsV1().DaemonSets(namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	return err
}

// createServiceAccount creates the ServiceAccount for FluentBit
func (m *FluentBitManager) createServiceAccount(ctx context.Context, namespace string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-bit",
			Namespace: namespace,
			Labels: map[string]string{
				"kecs.dev/component":  "fluent-bit",
				"kecs.dev/managed-by": "kecs",
			},
		},
	}

	_, err := m.clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, sa.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = m.clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			// Create ClusterRole and ClusterRoleBinding for FluentBit
			return m.createRBACForFluentBit(ctx, namespace)
		}
		return err
	}

	return nil
}

// createRBACForFluentBit creates ClusterRole and ClusterRoleBinding for FluentBit
func (m *FluentBitManager) createRBACForFluentBit(ctx context.Context, namespace string) error {
	// Create ClusterRole
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fluent-bit-kecs",
			Labels: map[string]string{
				"kecs.dev/component":  "fluent-bit",
				"kecs.dev/managed-by": "kecs",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "pods", "nodes"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	_, err := m.clientset.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = m.clientset.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create ClusterRole: %w", err)
			}
		} else {
			return err
		}
	}

	// Create ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fluent-bit-kecs",
			Labels: map[string]string{
				"kecs.dev/component":  "fluent-bit",
				"kecs.dev/managed-by": "kecs",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "fluent-bit-kecs",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "fluent-bit",
				Namespace: namespace,
			},
		},
	}

	_, err = m.clientset.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = m.clientset.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
			}
		} else {
			return err
		}
	}

	return nil
}

// RemoveFluentBitDaemonSet removes the FluentBit DaemonSet from a namespace
func (m *FluentBitManager) RemoveFluentBitDaemonSet(ctx context.Context, namespace string) error {
	// Delete DaemonSet
	err := m.clientset.AppsV1().DaemonSets(namespace).Delete(ctx, "fluent-bit", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete FluentBit DaemonSet: %w", err)
	}

	// Delete ConfigMap
	err = m.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, "fluent-bit-config", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete FluentBit ConfigMap: %w", err)
	}

	// Delete ServiceAccount
	err = m.clientset.CoreV1().ServiceAccounts(namespace).Delete(ctx, "fluent-bit", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete FluentBit ServiceAccount: %w", err)
	}

	logging.Info("FluentBit DaemonSet removed", "namespace", namespace)
	return nil
}