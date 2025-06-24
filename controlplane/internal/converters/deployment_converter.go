package converters

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

// ConvertDeploymentToK8s converts an internal deployment definition to a Kubernetes deployment
func ConvertDeploymentToK8s(deployment *types.Deployment, containerDefs []types.ContainerDefinition) *appsv1.Deployment {
	// Convert container definitions to pod spec containers
	containers := make([]corev1.Container, 0, len(containerDefs))
	for _, containerDef := range containerDefs {
		container := corev1.Container{
			Name:  containerDef.Name,
			Image: containerDef.Image,
			Env:   convertEnvironmentVariables(containerDef.Environment),
		}

		// Set resource requirements if specified
		if containerDef.CPU != 0 || containerDef.Memory != 0 {
			container.Resources = corev1.ResourceRequirements{}
			if containerDef.Memory != 0 {
				container.Resources.Requests = corev1.ResourceList{
					corev1.ResourceMemory: parseMemoryMiB(containerDef.Memory),
				}
				container.Resources.Limits = corev1.ResourceList{
					corev1.ResourceMemory: parseMemoryMiB(containerDef.Memory),
				}
			}
			if containerDef.CPU != 0 {
				if container.Resources.Requests == nil {
					container.Resources.Requests = corev1.ResourceList{}
				}
				if container.Resources.Limits == nil {
					container.Resources.Limits = corev1.ResourceList{}
				}
				container.Resources.Requests[corev1.ResourceCPU] = parseCPUUnits(containerDef.CPU)
				container.Resources.Limits[corev1.ResourceCPU] = parseCPUUnits(containerDef.CPU)
			}
		}

		// Add port mappings
		for _, portMapping := range containerDef.PortMappings {
			container.Ports = append(container.Ports, corev1.ContainerPort{
				ContainerPort: int32(portMapping.ContainerPort),
				Protocol:      parseProtocol(portMapping.Protocol),
			})
		}

		// Add health check if defined
		if containerDef.HealthCheck != nil {
			container.LivenessProbe = convertHealthCheck(containerDef.HealthCheck)
			container.ReadinessProbe = convertHealthCheck(containerDef.HealthCheck)
		}

		// Set working directory if specified
		if containerDef.WorkingDirectory != "" {
			container.WorkingDir = containerDef.WorkingDirectory
		}

		// Add commands if specified
		if len(containerDef.Command) > 0 {
			container.Command = containerDef.Command
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

	return k8sDeployment
}

// convertHealthCheck converts ECS health check to Kubernetes probe
func convertHealthCheck(healthCheck *types.HealthCheck) *corev1.Probe {
	if healthCheck == nil || len(healthCheck.Command) == 0 {
		return nil
	}

	probe := &corev1.Probe{
		InitialDelaySeconds: int32(healthCheck.StartPeriod),
		PeriodSeconds:       int32(healthCheck.Interval),
		TimeoutSeconds:      int32(healthCheck.Timeout),
		FailureThreshold:    int32(healthCheck.Retries),
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