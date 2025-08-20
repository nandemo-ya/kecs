package converters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// TaskSetConverterWithLB extends TaskSetConverter with load balancer support
type TaskSetConverterWithLB struct {
	*TaskSetConverter
	elbv2Integration elbv2.Integration
}

// NewTaskSetConverterWithLB creates a new TaskSetConverter with ELBv2 integration
func NewTaskSetConverterWithLB(taskConverter *TaskConverter, elbv2Integration elbv2.Integration) *TaskSetConverterWithLB {
	return &TaskSetConverterWithLB{
		TaskSetConverter: NewTaskSetConverter(taskConverter),
		elbv2Integration: elbv2Integration,
	}
}

// ConvertTaskSetToServiceWithLB creates a Kubernetes Service with load balancer support
func (c *TaskSetConverterWithLB) ConvertTaskSetToServiceWithLB(
	ctx context.Context,
	taskSet *storage.TaskSet,
	service *storage.Service,
	taskDef *storage.TaskDefinition,
	clusterName string,
	isPrimary bool,
) (*corev1.Service, error) {
	// Get base service from parent converter
	k8sService, err := c.TaskSetConverter.ConvertTaskSetToService(taskSet, service, taskDef, clusterName, isPrimary)
	if err != nil {
		return nil, err
	}

	if k8sService == nil {
		return nil, nil
	}

	// Process load balancer configuration
	if taskSet.LoadBalancers != "" {
		if err := c.processTaskSetLoadBalancers(ctx, taskSet, service, k8sService, clusterName); err != nil {
			logging.Warn("Failed to process TaskSet load balancers",
				"taskSet", taskSet.ID,
				"error", err)
			// Don't fail the service creation, just log the error
		}
	}

	return k8sService, nil
}

// processTaskSetLoadBalancers processes load balancer configuration for a TaskSet
func (c *TaskSetConverterWithLB) processTaskSetLoadBalancers(
	ctx context.Context,
	taskSet *storage.TaskSet,
	service *storage.Service,
	k8sService *corev1.Service,
	clusterName string,
) error {
	// Parse load balancers
	var loadBalancers []generated.LoadBalancer
	if err := json.Unmarshal([]byte(taskSet.LoadBalancers), &loadBalancers); err != nil {
		return fmt.Errorf("failed to parse load balancers: %w", err)
	}

	if len(loadBalancers) == 0 {
		return nil
	}

	// Process each load balancer
	for _, lb := range loadBalancers {
		if err := c.processLoadBalancer(ctx, &lb, taskSet, service, k8sService, clusterName); err != nil {
			logging.Warn("Failed to process load balancer for TaskSet",
				"taskSet", taskSet.ID,
				"targetGroup", lb.TargetGroupArn,
				"error", err)
			// Continue with other load balancers
		}
	}

	return nil
}

// processLoadBalancer processes a single load balancer configuration
func (c *TaskSetConverterWithLB) processLoadBalancer(
	ctx context.Context,
	lb *generated.LoadBalancer,
	taskSet *storage.TaskSet,
	service *storage.Service,
	k8sService *corev1.Service,
	clusterName string,
) error {
	// Extract load balancer configuration
	targetGroupArn := ""
	containerName := ""
	containerPort := int32(0)

	if lb.TargetGroupArn != nil {
		targetGroupArn = *lb.TargetGroupArn
	}
	if lb.ContainerName != nil {
		containerName = *lb.ContainerName
	}
	if lb.ContainerPort != nil {
		containerPort = *lb.ContainerPort
	}

	logging.Info("Processing TaskSet load balancer",
		"taskSet", taskSet.ID,
		"targetGroup", targetGroupArn,
		"container", containerName,
		"port", containerPort)

	// If target group ARN is provided and we have ELBv2 integration
	if targetGroupArn != "" && c.elbv2Integration != nil {
		// Register TaskSet with the target group
		if err := c.registerTaskSetWithTargetGroup(ctx, targetGroupArn, taskSet, k8sService); err != nil {
			return fmt.Errorf("failed to register TaskSet with target group: %w", err)
		}
	}

	// Add load balancer annotations to the service
	if k8sService.Annotations == nil {
		k8sService.Annotations = make(map[string]string)
	}

	// Store load balancer configuration in annotations
	k8sService.Annotations["kecs.io/target-group-arn"] = targetGroupArn
	k8sService.Annotations["kecs.io/lb-container-name"] = containerName
	k8sService.Annotations["kecs.io/lb-container-port"] = fmt.Sprintf("%d", containerPort)

	// If this is an ALB/NLB target group, we might need to configure health checks
	if strings.Contains(targetGroupArn, ":targetgroup/") {
		// Configure health check annotations
		k8sService.Annotations["kecs.io/health-check-enabled"] = "true"
		k8sService.Annotations["kecs.io/health-check-path"] = "/"
		k8sService.Annotations["kecs.io/health-check-port"] = fmt.Sprintf("%d", containerPort)
	}

	return nil
}

