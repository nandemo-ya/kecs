package converters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NetworkConverter handles conversion between ECS and Kubernetes networking
type NetworkConverter struct {
	region    string
	accountID string
}

// NewNetworkConverter creates a new NetworkConverter
func NewNetworkConverter(region, accountID string) *NetworkConverter {
	return &NetworkConverter{
		region:    region,
		accountID: accountID,
	}
}

// ConvertNetworkConfiguration converts ECS NetworkConfiguration to Kubernetes annotations
func (nc *NetworkConverter) ConvertNetworkConfiguration(config *generated.NetworkConfiguration, networkMode types.NetworkMode) map[string]string {
	annotations := make(map[string]string)
	
	// Add network mode annotation
	annotations["ecs.amazonaws.com/network-mode"] = string(networkMode)
	
	if config == nil || config.AwsvpcConfiguration == nil {
		return annotations
	}
	
	awsvpc := config.AwsvpcConfiguration
	
	// Add subnets
	if len(awsvpc.Subnets) > 0 {
		annotations["ecs.amazonaws.com/subnets"] = strings.Join(awsvpc.Subnets, ",")
	}
	
	// Add security groups
	if len(awsvpc.SecurityGroups) > 0 {
		annotations["ecs.amazonaws.com/security-groups"] = strings.Join(awsvpc.SecurityGroups, ",")
	}
	
	// Add public IP assignment
	if awsvpc.AssignPublicIp != nil {
		annotations["ecs.amazonaws.com/assign-public-ip"] = string(*awsvpc.AssignPublicIp)
	}
	
	return annotations
}

// ConvertToNetworkPolicy converts ECS security groups to Kubernetes NetworkPolicy
func (nc *NetworkConverter) ConvertToNetworkPolicy(serviceName string, namespace string, securityGroups []string) (*networkingv1.NetworkPolicy, error) {
	if len(securityGroups) == 0 {
		return nil, nil
	}
	
	// Create a NetworkPolicy that represents the security group rules
	// In a real implementation, this would fetch security group rules from LocalStack
	// For now, we'll create a basic policy
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-network-policy", serviceName),
			Namespace: namespace,
			Labels: map[string]string{
				"app":                   serviceName,
				"ecs.amazonaws.com/type": "service",
			},
			Annotations: map[string]string{
				"ecs.amazonaws.com/security-groups": strings.Join(securityGroups, ","),
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": serviceName,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			// Default: allow all ingress and egress
			// This should be refined based on actual security group rules
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					// Allow all ingress by default
					// In production, this would be based on security group rules
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					// Allow all egress by default
					// In production, this would be based on security group rules
				},
			},
		},
	}
	
	return policy, nil
}

// ConvertServiceRegistry converts ECS service registry to Kubernetes service configuration
func (nc *NetworkConverter) ConvertServiceRegistry(registry *generated.ServiceRegistry, serviceName string) (map[string]string, *corev1.ServicePort) {
	if registry == nil {
		return nil, nil
	}
	
	annotations := make(map[string]string)
	
	// Add service registry annotations
	if registry.RegistryArn != nil {
		annotations["ecs.amazonaws.com/service-registry-arn"] = *registry.RegistryArn
	}
	
	// Create service port if specified
	var servicePort *corev1.ServicePort
	if registry.Port != nil {
		servicePort = &corev1.ServicePort{
			Name:     "registry-port",
			Port:     *registry.Port,
			Protocol: corev1.ProtocolTCP,
		}
		
		if registry.ContainerPort != nil {
			servicePort.TargetPort = intstr.FromInt(int(*registry.ContainerPort))
		}
	} else if registry.ContainerPort != nil {
		servicePort = &corev1.ServicePort{
			Name:       "registry-port",
			Port:       *registry.ContainerPort,
			TargetPort: intstr.FromInt(int(*registry.ContainerPort)),
			Protocol:   corev1.ProtocolTCP,
		}
	}
	
	return annotations, servicePort
}

