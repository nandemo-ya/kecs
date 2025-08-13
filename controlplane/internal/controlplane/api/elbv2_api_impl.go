package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

// Target health states
const (
	TargetHealthStateInitial     = "initial"
	TargetHealthStateHealthy     = "healthy"
	TargetHealthStateUnhealthy   = "unhealthy"
	TargetHealthStateUnused      = "unused"
	TargetHealthStateRegistering = "registering"
	TargetHealthStateDeregistering = "deregistering"
	TargetHealthStateDraining    = "draining"
	TargetHealthStateUnavailable = "unavailable"
)

// ELBv2APIImpl implements the generated ElasticLoadBalancing_v10API interface
type ELBv2APIImpl struct {
	storage          storage.Storage
	elbv2Integration elbv2.Integration
	region           string
	accountID        string
}

// NewELBv2API creates a new ELBv2 API implementation
func NewELBv2API(storage storage.Storage, elbv2Integration elbv2.Integration, region, accountID string) generated_elbv2.ElasticLoadBalancing_v10API {
	return &ELBv2APIImpl{
		storage:          storage,
		elbv2Integration: elbv2Integration,
		region:           region,
		accountID:        accountID,
	}
}

// CreateLoadBalancer implements the CreateLoadBalancer operation
func (api *ELBv2APIImpl) CreateLoadBalancer(ctx context.Context, input *generated_elbv2.CreateLoadBalancerInput) (*generated_elbv2.CreateLoadBalancerOutput, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("LoadBalancerName is required")
	}

	// Check if load balancer already exists
	existingLB, err := api.storage.ELBv2Store().GetLoadBalancerByName(ctx, input.Name)
	if err != nil {
		return nil, err
	}
	if existingLB != nil {
		return nil, fmt.Errorf("load balancer %s already exists", input.Name)
	}

	// Generate ARN
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:loadbalancer/app/%s/%s",
		api.region, api.accountID, input.Name, uuid.New().String()[:8])

	// Generate DNS name
	dnsName := fmt.Sprintf("%s-%s.%s.elb.amazonaws.com", input.Name, uuid.New().String()[:8], api.region)

	// Determine scheme
	scheme := "internet-facing"
	if input.Scheme != nil {
		scheme = string(*input.Scheme)
	}

	// Determine type
	lbType := "application"
	if input.Type != nil {
		lbType = string(*input.Type)
	}

	// Determine IP address type
	ipAddressType := "ipv4"
	if input.IpAddressType != nil {
		ipAddressType = string(*input.IpAddressType)
	}

	// Convert subnets
	var subnets []string
	if input.Subnets != nil {
		for _, subnet := range input.Subnets {
			subnets = append(subnets, subnet)
		}
	}
	if input.SubnetMappings != nil {
		for _, mapping := range input.SubnetMappings {
			if mapping.SubnetId != nil {
				subnets = append(subnets, *mapping.SubnetId)
			}
		}
	}

	// Convert security groups
	var securityGroups []string
	if input.SecurityGroups != nil {
		for _, sg := range input.SecurityGroups {
			securityGroups = append(securityGroups, sg)
		}
	}

	// Create load balancer in storage
	now := time.Now()
	dbLB := &storage.ELBv2LoadBalancer{
		ARN:               arn,
		Name:              input.Name,
		DNSName:           dnsName,
		CanonicalHostedZoneID: "Z215JYRZR1TBD5",
		CreatedAt:         now,
		UpdatedAt:         now,
		Scheme:            scheme,
		VpcID:             "vpc-default",
		State:             "provisioning",
		Type:              lbType,
		IpAddressType:     ipAddressType,
		Subnets:           subnets,
		SecurityGroups:    securityGroups,
		Tags:              make(map[string]string),
		Region:            api.region,
		AccountID:         api.accountID,
	}

	if err := api.storage.ELBv2Store().CreateLoadBalancer(ctx, dbLB); err != nil {
		return nil, err
	}

	// Deploy Traefik for this load balancer
	if _, err := api.elbv2Integration.CreateLoadBalancer(ctx, input.Name, subnets, securityGroups); err != nil {
		return nil, fmt.Errorf("failed to deploy Traefik for load balancer: %w", err)
	}

	// Create response
	state := generated_elbv2.LoadBalancerStateEnumPROVISIONING
	output := &generated_elbv2.CreateLoadBalancerOutput{
		LoadBalancers: []generated_elbv2.LoadBalancer{
			{
				LoadBalancerArn:           &arn,
				DNSName:                   &dnsName,
				CanonicalHostedZoneId:     utils.Ptr("Z215JYRZR1TBD5"),
				CreatedTime:               &now,
				LoadBalancerName:          &input.Name,
				Scheme:                    (*generated_elbv2.LoadBalancerSchemeEnum)(&scheme),
				VpcId:                     utils.Ptr("vpc-default"),
				State:                     &generated_elbv2.LoadBalancerState{Code: &state},
				Type:                      (*generated_elbv2.LoadBalancerTypeEnum)(&lbType),
				IpAddressType:             (*generated_elbv2.IpAddressType)(&ipAddressType),
				SecurityGroups:            input.SecurityGroups,
				AvailabilityZones:         []generated_elbv2.AvailabilityZone{},
			},
		},
	}

	return output, nil
}

// DeleteLoadBalancer implements the DeleteLoadBalancer operation
func (api *ELBv2APIImpl) DeleteLoadBalancer(ctx context.Context, input *generated_elbv2.DeleteLoadBalancerInput) (*generated_elbv2.DeleteLoadBalancerOutput, error) {
	if input.LoadBalancerArn == "" {
		return nil, fmt.Errorf("LoadBalancerArn is required")
	}

	// Check if load balancer exists
	existingLB, err := api.storage.ELBv2Store().GetLoadBalancer(ctx, input.LoadBalancerArn)
	if err != nil {
		return nil, err
	}
	if existingLB == nil {
		return nil, fmt.Errorf("load balancer %s not found", input.LoadBalancerArn)
	}

	// Delete load balancer
	if err := api.storage.ELBv2Store().DeleteLoadBalancer(ctx, input.LoadBalancerArn); err != nil {
		return nil, err
	}

	return &generated_elbv2.DeleteLoadBalancerOutput{}, nil
}

// DescribeLoadBalancers implements the DescribeLoadBalancers operation
func (api *ELBv2APIImpl) DescribeLoadBalancers(ctx context.Context, input *generated_elbv2.DescribeLoadBalancersInput) (*generated_elbv2.DescribeLoadBalancersOutput, error) {
	var loadBalancers []generated_elbv2.LoadBalancer

	if input.LoadBalancerArns != nil && len(input.LoadBalancerArns) > 0 {
		// Get specific load balancers by ARN
		for _, arn := range input.LoadBalancerArns {
			lb, err := api.storage.ELBv2Store().GetLoadBalancer(ctx, arn)
			if err != nil {
				return nil, err
			}
			if lb != nil {
				loadBalancers = append(loadBalancers, convertToLoadBalancer(lb))
			}
		}
	} else if input.Names != nil && len(input.Names) > 0 {
		// Get specific load balancers by name
		for _, name := range input.Names {
			lb, err := api.storage.ELBv2Store().GetLoadBalancerByName(ctx, name)
			if err != nil {
				return nil, err
			}
			if lb != nil {
				loadBalancers = append(loadBalancers, convertToLoadBalancer(lb))
			}
		}
	} else {
		// Get all load balancers
		lbs, err := api.storage.ELBv2Store().ListLoadBalancers(ctx, api.region)
		if err != nil {
			return nil, err
		}
		for _, lb := range lbs {
			loadBalancers = append(loadBalancers, convertToLoadBalancer(lb))
		}
	}

	return &generated_elbv2.DescribeLoadBalancersOutput{
		LoadBalancers: loadBalancers,
	}, nil
}