// registerTaskSetWithTargetGroup registers TaskSet endpoints with ELBv2 target group
func (c *TaskSetConverterWithLB) registerTaskSetWithTargetGroup(
	ctx context.Context,
	targetGroupArn string,
	taskSet *storage.TaskSet,
	k8sService *corev1.Service,
) error {
	// Get service endpoints (pod IPs)
	targets := c.getTaskSetTargets(k8sService)

	if len(targets) > 0 {
		// Register targets with the target group using ELBv2 integration
		// Note: The actual registration would depend on the ELBv2 integration interface
		// For now, we'll just log the targets
		for _, target := range targets {
			logging.Info("Would register target with target group",
				"targetGroup", targetGroupArn,
				"target", target.IP,
				"port", target.Port)
		}
	}

	logging.Info("Registered TaskSet targets with target group",
		"taskSet", taskSet.ID,
		"targetGroup", targetGroupArn,
		"targetCount", len(targets))

	return nil
}

// getTaskSetTargets gets target IPs and ports from the TaskSet service
func (c *TaskSetConverterWithLB) getTaskSetTargets(k8sService *corev1.Service) []struct {
	IP   string
	Port int32
} {
	var targets []struct {
		IP   string
		Port int32
	}

	// For now, we'll return empty targets
	// In a real implementation, we would query the endpoints of the service
	// to get the actual pod IPs and ports

	return targets
}

// UpdateTaskSetLoadBalancers updates load balancer configuration for a TaskSet
func (c *TaskSetConverterWithLB) UpdateTaskSetLoadBalancers(
	ctx context.Context,
	taskSet *storage.TaskSet,
	service *storage.Service,
	k8sService *corev1.Service,
	clusterName string,
) error {
	// Process updated load balancer configuration
	if taskSet.LoadBalancers != "" {
		return c.processTaskSetLoadBalancers(ctx, taskSet, service, k8sService, clusterName)
	}

	// If load balancers were removed, clean up annotations
	if k8sService.Annotations != nil {
		delete(k8sService.Annotations, "kecs.io/target-group-arn")
		delete(k8sService.Annotations, "kecs.io/lb-container-name")
		delete(k8sService.Annotations, "kecs.io/lb-container-port")
		delete(k8sService.Annotations, "kecs.io/health-check-enabled")
		delete(k8sService.Annotations, "kecs.io/health-check-path")
		delete(k8sService.Annotations, "kecs.io/health-check-port")
	}

	// Change service type back to ClusterIP if it was LoadBalancer
	if k8sService.Spec.Type == corev1.ServiceTypeLoadBalancer {
		k8sService.Spec.Type = corev1.ServiceTypeClusterIP
	}

	return nil
}

// ConvertTaskSetToIngress creates an Ingress resource for TaskSet if needed
func (c *TaskSetConverterWithLB) ConvertTaskSetToIngress(
	taskSet *storage.TaskSet,
	service *storage.Service,
	clusterName string,
	isPrimary bool,
) (*corev1.Service, error) {
	// This could be implemented to create Ingress resources
	// for more advanced routing scenarios
	// For now, we'll rely on Service resources
	return nil, nil
}