// ConvertLoadBalancer converts ECS load balancer configuration to Kubernetes service annotations
func (nc *NetworkConverter) ConvertLoadBalancer(loadBalancer *generated.LoadBalancer) (map[string]string, *corev1.ServicePort) {
	if loadBalancer == nil {
		return nil, nil
	}
	
	annotations := make(map[string]string)
	
	// Add load balancer annotations
	if loadBalancer.TargetGroupArn != nil {
		annotations["ecs.amazonaws.com/target-group-arn"] = *loadBalancer.TargetGroupArn
	}
	
	if loadBalancer.LoadBalancerName != nil {
		annotations["ecs.amazonaws.com/load-balancer-name"] = *loadBalancer.LoadBalancerName
	}
	
	// Create service port for load balancer
	var servicePort *corev1.ServicePort
	if loadBalancer.ContainerPort != nil {
		servicePort = &corev1.ServicePort{
			Name:       fmt.Sprintf("lb-%s", getPortName(loadBalancer.ContainerName)),
			Port:       *loadBalancer.ContainerPort,
			TargetPort: intstr.FromInt(int(*loadBalancer.ContainerPort)),
			Protocol:   corev1.ProtocolTCP,
		}
	}
	
	return annotations, servicePort
}

// ExtractNetworkInterfaces extracts network interface information from pod status
func (nc *NetworkConverter) ExtractNetworkInterfaces(pod *corev1.Pod) []types.NetworkInterface {
	var interfaces []types.NetworkInterface
	
	// Primary interface from pod IP
	if pod.Status.PodIP != "" {
		attachmentID := fmt.Sprintf("eni-attach-%s", pod.UID)
		interfaces = append(interfaces, types.NetworkInterface{
			AttachmentId:       attachmentID,
			PrivateIpv4Address: pod.Status.PodIP,
		})
	}
	
	// IPv6 if available
	for _, podIP := range pod.Status.PodIPs {
		if strings.Contains(podIP.IP, ":") {
			// IPv6 address
			if len(interfaces) > 0 {
				interfaces[0].Ipv6Address = podIP.IP
			}
			break
		}
	}
	
	return interfaces
}

// GetNetworkBindings extracts network bindings from container ports
func (nc *NetworkConverter) GetNetworkBindings(container corev1.Container, podIP string) []types.NetworkBinding {
	var bindings []types.NetworkBinding
	
	for _, port := range container.Ports {
		containerPort := int(port.ContainerPort)
		hostPort := int(port.HostPort)
		protocol := string(port.Protocol)
		
		binding := types.NetworkBinding{
			ContainerPort: containerPort,
			Protocol:      protocol,
		}
		
		// For awsvpc mode, bind IP is the pod IP
		if podIP != "" {
			binding.BindIP = podIP
		}
		
		// Host port is only relevant for bridge/host modes
		if hostPort > 0 {
			binding.HostPort = hostPort
		}
		
		bindings = append(bindings, binding)
	}
	
	return bindings
}

// ParseNetworkAnnotations parses network-related annotations from Kubernetes resources
func (nc *NetworkConverter) ParseNetworkAnnotations(annotations map[string]string) *types.NetworkAnnotations {
	if annotations == nil {
		return nil
	}
	
	return &types.NetworkAnnotations{
		NetworkMode:    annotations["ecs.amazonaws.com/network-mode"],
		Subnets:        annotations["ecs.amazonaws.com/subnets"],
		SecurityGroups: annotations["ecs.amazonaws.com/security-groups"],
		AssignPublicIp: annotations["ecs.amazonaws.com/assign-public-ip"],
		PrivateIp:      annotations["ecs.amazonaws.com/private-ip"],
	}
}

// SerializeNetworkConfig serializes network configuration to JSON for storage
func (nc *NetworkConverter) SerializeNetworkConfig(config *generated.NetworkConfiguration) (string, error) {
	if config == nil {
		return "", nil
	}
	
	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to serialize network configuration: %w", err)
	}
	
	return string(data), nil
}

// DeserializeNetworkConfig deserializes network configuration from JSON
func (nc *NetworkConverter) DeserializeNetworkConfig(data string) (*generated.NetworkConfiguration, error) {
	if data == "" {
		return nil, nil
	}
	
	var config generated.NetworkConfiguration
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, fmt.Errorf("failed to deserialize network configuration: %w", err)
	}
	
	return &config, nil
}

// getPortName generates a port name from container name
func getPortName(containerName *string) string {
	if containerName == nil || *containerName == "" {
		return "default"
	}
	
	// Kubernetes port names must be lowercase and max 15 chars
	name := strings.ToLower(*containerName)
	if len(name) > 15 {
		name = name[:15]
	}
	
	// Replace invalid characters
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	
	return name
}