// CreateTargetGroup implements the CreateTargetGroup operation
func (api *ELBv2APIImpl) CreateTargetGroup(ctx context.Context, input *generated_elbv2.CreateTargetGroupInput) (*generated_elbv2.CreateTargetGroupOutput, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("TargetGroupName is required")
	}

	// Check if target group already exists
	existingTG, err := api.storage.ELBv2Store().GetTargetGroupByName(ctx, input.Name)
	if err != nil {
		return nil, err
	}
	if existingTG != nil {
		return nil, fmt.Errorf("target group %s already exists", input.Name)
	}

	// Generate ARN
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:targetgroup/%s/%s",
		api.region, api.accountID, input.Name, uuid.New().String()[:8])

	// Determine health check protocol
	healthCheckProtocol := "HTTP"
	if input.HealthCheckProtocol != nil {
		healthCheckProtocol = string(*input.HealthCheckProtocol)
	}

	// Create target group in storage
	now := time.Now()
	dbTG := &storage.ELBv2TargetGroup{
		ARN:                       arn,
		Name:                      input.Name,
		Protocol:                  string(*input.Protocol),
		Port:                      *input.Port,
		VpcID:                     *input.VpcId,
		TargetType:                "instance",
		HealthCheckEnabled:        true,
		HealthCheckPath:           "/",
		HealthCheckProtocol:       healthCheckProtocol,
		HealthCheckPort:           "traffic-port",
		HealthyThresholdCount:     2,
		UnhealthyThresholdCount:   5,
		HealthCheckTimeoutSeconds: 5,
		HealthCheckIntervalSeconds: 30,
		Matcher:                   "200",
		LoadBalancerArns:          []string{},
		Tags:                      make(map[string]string),
		Region:                    api.region,
		AccountID:                 api.accountID,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	if err := api.storage.ELBv2Store().CreateTargetGroup(ctx, dbTG); err != nil {
		return nil, err
	}

	// Create target group in Kubernetes
	if _, err := api.elbv2Integration.CreateTargetGroup(ctx, input.Name, *input.Port, string(*input.Protocol), *input.VpcId); err != nil {
		return nil, fmt.Errorf("failed to create target group in Kubernetes: %w", err)
	}

	// Create response
	output := &generated_elbv2.CreateTargetGroupOutput{
		TargetGroups: []generated_elbv2.TargetGroup{
			{
				TargetGroupArn:              &arn,
				TargetGroupName:             &input.Name,
				Protocol:                    input.Protocol,
				Port:                        input.Port,
				VpcId:                       input.VpcId,
				HealthCheckPath:             utils.Ptr("/"),
				HealthCheckProtocol:         (*generated_elbv2.ProtocolEnum)(&healthCheckProtocol),
				HealthCheckPort:             utils.Ptr("traffic-port"),
				HealthyThresholdCount:       utils.Ptr(int32(2)),
				UnhealthyThresholdCount:     utils.Ptr(int32(5)),
				HealthCheckTimeoutSeconds:   utils.Ptr(int32(5)),
				HealthCheckIntervalSeconds:  utils.Ptr(int32(30)),
				LoadBalancerArns:            []string{},
			},
		},
	}

	return output, nil
}

// DeleteTargetGroup implements the DeleteTargetGroup operation
func (api *ELBv2APIImpl) DeleteTargetGroup(ctx context.Context, input *generated_elbv2.DeleteTargetGroupInput) (*generated_elbv2.DeleteTargetGroupOutput, error) {
	if input.TargetGroupArn == "" {
		return nil, fmt.Errorf("TargetGroupArn is required")
	}

	// Check if target group exists
	existingTG, err := api.storage.ELBv2Store().GetTargetGroup(ctx, input.TargetGroupArn)
	if err != nil {
		return nil, err
	}
	if existingTG == nil {
		return nil, fmt.Errorf("target group %s not found", input.TargetGroupArn)
	}

	// Delete target group
	if err := api.storage.ELBv2Store().DeleteTargetGroup(ctx, input.TargetGroupArn); err != nil {
		return nil, err
	}

	return &generated_elbv2.DeleteTargetGroupOutput{}, nil
}

// RegisterTargets implements the RegisterTargets operation
func (api *ELBv2APIImpl) RegisterTargets(ctx context.Context, input *generated_elbv2.RegisterTargetsInput) (*generated_elbv2.RegisterTargetsOutput, error) {
	if input.TargetGroupArn == "" {
		return nil, fmt.Errorf("TargetGroupArn is required")
	}

	// Build targets array
	var targets []*storage.ELBv2Target
	for _, target := range input.Targets {
		if target.Id == "" {
			continue
		}

		port := int32(80)
		if target.Port != nil {
			port = *target.Port
		}

		az := ""
		if target.AvailabilityZone != nil {
			az = *target.AvailabilityZone
		}

		dbTarget := &storage.ELBv2Target{
			TargetGroupArn:   input.TargetGroupArn,
			ID:               target.Id,
			Port:             port,
			AvailabilityZone: az,
			HealthState:      TargetHealthStateRegistering,
			HealthReason:     "Target.RegistrationInProgress",
			HealthDescription: "Target registration is in progress",
			RegisteredAt:     time.Now(),
			UpdatedAt:        time.Now(),
		}
		targets = append(targets, dbTarget)
	}

	// Register all targets
	if err := api.storage.ELBv2Store().RegisterTargets(ctx, input.TargetGroupArn, targets); err != nil {
		return nil, err
	}

	// Convert to elbv2.Target type for integration
	var integrationTargets []elbv2.Target
	for _, t := range input.Targets {
		if t.Id != "" {
			port := int32(80)
			if t.Port != nil {
				port = *t.Port
			}
			integrationTargets = append(integrationTargets, elbv2.Target{
				Id:   t.Id,
				Port: port,
			})
		}
	}

	// Register targets in Kubernetes
	if err := api.elbv2Integration.RegisterTargets(ctx, input.TargetGroupArn, integrationTargets); err != nil {
		return nil, fmt.Errorf("failed to register targets in Kubernetes: %w", err)
	}

	return &generated_elbv2.RegisterTargetsOutput{}, nil
}

// DeregisterTargets implements the DeregisterTargets operation
func (api *ELBv2APIImpl) DeregisterTargets(ctx context.Context, input *generated_elbv2.DeregisterTargetsInput) (*generated_elbv2.DeregisterTargetsOutput, error) {
	if input.TargetGroupArn == "" {
		return nil, fmt.Errorf("TargetGroupArn is required")
	}

	// Mark targets as deregistering first
	for _, target := range input.Targets {
		if target.Id == "" {
			continue
		}
		
		// Update target health to deregistering
		targetHealth := &storage.ELBv2TargetHealth{
			State:       TargetHealthStateDeregistering,
			Reason:      "Target.DeregistrationInProgress",
			Description: "Target deregistration is in progress",
		}
		api.storage.ELBv2Store().UpdateTargetHealth(ctx, input.TargetGroupArn, target.Id, targetHealth)
	}

	// Build target IDs array for final removal
	var targetIDs []string
	for _, target := range input.Targets {
		if target.Id == "" {
			continue
		}
		targetIDs = append(targetIDs, target.Id)
	}

	// Deregister all targets (remove from storage)
	if err := api.storage.ELBv2Store().DeregisterTargets(ctx, input.TargetGroupArn, targetIDs); err != nil {
		return nil, err
	}

	// Convert to elbv2.Target type for integration
	var integrationTargets []elbv2.Target
	for _, t := range input.Targets {
		if t.Id != "" {
			port := int32(80)
			if t.Port != nil {
				port = *t.Port
			}
			integrationTargets = append(integrationTargets, elbv2.Target{
				Id:   t.Id,
				Port: port,
			})
		}
	}

	// Deregister targets in Kubernetes
	if err := api.elbv2Integration.DeregisterTargets(ctx, input.TargetGroupArn, integrationTargets); err != nil {
		return nil, fmt.Errorf("failed to deregister targets in Kubernetes: %w", err)
	}

	return &generated_elbv2.DeregisterTargetsOutput{}, nil
}

