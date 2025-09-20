package converters

import (
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

// DeploymentInfo represents the deployment information for a service
type DeploymentInfo struct {
	Name      string
	Namespace string
	Replicas  int32
	Labels    map[string]string
}

// ConvertServiceToDeployment converts a service and task definition to deployment info
func ConvertServiceToDeployment(service *storage.Service, taskDef *storage.TaskDefinition, namespace string) *DeploymentInfo {
	labels := map[string]string{
		"app":         service.ServiceName,
		"ecs-service": service.ServiceName,
		"ecs-cluster": service.ClusterARN,
	}

	// Add target group labels if LoadBalancers are configured
	if service.LoadBalancers != "" {
		var loadBalancers []types.LoadBalancer
		if err := json.Unmarshal([]byte(service.LoadBalancers), &loadBalancers); err != nil {
			logging.Warn("Failed to parse LoadBalancers JSON for service %s: %v", service.ServiceName, err)
		} else {
			var targetGroupNames []string
			for _, lb := range loadBalancers {
				if lb.TargetGroupArn != nil && *lb.TargetGroupArn != "" {
					// Extract target group name from ARN
					// ARN format: arn:aws:elasticloadbalancing:region:account:targetgroup/name/id
					parts := strings.Split(*lb.TargetGroupArn, "/")
					if len(parts) >= 2 {
						targetGroupName := parts[1]
						targetGroupNames = append(targetGroupNames, targetGroupName)
					}
				}
			}
			// If multiple target groups, use comma-separated list
			if len(targetGroupNames) > 0 {
				labels["kecs.io/elbv2-target-group-names"] = strings.Join(targetGroupNames, ",")
				// Also keep the first one as the primary for backward compatibility
				labels["kecs.io/elbv2-target-group-name"] = targetGroupNames[0]
			}
		}
	}

	return &DeploymentInfo{
		Name:      service.ServiceName,
		Namespace: namespace,
		Replicas:  int32(service.DesiredCount),
		Labels:    labels,
	}
}

// ConvertDeploymentToK8s converts an internal deployment definition to a Kubernetes deployment
func ConvertDeploymentToK8s(deployment *DeploymentInfo, containerDefs []types.ContainerDefinition) *appsv1.Deployment {
	// Convert container definitions to pod spec containers
	containers := make([]corev1.Container, 0, len(containerDefs))
	for _, containerDef := range containerDefs {
		container := corev1.Container{
			Name:  ptr.Deref(containerDef.Name, ""),
			Image: ptr.Deref(containerDef.Image, ""),
			Env:   convertEnvironmentVariables(containerDef.Environment),
		}

		// Set resource requirements if specified
		if containerDef.Cpu != nil && *containerDef.Cpu != 0 || containerDef.Memory != nil && *containerDef.Memory != 0 {
			container.Resources = corev1.ResourceRequirements{}
			if containerDef.Memory != nil && *containerDef.Memory != 0 {
				container.Resources.Requests = corev1.ResourceList{
					corev1.ResourceMemory: parseMemoryMiB(*containerDef.Memory),
				}
				container.Resources.Limits = corev1.ResourceList{
					corev1.ResourceMemory: parseMemoryMiB(*containerDef.Memory),
				}
			}
			if containerDef.Cpu != nil && *containerDef.Cpu != 0 {
				if container.Resources.Requests == nil {
					container.Resources.Requests = corev1.ResourceList{}
				}
				if container.Resources.Limits == nil {
					container.Resources.Limits = corev1.ResourceList{}
				}
				container.Resources.Requests[corev1.ResourceCPU] = parseCPUUnits(*containerDef.Cpu)
				container.Resources.Limits[corev1.ResourceCPU] = parseCPUUnits(*containerDef.Cpu)
			}
		}

		// Add port mappings
		for _, portMapping := range containerDef.PortMappings {
			if portMapping.ContainerPort != nil {
				container.Ports = append(container.Ports, corev1.ContainerPort{
					ContainerPort: int32(*portMapping.ContainerPort),
					Protocol:      parseProtocol(ptr.Deref(portMapping.Protocol, "tcp")),
				})
			}
		}

		// Add health check if defined
		if containerDef.HealthCheck != nil {
			container.LivenessProbe = convertHealthCheck(containerDef.HealthCheck)
			container.ReadinessProbe = convertHealthCheck(containerDef.HealthCheck)
		}

		// Set working directory if specified
		if containerDef.WorkingDirectory != nil && *containerDef.WorkingDirectory != "" {
			container.WorkingDir = *containerDef.WorkingDirectory
		}

		// Add commands if specified
		if len(containerDef.Command) > 0 {
			container.Args = containerDef.Command
		}
		if len(containerDef.EntryPoint) > 0 {
			container.Command = containerDef.EntryPoint
			if len(containerDef.Command) > 0 {
				container.Args = containerDef.Command
			}
		}

		containers = append(containers, container)
	}

	// Create deployment
	k8sDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Labels:    deployment.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(deployment.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deployment.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": deployment.Name,
					},
					Annotations: make(map[string]string),
				},
				Spec: corev1.PodSpec{
					Containers: containers,
					// Add service account if IAM role is specified
					ServiceAccountName: "default", // TODO: Create service account from IAM role
				},
			},
		},
	}

	// Add all deployment labels to pod template
	for k, v := range deployment.Labels {
		k8sDeployment.Spec.Template.ObjectMeta.Labels[k] = v
	}

	// Add secret annotations for pod template
	addSecretAnnotationsToPodTemplate(&k8sDeployment.Spec.Template, containerDefs)

	return k8sDeployment
}

