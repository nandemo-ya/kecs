package api

import (
	"testing"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestELBv2APIImpl_convertHealthStateToEnum(t *testing.T) {
	api := &ELBv2APIImpl{}

	tests := []struct {
		name        string
		healthState string
		expected    generated_elbv2.TargetHealthStateEnum
	}{
		{
			name:        "healthy state",
			healthState: TargetHealthStateHealthy,
			expected:    generated_elbv2.TargetHealthStateEnumHEALTHY,
		},
		{
			name:        "unhealthy state",
			healthState: TargetHealthStateUnhealthy,
			expected:    generated_elbv2.TargetHealthStateEnumUNHEALTHY,
		},
		{
			name:        "initial state",
			healthState: TargetHealthStateInitial,
			expected:    generated_elbv2.TargetHealthStateEnumINITIAL,
		},
		{
			name:        "registering state",
			healthState: TargetHealthStateRegistering,
			expected:    generated_elbv2.TargetHealthStateEnumUNUSED,
		},
		{
			name:        "deregistering state",
			healthState: TargetHealthStateDeregistering,
			expected:    generated_elbv2.TargetHealthStateEnumDRAINING,
		},
		{
			name:        "unknown state",
			healthState: "unknown",
			expected:    generated_elbv2.TargetHealthStateEnumUNAVAILABLE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.convertHealthStateToEnum(tt.healthState)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestELBv2APIImpl_getHealthReason(t *testing.T) {
	api := &ELBv2APIImpl{}

	tests := []struct {
		name        string
		healthState string
		expected    string
	}{
		{
			name:        "healthy state",
			healthState: TargetHealthStateHealthy,
			expected:    "Target.ResponseCodeMismatch",
		},
		{
			name:        "unhealthy state",
			healthState: TargetHealthStateUnhealthy,
			expected:    "Target.FailedHealthChecks",
		},
		{
			name:        "initial state",
			healthState: TargetHealthStateInitial,
			expected:    "Target.NotRegistered",
		},
		{
			name:        "registering state",
			healthState: TargetHealthStateRegistering,
			expected:    "Target.RegistrationInProgress",
		},
		{
			name:        "deregistering state",
			healthState: TargetHealthStateDeregistering,
			expected:    "Target.DeregistrationInProgress",
		},
		{
			name:        "unknown state",
			healthState: "unknown",
			expected:    "Target.InvalidState",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.getHealthReason(tt.healthState)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestELBv2APIImpl_getHealthDescription(t *testing.T) {
	api := &ELBv2APIImpl{}

	tests := []struct {
		name        string
		healthState string
		expected    string
	}{
		{
			name:        "healthy state",
			healthState: TargetHealthStateHealthy,
			expected:    "Health checks succeeded",
		},
		{
			name:        "unhealthy state",
			healthState: TargetHealthStateUnhealthy,
			expected:    "Health checks failed",
		},
		{
			name:        "initial state",
			healthState: TargetHealthStateInitial,
			expected:    "Target registration is in progress",
		},
		{
			name:        "registering state",
			healthState: TargetHealthStateRegistering,
			expected:    "Target registration is in progress",
		},
		{
			name:        "deregistering state",
			healthState: TargetHealthStateDeregistering,
			expected:    "Target deregistration is in progress",
		},
		{
			name:        "unknown state",
			healthState: "unknown",
			expected:    "Target is in an invalid state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.getHealthDescription(tt.healthState)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestELBv2APIImpl_convertToTargetGroup(t *testing.T) {
	api := &ELBv2APIImpl{}

	tg := &storage.ELBv2TargetGroup{
		ARN:      "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/123456",
		Name:     "my-targets",
		Protocol: "HTTP",
		Port:     80,
		VpcID:    "vpc-123456",
		HealthCheckPath: "/health",
		HealthCheckProtocol: "HTTP",
		HealthCheckPort: "traffic-port",
		HealthyThresholdCount: 5,
		UnhealthyThresholdCount: 2,
		HealthCheckTimeoutSeconds: 5,
		HealthCheckIntervalSeconds: 30,
		TargetType: "ip",
	}

	result := api.convertToTargetGroup(tg)

	assert.Equal(t, &tg.ARN, result.TargetGroupArn)
	assert.Equal(t, &tg.Name, result.TargetGroupName)
	assert.Equal(t, (*generated_elbv2.ProtocolEnum)(&tg.Protocol), result.Protocol)
	assert.Equal(t, &tg.Port, result.Port)
	assert.Equal(t, &tg.VpcID, result.VpcId)
	assert.Equal(t, &tg.HealthCheckPath, result.HealthCheckPath)
	assert.Equal(t, (*generated_elbv2.ProtocolEnum)(&tg.HealthCheckProtocol), result.HealthCheckProtocol)
	assert.Equal(t, &tg.HealthCheckPort, result.HealthCheckPort)
	assert.Equal(t, &tg.HealthyThresholdCount, result.HealthyThresholdCount)
	assert.Equal(t, &tg.UnhealthyThresholdCount, result.UnhealthyThresholdCount)
	assert.Equal(t, &tg.HealthCheckTimeoutSeconds, result.HealthCheckTimeoutSeconds)
	assert.Equal(t, &tg.HealthCheckIntervalSeconds, result.HealthCheckIntervalSeconds)
	assert.Equal(t, (*generated_elbv2.TargetTypeEnum)(&tg.TargetType), result.TargetType)
}

func TestELBv2APIImpl_convertToListener(t *testing.T) {
	api := &ELBv2APIImpl{}

	listener := &storage.ELBv2Listener{
		ARN:             "arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-load-balancer/50dc6c495c0c9188/f2f7dc8efc522ab2",
		LoadBalancerArn: "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-load-balancer/50dc6c495c0c9188",
		Port:            80,
		Protocol:        "HTTP",
	}

	result := api.convertToListener(listener)

	assert.Equal(t, &listener.ARN, result.ListenerArn)
	assert.Equal(t, &listener.LoadBalancerArn, result.LoadBalancerArn)
	assert.Equal(t, &listener.Port, result.Port)
	assert.Equal(t, (*generated_elbv2.ProtocolEnum)(&listener.Protocol), result.Protocol)
	assert.NotNil(t, result.DefaultActions)
}