// DescribeTargetHealth implements the DescribeTargetHealth operation
func (api *ELBv2APIImpl) DescribeTargetHealth(ctx context.Context, input *generated_elbv2.DescribeTargetHealthInput) (*generated_elbv2.DescribeTargetHealthOutput, error) {
	if input.TargetGroupArn == "" {
		return nil, fmt.Errorf("TargetGroupArn is required")
	}

	// Get target group for health check configuration
	targetGroup, err := api.storage.ELBv2Store().GetTargetGroup(ctx, input.TargetGroupArn)
	if err != nil {
		return nil, err
	}

	targets, err := api.storage.ELBv2Store().ListTargets(ctx, input.TargetGroupArn)
	if err != nil {
		return nil, err
	}

	var targetHealthDescriptions []generated_elbv2.TargetHealthDescription
	for _, target := range targets {
		// Perform health check and update target health
		healthState := api.performHealthCheck(ctx, target, targetGroup)
		
		// Update target health in storage
		targetHealth := &storage.ELBv2TargetHealth{
			State:       healthState,
			Reason:      api.getHealthReason(healthState),
			Description: api.getHealthDescription(healthState),
		}
		api.storage.ELBv2Store().UpdateTargetHealth(ctx, input.TargetGroupArn, target.ID, targetHealth)
		
		// Convert port to pointer
		port := target.Port
		az := target.AvailabilityZone
		var azPtr *string
		if az != "" {
			azPtr = &az
		}

		// Convert health state to generated enum
		healthStateEnum := api.convertHealthStateToEnum(healthState)

		targetHealthDescriptions = append(targetHealthDescriptions, generated_elbv2.TargetHealthDescription{
			Target: &generated_elbv2.TargetDescription{
				Id:               target.ID,
				Port:             &port,
				AvailabilityZone: azPtr,
			},
			TargetHealth: &generated_elbv2.TargetHealth{
				State:       &healthStateEnum,
				Description: &targetHealth.Description,
			},
		})
	}

	return &generated_elbv2.DescribeTargetHealthOutput{
		TargetHealthDescriptions: targetHealthDescriptions,
	}, nil
}

// CreateListener implements the CreateListener operation
func (api *ELBv2APIImpl) CreateListener(ctx context.Context, input *generated_elbv2.CreateListenerInput) (*generated_elbv2.CreateListenerOutput, error) {
	if input.LoadBalancerArn == "" {
		return nil, fmt.Errorf("LoadBalancerArn is required")
	}

	// Check if load balancer exists
	existingLB, err := api.storage.ELBv2Store().GetLoadBalancer(ctx, input.LoadBalancerArn)
	if err != nil {
		return nil, err
	}
	if existingLB == nil {
		return nil, fmt.Errorf("load balancer %s not found", input.LoadBalancerArn)
	}

	// Generate ARN
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:listener/app/%s/%s",
		api.region, api.accountID, uuid.New().String()[:8], uuid.New().String()[:8])

	// Create listener in storage
	now := time.Now()
	dbListener := &storage.ELBv2Listener{
		ARN:             arn,
		LoadBalancerArn: input.LoadBalancerArn,
		Port:            *input.Port,
		Protocol:        string(*input.Protocol),
		DefaultActions:  "[]", // JSON encoded empty array
		SslPolicy:       "",
		Certificates:    "[]", // JSON encoded empty array
		AlpnPolicy:      []string{},
		Tags:            make(map[string]string),
		Region:          api.region,
		AccountID:       api.accountID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := api.storage.ELBv2Store().CreateListener(ctx, dbListener); err != nil {
		return nil, err
	}

	// Get default target group ARN from DefaultActions
	var targetGroupArn string
	if len(input.DefaultActions) > 0 && input.DefaultActions[0].TargetGroupArn != nil {
		targetGroupArn = *input.DefaultActions[0].TargetGroupArn
	}

	// Create listener in Kubernetes
	if _, err := api.elbv2Integration.CreateListener(ctx, input.LoadBalancerArn, *input.Port, string(*input.Protocol), targetGroupArn); err != nil {
		return nil, fmt.Errorf("failed to create listener in Kubernetes: %w", err)
	}

	// Create response
	output := &generated_elbv2.CreateListenerOutput{
		Listeners: []generated_elbv2.Listener{
			{
				ListenerArn:     &arn,
				LoadBalancerArn: &input.LoadBalancerArn,
				Port:            input.Port,
				Protocol:        input.Protocol,
				DefaultActions:  []generated_elbv2.Action{},
			},
		},
	}

	return output, nil
}

// DeleteListener implements the DeleteListener operation
func (api *ELBv2APIImpl) DeleteListener(ctx context.Context, input *generated_elbv2.DeleteListenerInput) (*generated_elbv2.DeleteListenerOutput, error) {
	if input.ListenerArn == "" {
		return nil, fmt.Errorf("ListenerArn is required")
	}

	// Check if listener exists
	existingListener, err := api.storage.ELBv2Store().GetListener(ctx, input.ListenerArn)
	if err != nil {
		return nil, err
	}
	if existingListener == nil {
		return nil, fmt.Errorf("listener %s not found", input.ListenerArn)
	}

	// Delete from Kubernetes integration first if available
	if api.elbv2Integration != nil {
		if err := api.elbv2Integration.DeleteListener(ctx, input.ListenerArn); err != nil {
			logging.Debug("Failed to delete listener from Kubernetes", "error", err)
			// Don't fail the operation if K8s deletion fails
		}
	}

	// Delete listener from storage
	if err := api.storage.ELBv2Store().DeleteListener(ctx, input.ListenerArn); err != nil {
		return nil, err
	}

	return &generated_elbv2.DeleteListenerOutput{}, nil
}

// Helper functions and stub implementations for remaining operations

// Stub implementations for all remaining operations
func (api *ELBv2APIImpl) AddListenerCertificates(ctx context.Context, input *generated_elbv2.AddListenerCertificatesInput) (*generated_elbv2.AddListenerCertificatesOutput, error) {
	return &generated_elbv2.AddListenerCertificatesOutput{}, nil
}

func (api *ELBv2APIImpl) AddTags(ctx context.Context, input *generated_elbv2.AddTagsInput) (*generated_elbv2.AddTagsOutput, error) {
	if len(input.ResourceArns) == 0 {
		return nil, fmt.Errorf("ResourceArns is required")
	}
	if len(input.Tags) == 0 {
		return nil, fmt.Errorf("Tags is required")
	}

	// Convert tags to map format
	tagMap := make(map[string]string)
	for _, tag := range input.Tags {
		if tag.Value != nil {
			tagMap[tag.Key] = *tag.Value
		} else {
			tagMap[tag.Key] = ""
		}
	}

	// Process each resource
	for _, resourceArn := range input.ResourceArns {
		// Determine resource type from ARN
		if strings.Contains(resourceArn, ":loadbalancer/") {
			// Load balancer
			lb, err := api.storage.ELBv2Store().GetLoadBalancer(ctx, resourceArn)
			if err != nil {
				return nil, fmt.Errorf("failed to get load balancer %s: %w", resourceArn, err)
			}
			if lb == nil {
				return nil, fmt.Errorf("load balancer not found: %s", resourceArn)
			}

			// Merge tags
			if lb.Tags == nil {
				lb.Tags = make(map[string]string)
			}
			for k, v := range tagMap {
				lb.Tags[k] = v
			}

			// Update load balancer
			if err := api.storage.ELBv2Store().UpdateLoadBalancer(ctx, lb); err != nil {
				return nil, fmt.Errorf("failed to update load balancer tags: %w", err)
			}

		} else if strings.Contains(resourceArn, ":targetgroup/") {
			// Target group
			tg, err := api.storage.ELBv2Store().GetTargetGroup(ctx, resourceArn)
			if err != nil {
				return nil, fmt.Errorf("failed to get target group %s: %w", resourceArn, err)
			}
			if tg == nil {
				return nil, fmt.Errorf("target group not found: %s", resourceArn)
			}

			// Merge tags
			if tg.Tags == nil {
				tg.Tags = make(map[string]string)
			}
			for k, v := range tagMap {
				tg.Tags[k] = v
			}

			// Update target group
			if err := api.storage.ELBv2Store().UpdateTargetGroup(ctx, tg); err != nil {
				return nil, fmt.Errorf("failed to update target group tags: %w", err)
			}

		} else if strings.Contains(resourceArn, ":listener/") {
			// Listener
			listener, err := api.storage.ELBv2Store().GetListener(ctx, resourceArn)
			if err != nil {
				return nil, fmt.Errorf("failed to get listener %s: %w", resourceArn, err)
			}
			if listener == nil {
				return nil, fmt.Errorf("listener not found: %s", resourceArn)
			}

			// Merge tags
			if listener.Tags == nil {
				listener.Tags = make(map[string]string)
			}
			for k, v := range tagMap {
				listener.Tags[k] = v
			}

			// Update listener
			if err := api.storage.ELBv2Store().UpdateListener(ctx, listener); err != nil {
				return nil, fmt.Errorf("failed to update listener tags: %w", err)
			}

		} else {
			return nil, fmt.Errorf("unsupported resource type: %s", resourceArn)
		}
	}

	return &generated_elbv2.AddTagsOutput{}, nil
}

