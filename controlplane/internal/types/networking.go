package types

// NetworkMode represents the Docker networking mode for a task
type NetworkMode string

const (
	NetworkModeAWSVPC NetworkMode = "awsvpc"
	NetworkModeBridge NetworkMode = "bridge"
	NetworkModeHost   NetworkMode = "host"
	NetworkModeNone   NetworkMode = "none"
)

// NetworkConfiguration represents the network configuration for ECS tasks
type NetworkConfiguration struct {
	AwsvpcConfiguration *AwsvpcConfiguration `json:"awsvpcConfiguration,omitempty"`
}

// AwsvpcConfiguration represents the awsvpc network configuration
type AwsvpcConfiguration struct {
	Subnets        []string       `json:"subnets"`
	SecurityGroups []string       `json:"securityGroups,omitempty"`
	AssignPublicIp AssignPublicIp `json:"assignPublicIp,omitempty"`
}

// AssignPublicIp represents whether tasks receive a public IP address
type AssignPublicIp string

const (
	AssignPublicIpEnabled  AssignPublicIp = "ENABLED"
	AssignPublicIpDisabled AssignPublicIp = "DISABLED"
)

// Note: NetworkInterface and NetworkBinding are already defined in task.go

// ServiceRegistry represents a service registry configuration
type ServiceRegistry struct {
	RegistryArn   *string `json:"registryArn,omitempty"`
	Port          *int    `json:"port,omitempty"`
	ContainerName *string `json:"containerName,omitempty"`
	ContainerPort *int    `json:"containerPort,omitempty"`
}

// LoadBalancer represents a load balancer configuration
type LoadBalancer struct {
	TargetGroupArn   *string `json:"targetGroupArn,omitempty"`
	LoadBalancerName *string `json:"loadBalancerName,omitempty"`
	ContainerName    *string `json:"containerName,omitempty"`
	ContainerPort    *int    `json:"containerPort,omitempty"`
}

// NetworkAnnotations contains Kubernetes annotations for network configuration
type NetworkAnnotations struct {
	NetworkMode    string `json:"ecs.amazonaws.com/network-mode,omitempty"`
	Subnets        string `json:"ecs.amazonaws.com/subnets,omitempty"`
	SecurityGroups string `json:"ecs.amazonaws.com/security-groups,omitempty"`
	AssignPublicIp string `json:"ecs.amazonaws.com/assign-public-ip,omitempty"`
	PrivateIp      string `json:"ecs.amazonaws.com/private-ip,omitempty"`
}

// GetNetworkMode returns the network mode from a string pointer, defaulting to awsvpc
func GetNetworkMode(mode *string) NetworkMode {
	if mode == nil || *mode == "" {
		return NetworkModeAWSVPC
	}
	return NetworkMode(*mode)
}

// IsValidNetworkMode checks if the provided network mode is valid
func IsValidNetworkMode(mode string) bool {
	switch NetworkMode(mode) {
	case NetworkModeAWSVPC, NetworkModeBridge, NetworkModeHost, NetworkModeNone:
		return true
	default:
		return false
	}
}

// RequiresNetworkConfiguration returns true if the network mode requires network configuration
func RequiresNetworkConfiguration(mode NetworkMode) bool {
	return mode == NetworkModeAWSVPC
}