// convertHealthCheck converts ECS health check to Kubernetes probe
func convertHealthCheck(healthCheck *types.HealthCheck) *corev1.Probe {
	if healthCheck == nil || len(healthCheck.Command) == 0 {
		return nil
	}

	probe := &corev1.Probe{
		InitialDelaySeconds: int32(ptr.Deref(healthCheck.StartPeriod, 0)),
		PeriodSeconds:       int32(ptr.Deref(healthCheck.Interval, 30)),
		TimeoutSeconds:      int32(ptr.Deref(healthCheck.Timeout, 5)),
		FailureThreshold:    int32(ptr.Deref(healthCheck.Retries, 3)),
	}

	// Convert command
	if len(healthCheck.Command) > 0 {
		probe.Exec = &corev1.ExecAction{
			Command: healthCheck.Command,
		}
	}

	return probe
}

// parseProtocol converts ECS protocol to Kubernetes protocol
func parseProtocol(protocol string) corev1.Protocol {
	switch protocol {
	case "udp", "UDP":
		return corev1.ProtocolUDP
	case "sctp", "SCTP":
		return corev1.ProtocolSCTP
	default:
		return corev1.ProtocolTCP
	}
}

// convertEnvironmentVariables converts ECS environment variables to Kubernetes env vars
func convertEnvironmentVariables(envVars []types.KeyValuePair) []corev1.EnvVar {
	var result []corev1.EnvVar
	for _, env := range envVars {
		if env.Name != nil && env.Value != nil {
			result = append(result, corev1.EnvVar{
				Name:  *env.Name,
				Value: *env.Value,
			})
		}
	}
	return result
}

// parseMemoryMiB converts memory in MiB to Kubernetes resource quantity
func parseMemoryMiB(memoryMiB int) resource.Quantity {
	return *resource.NewQuantity(int64(memoryMiB)*1024*1024, resource.BinarySI)
}

// parseCPUUnits converts ECS CPU units to Kubernetes resource quantity
func parseCPUUnits(cpuUnits int) resource.Quantity {
	// ECS CPU units: 1024 = 1 vCPU
	// Convert to millicores (1000m = 1 CPU)
	millicores := (cpuUnits * 1000) / 1024
	return *resource.NewMilliQuantity(int64(millicores), resource.DecimalSI)
}

// addSecretAnnotationsToPodTemplate adds annotations for secrets used by the containers
func addSecretAnnotationsToPodTemplate(podTemplate *corev1.PodTemplateSpec, containerDefs []types.ContainerDefinition) {
	if podTemplate.Annotations == nil {
		podTemplate.Annotations = make(map[string]string)
	}

	secretIndex := 0
	for _, containerDef := range containerDefs {
		if containerDef.Secrets != nil {
			for _, secret := range containerDef.Secrets {
				if secret.Name != nil && secret.ValueFrom != nil {
					// Add annotation for each secret with container and environment variable info
					annotationKey := fmt.Sprintf("kecs.dev/secret-%d-arn", secretIndex)
					annotationValue := fmt.Sprintf("%s:%s:%s", *containerDef.Name, *secret.Name, *secret.ValueFrom)
					podTemplate.Annotations[annotationKey] = annotationValue
					secretIndex++
				}
			}
		}
	}

	// Add total count of secrets
	if secretIndex > 0 {
		podTemplate.Annotations["kecs.dev/secret-count"] = fmt.Sprintf("%d", secretIndex)
	}
}