func (api *ELBv2APIImpl) AddTrustStoreRevocations(ctx context.Context, input *generated_elbv2.AddTrustStoreRevocationsInput) (*generated_elbv2.AddTrustStoreRevocationsOutput, error) {
	return &generated_elbv2.AddTrustStoreRevocationsOutput{}, nil
}

func (api *ELBv2APIImpl) CreateRule(ctx context.Context, input *generated_elbv2.CreateRuleInput) (*generated_elbv2.CreateRuleOutput, error) {
	if input.ListenerArn == "" {
		return nil, fmt.Errorf("ListenerArn is required")
	}
	if input.Priority == 0 {
		return nil, fmt.Errorf("Priority is required")
	}
	if len(input.Conditions) == 0 {
		return nil, fmt.Errorf("Conditions is required")
	}
	if len(input.Actions) == 0 {
		return nil, fmt.Errorf("Actions is required")
	}

	// Verify listener exists
	listener, err := api.storage.ELBv2Store().GetListener(ctx, input.ListenerArn)
	if err != nil {
		return nil, fmt.Errorf("failed to get listener %s: %w", input.ListenerArn, err)
	}
	if listener == nil {
		return nil, fmt.Errorf("listener not found: %s", input.ListenerArn)
	}

	// Check if priority is already used
	existingRules, err := api.storage.ELBv2Store().ListRules(ctx, input.ListenerArn)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing rules: %w", err)
	}
	for _, rule := range existingRules {
		if rule.Priority == input.Priority {
			return nil, fmt.Errorf("priority %d is already in use", input.Priority)
		}
	}

	// Generate ARN - extract load balancer name from ARN
	lbName := "unknown"
	if parts := strings.Split(listener.LoadBalancerArn, "/"); len(parts) >= 3 {
		lbName = parts[2]
	}
	ruleArn := fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:listener-rule/app/%s/%s/%s",
		api.region, api.accountID, lbName, uuid.New().String()[:8], uuid.New().String()[:8])

	// Marshal conditions and actions
	conditionsJSON, err := json.Marshal(input.Conditions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal conditions: %w", err)
	}
	actionsJSON, err := json.Marshal(input.Actions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal actions: %w", err)
	}

	// Create rule
	now := time.Now()
	rule := &storage.ELBv2Rule{
		ARN:         ruleArn,
		ListenerArn: input.ListenerArn,
		Priority:    input.Priority,
		Conditions:  string(conditionsJSON),
		Actions:     string(actionsJSON),
		IsDefault:   false,
		Tags:        make(map[string]string),
		Region:      api.region,
		AccountID:   api.accountID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Add tags if provided
	if input.Tags != nil {
		for _, tag := range input.Tags {
			if tag.Key != "" && tag.Value != nil {
				rule.Tags[tag.Key] = *tag.Value
			}
		}
	}

	// Save to storage
	if err := api.storage.ELBv2Store().CreateRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	// Sync rule to Kubernetes IngressRoute if integration is available
	if api.elbv2Integration != nil {
		// Check if the integration supports rule syncing
		if ruleSyncable, ok := api.elbv2Integration.(elbv2.RuleSyncable); ok {
			// Get listener details to find load balancer name and port
			listener, _ := api.storage.ELBv2Store().GetListener(ctx, rule.ListenerArn)
			if listener != nil {
				// Extract load balancer name from listener's load balancer ARN
				lbName := "unknown"
				if parts := strings.Split(listener.LoadBalancerArn, "/"); len(parts) >= 3 {
					lbName = parts[2]
				}
				// TODO: Get actual port from listener - for now use port 80
				port := int32(80)
				
				// Sync rules to IngressRoute
				if err := ruleSyncable.SyncRulesToListener(ctx, api.storage, rule.ListenerArn, lbName, port); err != nil {
					logging.Debug("Failed to sync rules to IngressRoute", "error", err)
				}
			}
		}
	}

	// Return created rule
	output := &generated_elbv2.CreateRuleOutput{
		Rules: []generated_elbv2.Rule{
			{
				RuleArn:     &ruleArn,
				Priority:    utils.Ptr(fmt.Sprintf("%d", rule.Priority)),
				Conditions:  input.Conditions,
				Actions:     input.Actions,
				IsDefault:   utils.Ptr(false),
			},
		},
	}

	return output, nil
}

func (api *ELBv2APIImpl) CreateTrustStore(ctx context.Context, input *generated_elbv2.CreateTrustStoreInput) (*generated_elbv2.CreateTrustStoreOutput, error) {
	return &generated_elbv2.CreateTrustStoreOutput{}, nil
}

func (api *ELBv2APIImpl) DeleteRule(ctx context.Context, input *generated_elbv2.DeleteRuleInput) (*generated_elbv2.DeleteRuleOutput, error) {
	if input.RuleArn == "" {
		return nil, fmt.Errorf("RuleArn is required")
	}

	// Check if rule exists
	rule, err := api.storage.ELBv2Store().GetRule(ctx, input.RuleArn)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("rule not found: %s", input.RuleArn)
		}
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	// Cannot delete default rule
	if rule.IsDefault {
		return nil, fmt.Errorf("cannot delete default rule")
	}

	// Get listener ARN before deleting the rule
	listenerArn := rule.ListenerArn
	
	// Delete rule from storage
	if err := api.storage.ELBv2Store().DeleteRule(ctx, input.RuleArn); err != nil {
		return nil, fmt.Errorf("failed to delete rule: %w", err)
	}

	// Sync rules to Kubernetes IngressRoute if integration is available
	if api.elbv2Integration != nil {
		// Check if the integration supports rule syncing
		if ruleSyncable, ok := api.elbv2Integration.(elbv2.RuleSyncable); ok {
			// Get listener details to find load balancer name and port
			listener, _ := api.storage.ELBv2Store().GetListener(ctx, listenerArn)
			if listener != nil {
				// Extract load balancer name from listener's load balancer ARN
				lbName := "unknown"
				if parts := strings.Split(listener.LoadBalancerArn, "/"); len(parts) >= 3 {
					lbName = parts[2]
				}
				// TODO: Get actual port from listener - for now use port 80
				port := int32(80)
				
				// Sync rules to IngressRoute
				if err := ruleSyncable.SyncRulesToListener(ctx, api.storage, listenerArn, lbName, port); err != nil {
					logging.Debug("Failed to sync rules to IngressRoute after delete", "error", err)
				}
			}
		}
	}

	return &generated_elbv2.DeleteRuleOutput{}, nil
}

func (api *ELBv2APIImpl) DeleteSharedTrustStoreAssociation(ctx context.Context, input *generated_elbv2.DeleteSharedTrustStoreAssociationInput) (*generated_elbv2.DeleteSharedTrustStoreAssociationOutput, error) {
	return &generated_elbv2.DeleteSharedTrustStoreAssociationOutput{}, nil
}

func (api *ELBv2APIImpl) DeleteTrustStore(ctx context.Context, input *generated_elbv2.DeleteTrustStoreInput) (*generated_elbv2.DeleteTrustStoreOutput, error) {
	return &generated_elbv2.DeleteTrustStoreOutput{}, nil
}

func (api *ELBv2APIImpl) DescribeAccountLimits(ctx context.Context, input *generated_elbv2.DescribeAccountLimitsInput) (*generated_elbv2.DescribeAccountLimitsOutput, error) {
	return &generated_elbv2.DescribeAccountLimitsOutput{}, nil
}

func (api *ELBv2APIImpl) DescribeCapacityReservation(ctx context.Context, input *generated_elbv2.DescribeCapacityReservationInput) (*generated_elbv2.DescribeCapacityReservationOutput, error) {
	return &generated_elbv2.DescribeCapacityReservationOutput{}, nil
}

func (api *ELBv2APIImpl) DescribeListenerAttributes(ctx context.Context, input *generated_elbv2.DescribeListenerAttributesInput) (*generated_elbv2.DescribeListenerAttributesOutput, error) {
	return &generated_elbv2.DescribeListenerAttributesOutput{}, nil
}

func (api *ELBv2APIImpl) DescribeListenerCertificates(ctx context.Context, input *generated_elbv2.DescribeListenerCertificatesInput) (*generated_elbv2.DescribeListenerCertificatesOutput, error) {
	return &generated_elbv2.DescribeListenerCertificatesOutput{}, nil
}

func (api *ELBv2APIImpl) DescribeListeners(ctx context.Context, input *generated_elbv2.DescribeListenersInput) (*generated_elbv2.DescribeListenersOutput, error) {
	var listeners []*storage.ELBv2Listener
	var err error

	// If specific listener ARNs are provided
	if input.ListenerArns != nil && len(input.ListenerArns) > 0 {
		for _, arn := range input.ListenerArns {
			listener, getErr := api.storage.ELBv2Store().GetListener(ctx, arn)
			if getErr != nil {
				return nil, getErr
			}
			if listener != nil {
				listeners = append(listeners, listener)
			}
		}
	} else if input.LoadBalancerArn != nil {
		// Get listeners for a specific load balancer
		listeners, err = api.storage.ELBv2Store().ListListeners(ctx, *input.LoadBalancerArn)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("LoadBalancerArn or ListenerArns must be specified")
	}

	// Convert to response format
	var responseListeners []generated_elbv2.Listener
	for _, listener := range listeners {
		responseListeners = append(responseListeners, api.convertToListener(listener))
	}

	return &generated_elbv2.DescribeListenersOutput{
		Listeners: responseListeners,
	}, nil
}

func (api *ELBv2APIImpl) DescribeLoadBalancerAttributes(ctx context.Context, input *generated_elbv2.DescribeLoadBalancerAttributesInput) (*generated_elbv2.DescribeLoadBalancerAttributesOutput, error) {
	return &generated_elbv2.DescribeLoadBalancerAttributesOutput{}, nil
}

func (api *ELBv2APIImpl) DescribeRules(ctx context.Context, input *generated_elbv2.DescribeRulesInput) (*generated_elbv2.DescribeRulesOutput, error) {
	return &generated_elbv2.DescribeRulesOutput{}, nil
}

func (api *ELBv2APIImpl) DescribeSSLPolicies(ctx context.Context, input *generated_elbv2.DescribeSSLPoliciesInput) (*generated_elbv2.DescribeSSLPoliciesOutput, error) {
	return &generated_elbv2.DescribeSSLPoliciesOutput{}, nil
}

func (api *ELBv2APIImpl) DescribeTags(ctx context.Context, input *generated_elbv2.DescribeTagsInput) (*generated_elbv2.DescribeTagsOutput, error) {
	if len(input.ResourceArns) == 0 {
		return nil, fmt.Errorf("ResourceArns is required")
	}

	var tagDescriptions []generated_elbv2.TagDescription

	// Process each resource
	for _, resourceArn := range input.ResourceArns {
		// Determine resource type from ARN
		var tags map[string]string
		var found bool

		if strings.Contains(resourceArn, ":loadbalancer/") {
			// Load balancer
			lb, err := api.storage.ELBv2Store().GetLoadBalancer(ctx, resourceArn)
			if err != nil {
				// Skip resources that are not found
				continue
			}
			if lb != nil {
				tags = lb.Tags
				found = true
			}

		} else if strings.Contains(resourceArn, ":targetgroup/") {
			// Target group
			tg, err := api.storage.ELBv2Store().GetTargetGroup(ctx, resourceArn)
			if err != nil {
				// Skip resources that are not found
				continue
			}
			if tg != nil {
				tags = tg.Tags
				found = true
			}

		} else if strings.Contains(resourceArn, ":listener/") {
			// Listener
			listener, err := api.storage.ELBv2Store().GetListener(ctx, resourceArn)
			if err != nil {
				// Skip resources that are not found
				continue
			}
			if listener != nil {
				tags = listener.Tags
				found = true
			}
		}

		// Add tag description if resource was found
		if found {
			// Convert map to Tag slice
			var tagList []generated_elbv2.Tag
			for k, v := range tags {
				tagList = append(tagList, generated_elbv2.Tag{
					Key:   k,
					Value: utils.Ptr(v),
				})
			}

			tagDescriptions = append(tagDescriptions, generated_elbv2.TagDescription{
				ResourceArn: &resourceArn,
				Tags:        tagList,
			})
		}
	}

	return &generated_elbv2.DescribeTagsOutput{
		TagDescriptions: tagDescriptions,
	}, nil
}

func (api *ELBv2APIImpl) DescribeTargetGroupAttributes(ctx context.Context, input *generated_elbv2.DescribeTargetGroupAttributesInput) (*generated_elbv2.DescribeTargetGroupAttributesOutput, error) {
	return &generated_elbv2.DescribeTargetGroupAttributesOutput{}, nil
}

func (api *ELBv2APIImpl) DescribeTargetGroups(ctx context.Context, input *generated_elbv2.DescribeTargetGroupsInput) (*generated_elbv2.DescribeTargetGroupsOutput, error) {
	var targetGroups []*storage.ELBv2TargetGroup
	var err error

	// If specific ARNs are provided, get those target groups
	if input.TargetGroupArns != nil && len(input.TargetGroupArns) > 0 {
		for _, arn := range input.TargetGroupArns {
			tg, getErr := api.storage.ELBv2Store().GetTargetGroup(ctx, arn)
			if getErr != nil {
				return nil, getErr
			}
			if tg != nil {
				targetGroups = append(targetGroups, tg)
			}
		}
	} else if input.LoadBalancerArn != nil {
		// Get target groups for a specific load balancer
		// TODO: Implement LoadBalancer -> TargetGroup relationship
		targetGroups, err = api.storage.ELBv2Store().ListTargetGroups(ctx, api.region)
		if err != nil {
			return nil, err
		}
	} else if input.Names != nil && len(input.Names) > 0 {
		// Get target groups by names
		for _, name := range input.Names {
			tg, getErr := api.storage.ELBv2Store().GetTargetGroupByName(ctx, name)
			if getErr != nil {
				return nil, getErr
			}
			if tg != nil {
				targetGroups = append(targetGroups, tg)
			}
		}
	} else {
		// List all target groups in the region
		targetGroups, err = api.storage.ELBv2Store().ListTargetGroups(ctx, api.region)
		if err != nil {
			return nil, err
		}
	}

	// Convert to response format
	var responseTargetGroups []generated_elbv2.TargetGroup
	for _, tg := range targetGroups {
		responseTargetGroups = append(responseTargetGroups, api.convertToTargetGroup(tg))
	}

	return &generated_elbv2.DescribeTargetGroupsOutput{
		TargetGroups: responseTargetGroups,
	}, nil
}

func (api *ELBv2APIImpl) DescribeTrustStoreAssociations(ctx context.Context, input *generated_elbv2.DescribeTrustStoreAssociationsInput) (*generated_elbv2.DescribeTrustStoreAssociationsOutput, error) {
	return &generated_elbv2.DescribeTrustStoreAssociationsOutput{}, nil
}

func (api *ELBv2APIImpl) DescribeTrustStoreRevocations(ctx context.Context, input *generated_elbv2.DescribeTrustStoreRevocationsInput) (*generated_elbv2.DescribeTrustStoreRevocationsOutput, error) {
	return &generated_elbv2.DescribeTrustStoreRevocationsOutput{}, nil
}

func (api *ELBv2APIImpl) DescribeTrustStores(ctx context.Context, input *generated_elbv2.DescribeTrustStoresInput) (*generated_elbv2.DescribeTrustStoresOutput, error) {
	return &generated_elbv2.DescribeTrustStoresOutput{}, nil
}

func (api *ELBv2APIImpl) GetResourcePolicy(ctx context.Context, input *generated_elbv2.GetResourcePolicyInput) (*generated_elbv2.GetResourcePolicyOutput, error) {
	return &generated_elbv2.GetResourcePolicyOutput{}, nil
}

func (api *ELBv2APIImpl) GetTrustStoreCaCertificatesBundle(ctx context.Context, input *generated_elbv2.GetTrustStoreCaCertificatesBundleInput) (*generated_elbv2.GetTrustStoreCaCertificatesBundleOutput, error) {
	return &generated_elbv2.GetTrustStoreCaCertificatesBundleOutput{}, nil
}

func (api *ELBv2APIImpl) GetTrustStoreRevocationContent(ctx context.Context, input *generated_elbv2.GetTrustStoreRevocationContentInput) (*generated_elbv2.GetTrustStoreRevocationContentOutput, error) {
	return &generated_elbv2.GetTrustStoreRevocationContentOutput{}, nil
}

func (api *ELBv2APIImpl) ModifyCapacityReservation(ctx context.Context, input *generated_elbv2.ModifyCapacityReservationInput) (*generated_elbv2.ModifyCapacityReservationOutput, error) {
	return &generated_elbv2.ModifyCapacityReservationOutput{}, nil
}

func (api *ELBv2APIImpl) ModifyIpPools(ctx context.Context, input *generated_elbv2.ModifyIpPoolsInput) (*generated_elbv2.ModifyIpPoolsOutput, error) {
	return &generated_elbv2.ModifyIpPoolsOutput{}, nil
}

func (api *ELBv2APIImpl) ModifyListener(ctx context.Context, input *generated_elbv2.ModifyListenerInput) (*generated_elbv2.ModifyListenerOutput, error) {
	if input.ListenerArn == "" {
		return nil, fmt.Errorf("ListenerArn is required")
	}

	// Get existing listener
	listener, err := api.storage.ELBv2Store().GetListener(ctx, input.ListenerArn)
	if err != nil {
		return nil, err
	}
	if listener == nil {
		return nil, fmt.Errorf("listener not found: %s", input.ListenerArn)
	}

	// Update listener fields
	now := time.Now()
	if input.Port != nil {
		listener.Port = *input.Port
	}
	if input.Protocol != nil {
		listener.Protocol = string(*input.Protocol)
	}
	if input.SslPolicy != nil {
		listener.SslPolicy = *input.SslPolicy
	}
	if input.DefaultActions != nil {
		// Convert actions to JSON for storage
		actionsJSON, err := json.Marshal(input.DefaultActions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal default actions: %w", err)
		}
		listener.DefaultActions = string(actionsJSON)
	}
	listener.UpdatedAt = now

	// Update in storage
	if err := api.storage.ELBv2Store().UpdateListener(ctx, listener); err != nil {
		return nil, fmt.Errorf("failed to update listener: %w", err)
	}

	// Update Kubernetes resources if integration is available
	if api.elbv2Integration != nil {
		// Get target group ARN from default actions if available
		var targetGroupArn string
		if input.DefaultActions != nil && len(input.DefaultActions) > 0 {
			for _, action := range input.DefaultActions {
				if action.Type == generated_elbv2.ActionTypeEnum("forward") && action.TargetGroupArn != nil {
					targetGroupArn = *action.TargetGroupArn
					break
				}
			}
		}

		// Update listener in Kubernetes
		if _, err := api.elbv2Integration.CreateListener(ctx, listener.LoadBalancerArn, listener.Port, listener.Protocol, targetGroupArn); err != nil {
			logging.Debug("Failed to update listener in Kubernetes", "error", err)
			// Don't fail the operation if K8s update fails
		}
	}

	// Return updated listener
	output := &generated_elbv2.ModifyListenerOutput{
		Listeners: []generated_elbv2.Listener{
			api.convertToListener(listener),
		},
	}

	return output, nil
}

func (api *ELBv2APIImpl) ModifyListenerAttributes(ctx context.Context, input *generated_elbv2.ModifyListenerAttributesInput) (*generated_elbv2.ModifyListenerAttributesOutput, error) {
	return &generated_elbv2.ModifyListenerAttributesOutput{}, nil
}

func (api *ELBv2APIImpl) ModifyLoadBalancerAttributes(ctx context.Context, input *generated_elbv2.ModifyLoadBalancerAttributesInput) (*generated_elbv2.ModifyLoadBalancerAttributesOutput, error) {
	return &generated_elbv2.ModifyLoadBalancerAttributesOutput{}, nil
}

func (api *ELBv2APIImpl) ModifyRule(ctx context.Context, input *generated_elbv2.ModifyRuleInput) (*generated_elbv2.ModifyRuleOutput, error) {
	if input.RuleArn == "" {
		return nil, fmt.Errorf("RuleArn is required")
	}

	// Get existing rule
	rule, err := api.storage.ELBv2Store().GetRule(ctx, input.RuleArn)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("rule not found: %s", input.RuleArn)
		}
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	// Cannot modify default rule
	if rule.IsDefault {
		return nil, fmt.Errorf("cannot modify default rule")
	}

	// Update conditions if provided
	if input.Conditions != nil {
		conditionsJSON, err := json.Marshal(input.Conditions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal conditions: %w", err)
		}
		rule.Conditions = string(conditionsJSON)
	}

	// Update actions if provided
	if input.Actions != nil {
		actionsJSON, err := json.Marshal(input.Actions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal actions: %w", err)
		}
		rule.Actions = string(actionsJSON)
	}

	// Update rule in storage
	if err := api.storage.ELBv2Store().UpdateRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update rule: %w", err)
	}

	// Return updated rule
	var conditions []generated_elbv2.RuleCondition
	var actions []generated_elbv2.Action
	
	if input.Conditions != nil {
		conditions = input.Conditions
	} else {
		json.Unmarshal([]byte(rule.Conditions), &conditions)
	}
	
	if input.Actions != nil {
		actions = input.Actions
	} else {
		json.Unmarshal([]byte(rule.Actions), &actions)
	}

	output := &generated_elbv2.ModifyRuleOutput{
		Rules: []generated_elbv2.Rule{
			{
				RuleArn:     &rule.ARN,
				Priority:    utils.Ptr(fmt.Sprintf("%d", rule.Priority)),
				Conditions:  conditions,
				Actions:     actions,
				IsDefault:   utils.Ptr(rule.IsDefault),
			},
		},
	}

	return output, nil
}

func (api *ELBv2APIImpl) ModifyTargetGroup(ctx context.Context, input *generated_elbv2.ModifyTargetGroupInput) (*generated_elbv2.ModifyTargetGroupOutput, error) {
	if input.TargetGroupArn == "" {
		return nil, fmt.Errorf("TargetGroupArn is required")
	}

	// Get existing target group
	targetGroup, err := api.storage.ELBv2Store().GetTargetGroup(ctx, input.TargetGroupArn)
	if err != nil {
		return nil, err
	}
	if targetGroup == nil {
		return nil, fmt.Errorf("target group not found: %s", input.TargetGroupArn)
	}

	// Update target group fields
	now := time.Now()
	if input.HealthCheckEnabled != nil {
		targetGroup.HealthCheckEnabled = *input.HealthCheckEnabled
	}
	if input.HealthCheckPath != nil {
		targetGroup.HealthCheckPath = *input.HealthCheckPath
	}
	if input.HealthCheckPort != nil {
		targetGroup.HealthCheckPort = *input.HealthCheckPort
	}
	if input.HealthCheckProtocol != nil {
		targetGroup.HealthCheckProtocol = string(*input.HealthCheckProtocol)
	}
	if input.HealthCheckIntervalSeconds != nil {
		targetGroup.HealthCheckIntervalSeconds = *input.HealthCheckIntervalSeconds
	}
	if input.HealthCheckTimeoutSeconds != nil {
		targetGroup.HealthCheckTimeoutSeconds = *input.HealthCheckTimeoutSeconds
	}
	if input.HealthyThresholdCount != nil {
		targetGroup.HealthyThresholdCount = *input.HealthyThresholdCount
	}
	if input.UnhealthyThresholdCount != nil {
		targetGroup.UnhealthyThresholdCount = *input.UnhealthyThresholdCount
	}
	if input.Matcher != nil {
		// Convert matcher to JSON for storage
		matcherJSON, err := json.Marshal(input.Matcher)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal matcher: %w", err)
		}
		targetGroup.Matcher = string(matcherJSON)
	}
	targetGroup.UpdatedAt = now

	// Update in storage
	if err := api.storage.ELBv2Store().UpdateTargetGroup(ctx, targetGroup); err != nil {
		return nil, fmt.Errorf("failed to update target group: %w", err)
	}

	// Update Kubernetes resources if integration is available
	if api.elbv2Integration != nil {
		// For now, we log a message about K8s update
		// In the future, we might need to update Service annotations or other K8s resources
		logging.Debug("Target group updated, Kubernetes resources may need manual update", "targetGroupArn", targetGroup.ARN)
	}

	// Return updated target group
	output := &generated_elbv2.ModifyTargetGroupOutput{
		TargetGroups: []generated_elbv2.TargetGroup{
			api.convertToTargetGroup(targetGroup),
		},
	}

	return output, nil
}

func (api *ELBv2APIImpl) ModifyTargetGroupAttributes(ctx context.Context, input *generated_elbv2.ModifyTargetGroupAttributesInput) (*generated_elbv2.ModifyTargetGroupAttributesOutput, error) {
	return &generated_elbv2.ModifyTargetGroupAttributesOutput{}, nil
}

func (api *ELBv2APIImpl) ModifyTrustStore(ctx context.Context, input *generated_elbv2.ModifyTrustStoreInput) (*generated_elbv2.ModifyTrustStoreOutput, error) {
	return &generated_elbv2.ModifyTrustStoreOutput{}, nil
}

func (api *ELBv2APIImpl) RemoveListenerCertificates(ctx context.Context, input *generated_elbv2.RemoveListenerCertificatesInput) (*generated_elbv2.RemoveListenerCertificatesOutput, error) {
	return &generated_elbv2.RemoveListenerCertificatesOutput{}, nil
}

func (api *ELBv2APIImpl) RemoveTags(ctx context.Context, input *generated_elbv2.RemoveTagsInput) (*generated_elbv2.RemoveTagsOutput, error) {
	if len(input.ResourceArns) == 0 {
		return nil, fmt.Errorf("ResourceArns is required")
	}
	if len(input.TagKeys) == 0 {
		return nil, fmt.Errorf("TagKeys is required")
	}

	// Process each resource
	for _, resourceArn := range input.ResourceArns {
		// Determine resource type from ARN
		if strings.Contains(resourceArn, ":loadbalancer/") {
			// Load balancer
			lb, err := api.storage.ELBv2Store().GetLoadBalancer(ctx, resourceArn)
			if err != nil {
				return nil, fmt.Errorf("failed to get load balancer %s: %w", resourceArn, err)
			}
			if lb == nil {
				return nil, fmt.Errorf("load balancer not found: %s", resourceArn)
			}

			// Remove tags
			if lb.Tags != nil {
				for _, key := range input.TagKeys {
					delete(lb.Tags, key)
				}
			}

			// Update load balancer
			if err := api.storage.ELBv2Store().UpdateLoadBalancer(ctx, lb); err != nil {
				return nil, fmt.Errorf("failed to update load balancer tags: %w", err)
			}

		} else if strings.Contains(resourceArn, ":targetgroup/") {
			// Target group
			tg, err := api.storage.ELBv2Store().GetTargetGroup(ctx, resourceArn)
			if err != nil {
				return nil, fmt.Errorf("failed to get target group %s: %w", resourceArn, err)
			}
			if tg == nil {
				return nil, fmt.Errorf("target group not found: %s", resourceArn)
			}

			// Remove tags
			if tg.Tags != nil {
				for _, key := range input.TagKeys {
					delete(tg.Tags, key)
				}
			}

			// Update target group
			if err := api.storage.ELBv2Store().UpdateTargetGroup(ctx, tg); err != nil {
				return nil, fmt.Errorf("failed to update target group tags: %w", err)
			}

		} else if strings.Contains(resourceArn, ":listener/") {
			// Listener
			listener, err := api.storage.ELBv2Store().GetListener(ctx, resourceArn)
			if err != nil {
				return nil, fmt.Errorf("failed to get listener %s: %w", resourceArn, err)
			}
			if listener == nil {
				return nil, fmt.Errorf("listener not found: %s", resourceArn)
			}

			// Remove tags
			if listener.Tags != nil {
				for _, key := range input.TagKeys {
					delete(listener.Tags, key)
				}
			}

			// Update listener
			if err := api.storage.ELBv2Store().UpdateListener(ctx, listener); err != nil {
				return nil, fmt.Errorf("failed to update listener tags: %w", err)
			}

		} else {
			return nil, fmt.Errorf("unsupported resource type: %s", resourceArn)
		}
	}

	return &generated_elbv2.RemoveTagsOutput{}, nil
}

func (api *ELBv2APIImpl) RemoveTrustStoreRevocations(ctx context.Context, input *generated_elbv2.RemoveTrustStoreRevocationsInput) (*generated_elbv2.RemoveTrustStoreRevocationsOutput, error) {
	return &generated_elbv2.RemoveTrustStoreRevocationsOutput{}, nil
}

func (api *ELBv2APIImpl) SetIpAddressType(ctx context.Context, input *generated_elbv2.SetIpAddressTypeInput) (*generated_elbv2.SetIpAddressTypeOutput, error) {
	return &generated_elbv2.SetIpAddressTypeOutput{}, nil
}

func (api *ELBv2APIImpl) SetRulePriorities(ctx context.Context, input *generated_elbv2.SetRulePrioritiesInput) (*generated_elbv2.SetRulePrioritiesOutput, error) {
	return &generated_elbv2.SetRulePrioritiesOutput{}, nil
}

func (api *ELBv2APIImpl) SetSecurityGroups(ctx context.Context, input *generated_elbv2.SetSecurityGroupsInput) (*generated_elbv2.SetSecurityGroupsOutput, error) {
	return &generated_elbv2.SetSecurityGroupsOutput{}, nil
}

func (api *ELBv2APIImpl) SetSubnets(ctx context.Context, input *generated_elbv2.SetSubnetsInput) (*generated_elbv2.SetSubnetsOutput, error) {
	return &generated_elbv2.SetSubnetsOutput{}, nil
}


func convertToLoadBalancer(lb *storage.ELBv2LoadBalancer) generated_elbv2.LoadBalancer {
	state := generated_elbv2.LoadBalancerStateEnumACTIVE
	if lb.State == "provisioning" {
		state = generated_elbv2.LoadBalancerStateEnumPROVISIONING
	}

	return generated_elbv2.LoadBalancer{
		LoadBalancerArn:       &lb.ARN,
		DNSName:               &lb.DNSName,
		CanonicalHostedZoneId: &lb.CanonicalHostedZoneID,
		CreatedTime:           &lb.CreatedAt,
		LoadBalancerName:      &lb.Name,
		Scheme:                (*generated_elbv2.LoadBalancerSchemeEnum)(&lb.Scheme),
		VpcId:                 &lb.VpcID,
		State:                 &generated_elbv2.LoadBalancerState{Code: &state},
		Type:                  (*generated_elbv2.LoadBalancerTypeEnum)(&lb.Type),
		IpAddressType:         (*generated_elbv2.IpAddressType)(&lb.IpAddressType),
		SecurityGroups:        lb.SecurityGroups,
		AvailabilityZones:     []generated_elbv2.AvailabilityZone{},
	}
}

// performHealthCheck performs a health check on a target
func (api *ELBv2APIImpl) performHealthCheck(ctx context.Context, target *storage.ELBv2Target, targetGroup *storage.ELBv2TargetGroup) string {
	// Use target group configuration for health check settings
	if targetGroup == nil || !targetGroup.HealthCheckEnabled {
		// If health check is disabled or target group is nil, consider target healthy
		return TargetHealthStateHealthy
	}
	
	// Determine health check port
	healthCheckPort := target.Port
	if targetGroup.HealthCheckPort != "" && targetGroup.HealthCheckPort != "traffic-port" {
		// Parse the health check port if it's not "traffic-port"
		fmt.Sscanf(targetGroup.HealthCheckPort, "%d", &healthCheckPort)
	}
	
	// Try Kubernetes-based health check first if integration is available
	if api.elbv2Integration != nil {
		healthState, err := api.elbv2Integration.CheckTargetHealthWithK8s(ctx, target.ID, healthCheckPort, targetGroup.ARN)
		if err != nil {
			logging.Debug("Kubernetes health check failed for target, falling back to legacy check", "targetId", target.ID, "error", err)
		} else {
			logging.Debug("Kubernetes health check for target returned", "targetId", target.ID, "healthState", healthState)
			return healthState
		}
	}
	
	// Fallback to legacy HTTP/TCP health check
	return api.performLegacyHealthCheck(ctx, target, targetGroup)
}

// performLegacyHealthCheck performs the original HTTP/TCP-based health check
func (api *ELBv2APIImpl) performLegacyHealthCheck(ctx context.Context, target *storage.ELBv2Target, targetGroup *storage.ELBv2TargetGroup) string {
	// Set timeout based on health check timeout
	timeout := time.Duration(targetGroup.HealthCheckTimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second // Default timeout
	}
	
	client := &http.Client{
		Timeout: timeout,
	}
	
	// Determine health check port
	healthCheckPort := target.Port
	if targetGroup.HealthCheckPort != "" && targetGroup.HealthCheckPort != "traffic-port" {
		// Parse the health check port if it's not "traffic-port"
		fmt.Sscanf(targetGroup.HealthCheckPort, "%d", &healthCheckPort)
	}
	
	// Construct health check URL based on protocol
	var url string
	protocol := targetGroup.HealthCheckProtocol
	if protocol == "" {
		protocol = targetGroup.Protocol // Use target group protocol if health check protocol not specified
	}
	
	path := targetGroup.HealthCheckPath
	if path == "" {
		path = "/" // Default path
	}
	
	switch protocol {
	case "HTTP":
		url = fmt.Sprintf("http://%s:%d%s", target.ID, healthCheckPort, path)
	case "HTTPS":
		url = fmt.Sprintf("https://%s:%d%s", target.ID, healthCheckPort, path)
	case "TCP", "TLS", "UDP", "TCP_UDP":
		// For TCP-based health checks, we'll do a simple connection check
		// This is a simplified implementation
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", target.ID, healthCheckPort), timeout)
		if err != nil {
			return TargetHealthStateUnhealthy
		}
		conn.Close()
		return TargetHealthStateHealthy
	default:
		// Default to HTTP
		url = fmt.Sprintf("http://%s:%d%s", target.ID, healthCheckPort, path)
	}
	
	// For HTTP/HTTPS health checks
	if protocol == "HTTP" || protocol == "HTTPS" {
		resp, err := client.Get(url)
		if err != nil {
			return TargetHealthStateUnhealthy
		}
		defer resp.Body.Close()
		
		// Check if response matches expected status codes
		if targetGroup.Matcher != "" {
			// Parse matcher (e.g., "200", "200-299", "200,202,301")
			if MatchesHealthCheckResponse(resp.StatusCode, targetGroup.Matcher) {
				return TargetHealthStateHealthy
			}
		} else {
			// Default: 200-299 is healthy
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return TargetHealthStateHealthy
			}
		}
	}
	
	return TargetHealthStateUnhealthy
}

// MatchesHealthCheckResponse checks if the status code matches the expected matcher pattern
func MatchesHealthCheckResponse(statusCode int, matcher string) bool {
	// Remove any whitespace
	matcher = strings.TrimSpace(matcher)
	
	// Handle comma-separated values (e.g., "200,202,301")
	if strings.Contains(matcher, ",") {
		codes := strings.Split(matcher, ",")
		for _, code := range codes {
			code = strings.TrimSpace(code)
			if expected, err := strconv.Atoi(code); err == nil && statusCode == expected {
				return true
			}
		}
		return false
	}
	
	// Handle range (e.g., "200-299")
	if strings.Contains(matcher, "-") {
		parts := strings.Split(matcher, "-")
		if len(parts) == 2 {
			min, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			max, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil {
				return statusCode >= min && statusCode <= max
			}
		}
		return false
	}
	
	// Handle single value (e.g., "200")
	if expected, err := strconv.Atoi(matcher); err == nil {
		return statusCode == expected
	}
	
	return false
}

// getHealthReason returns the reason for the health state
func (api *ELBv2APIImpl) getHealthReason(healthState string) string {
	switch healthState {
	case TargetHealthStateHealthy:
		return "Target.ResponseCodeMismatch"
	case TargetHealthStateUnhealthy:
		return "Target.FailedHealthChecks"
	case TargetHealthStateInitial:
		return "Target.NotRegistered"
	case TargetHealthStateRegistering:
		return "Target.RegistrationInProgress"
	case TargetHealthStateDeregistering:
		return "Target.DeregistrationInProgress"
	default:
		return "Target.InvalidState"
	}
}

// getHealthDescription returns the description for the health state
func (api *ELBv2APIImpl) getHealthDescription(healthState string) string {
	switch healthState {
	case TargetHealthStateHealthy:
		return "Health checks succeeded"
	case TargetHealthStateUnhealthy:
		return "Health checks failed"
	case TargetHealthStateInitial:
		return "Target registration is in progress"
	case TargetHealthStateRegistering:
		return "Target registration is in progress"
	case TargetHealthStateDeregistering:
		return "Target deregistration is in progress"
	default:
		return "Target is in an invalid state"
	}
}

// convertHealthStateToEnum converts internal health state to generated enum
func (api *ELBv2APIImpl) convertHealthStateToEnum(healthState string) generated_elbv2.TargetHealthStateEnum {
	switch healthState {
	case TargetHealthStateHealthy:
		return generated_elbv2.TargetHealthStateEnumHEALTHY
	case TargetHealthStateUnhealthy:
		return generated_elbv2.TargetHealthStateEnumUNHEALTHY
	case TargetHealthStateInitial:
		return generated_elbv2.TargetHealthStateEnumINITIAL
	case TargetHealthStateRegistering:
		return generated_elbv2.TargetHealthStateEnumUNUSED
	case TargetHealthStateDeregistering:
		return generated_elbv2.TargetHealthStateEnumDRAINING
	default:
		return generated_elbv2.TargetHealthStateEnumUNAVAILABLE
	}
}

// updateLoadBalancerToActive updates a load balancer state to active
func (api *ELBv2APIImpl) updateLoadBalancerToActive(ctx context.Context, arn string) error {
	lb, err := api.storage.ELBv2Store().GetLoadBalancer(ctx, arn)
	if err != nil {
		return err
	}
	
	if lb == nil {
		return fmt.Errorf("load balancer not found: %s", arn)
	}
	
	if lb.State != "provisioning" {
		return nil // Already active or in another state
	}
	
	lb.State = "active"
	lb.UpdatedAt = time.Now()
	
	return api.storage.ELBv2Store().UpdateLoadBalancer(ctx, lb)
}

// convertToTargetGroup converts storage target group to API response format
func (api *ELBv2APIImpl) convertToTargetGroup(tg *storage.ELBv2TargetGroup) generated_elbv2.TargetGroup {
	return generated_elbv2.TargetGroup{
		TargetGroupArn:       &tg.ARN,
		TargetGroupName:      &tg.Name,
		Protocol:             (*generated_elbv2.ProtocolEnum)(&tg.Protocol),
		Port:                 &tg.Port,
		VpcId:                &tg.VpcID,
		HealthCheckPath:      &tg.HealthCheckPath,
		HealthCheckProtocol:  (*generated_elbv2.ProtocolEnum)(&tg.HealthCheckProtocol),
		HealthCheckPort:      &tg.HealthCheckPort,
		HealthyThresholdCount: &tg.HealthyThresholdCount,
		UnhealthyThresholdCount: &tg.UnhealthyThresholdCount,
		HealthCheckTimeoutSeconds: &tg.HealthCheckTimeoutSeconds,
		HealthCheckIntervalSeconds: &tg.HealthCheckIntervalSeconds,
		TargetType:           (*generated_elbv2.TargetTypeEnum)(&tg.TargetType),
	}
}

// convertToListener converts storage listener to API response format
func (api *ELBv2APIImpl) convertToListener(listener *storage.ELBv2Listener) generated_elbv2.Listener {
	// Convert default actions - ensure we always have a non-nil slice
	defaultActions := make([]generated_elbv2.Action, 0)
	// TODO: Parse and convert stored default actions JSON

	return generated_elbv2.Listener{
		ListenerArn:      &listener.ARN,
		LoadBalancerArn:  &listener.LoadBalancerArn,
		Port:             &listener.Port,
		Protocol:         (*generated_elbv2.ProtocolEnum)(&listener.Protocol),
		DefaultActions:   defaultActions,
	}
}