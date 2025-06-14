package converters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

func TestApplyResourceConstraints(t *testing.T) {
	converter := NewTaskConverter("us-east-1", "123456789012")

	tests := []struct {
		name           string
		pod            *corev1.Pod
		cpu            string
		memory         string
		expectedCPU    map[string]int64 // container name -> cpu millis
		expectedMemory map[string]int64 // container name -> memory MiB
	}{
		{
			name: "even distribution with no existing resources",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Resources: corev1.ResourceRequirements{}},
						{Name: "sidecar", Resources: corev1.ResourceRequirements{}},
					},
				},
			},
			cpu:    "2048", // 2 vCPU
			memory: "4096", // 4 GiB
			expectedCPU: map[string]int64{
				"app":     1000, // 1 vCPU
				"sidecar": 1000, // 1 vCPU
			},
			expectedMemory: map[string]int64{
				"app":     2048, // 2 GiB
				"sidecar": 2048, // 2 GiB
			},
		},
		{
			name: "proportional distribution with existing resources",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("750m"),
									corev1.ResourceMemory: resource.MustParse("1536Mi"),
								},
							},
						},
						{
							Name: "sidecar",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("250m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
				},
			},
			cpu:    "2048", // 2 vCPU
			memory: "4096", // 4 GiB
			expectedCPU: map[string]int64{
				"app":     1500, // 75% of 2000 millis
				"sidecar": 500,  // 25% of 2000 millis
			},
			expectedMemory: map[string]int64{
				"app":     3072, // 75% of 4096 MiB
				"sidecar": 1024, // 25% of 4096 MiB
			},
		},
		{
			name: "single container",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Resources: corev1.ResourceRequirements{}},
					},
				},
			},
			cpu:    "1024", // 1 vCPU
			memory: "2048", // 2 GiB
			expectedCPU: map[string]int64{
				"app": 1000, // All CPU
			},
			expectedMemory: map[string]int64{
				"app": 2048, // All memory
			},
		},
		{
			name: "no task-level constraints",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1024Mi"),
								},
							},
						},
					},
				},
			},
			cpu:    "",
			memory: "",
			expectedCPU: map[string]int64{
				"app": 500, // Unchanged
			},
			expectedMemory: map[string]int64{
				"app": 1024, // Unchanged
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply resource constraints
			converter.applyResourceConstraints(tt.pod, tt.cpu, tt.memory)

			// Verify CPU allocations
			for _, container := range tt.pod.Spec.Containers {
				expectedCPU, ok := tt.expectedCPU[container.Name]
				if !ok {
					t.Errorf("No expected CPU for container %s", container.Name)
					continue
				}

				actualCPU := int64(0)
				if container.Resources.Requests != nil {
					if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
						actualCPU = cpu.MilliValue()
					}
				}

				if actualCPU != expectedCPU {
					t.Errorf("Container %s: expected CPU %d millis, got %d millis",
						container.Name, expectedCPU, actualCPU)
				}

				// Verify limits match requests
				if container.Resources.Limits != nil {
					if cpuLimit, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
						if cpuLimit.MilliValue() != actualCPU {
							t.Errorf("Container %s: CPU limit %d doesn't match request %d",
								container.Name, cpuLimit.MilliValue(), actualCPU)
						}
					}
				}
			}

			// Verify memory allocations
			for _, container := range tt.pod.Spec.Containers {
				expectedMem, ok := tt.expectedMemory[container.Name]
				if !ok {
					t.Errorf("No expected memory for container %s", container.Name)
					continue
				}

				actualMem := int64(0)
				if container.Resources.Requests != nil {
					if mem, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
						actualMem = mem.Value() / (1024 * 1024) // Convert to MiB
					}
				}

				if actualMem != expectedMem {
					t.Errorf("Container %s: expected memory %d MiB, got %d MiB",
						container.Name, expectedMem, actualMem)
				}

				// Verify limits match requests
				if container.Resources.Limits != nil {
					if memLimit, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
						actualMemLimit := memLimit.Value() / (1024 * 1024)
						if actualMemLimit != actualMem {
							t.Errorf("Container %s: memory limit %d MiB doesn't match request %d MiB",
								container.Name, actualMemLimit, actualMem)
						}
					}
				}
			}
		})
	}
}

func TestApplyContainerOverride(t *testing.T) {
	converter := NewTaskConverter("us-east-1", "123456789012")

	tests := []struct {
		name            string
		container       *corev1.Container
		override        *types.ContainerOverride
		expectedCommand []string
		expectedEnv     map[string]string
		expectedCPU     int64 // millis
		expectedMemory  int64 // MiB
	}{
		{
			name: "override command",
			container: &corev1.Container{
				Name:    "app",
				Command: []string{"original-cmd"},
			},
			override: &types.ContainerOverride{
				Name:    ptr.To("app"),
				Command: []string{"new-cmd", "--arg"},
			},
			expectedCommand: []string{"new-cmd", "--arg"},
		},
		{
			name: "override environment variables",
			container: &corev1.Container{
				Name: "app",
				Env: []corev1.EnvVar{
					{Name: "FOO", Value: "original"},
					{Name: "BAR", Value: "baz"},
				},
			},
			override: &types.ContainerOverride{
				Name: ptr.To("app"),
				Environment: []types.KeyValuePair{
					{Name: ptr.To("FOO"), Value: ptr.To("overridden")},
					{Name: ptr.To("NEW"), Value: ptr.To("value")},
				},
			},
			expectedEnv: map[string]string{
				"FOO": "overridden",
				"BAR": "baz",
				"NEW": "value",
			},
		},
		{
			name: "override CPU and memory",
			container: &corev1.Container{
				Name: "app",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1024Mi"),
					},
				},
			},
			override: &types.ContainerOverride{
				Name:   ptr.To("app"),
				Cpu:    ptr.To(1024), // 1 vCPU
				Memory: ptr.To(2048), // 2 GiB
			},
			expectedCPU:    1000, // 1024 * 1000 / 1024
			expectedMemory: 2048,
		},
		{
			name: "override memory reservation only",
			container: &corev1.Container{
				Name:      "app",
				Resources: corev1.ResourceRequirements{},
			},
			override: &types.ContainerOverride{
				Name:              ptr.To("app"),
				MemoryReservation: ptr.To(512), // 512 MiB
			},
			expectedMemory: 512,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of the container to avoid mutations
			container := tt.container.DeepCopy()

			// Apply override
			converter.applyContainerOverride(container, tt.override)

			// Verify command
			if tt.expectedCommand != nil {
				if len(container.Command) != len(tt.expectedCommand) {
					t.Errorf("Expected command length %d, got %d",
						len(tt.expectedCommand), len(container.Command))
				} else {
					for i, cmd := range tt.expectedCommand {
						if container.Command[i] != cmd {
							t.Errorf("Expected command[%d] = %s, got %s",
								i, cmd, container.Command[i])
						}
					}
				}
			}

			// Verify environment variables
			if tt.expectedEnv != nil {
				actualEnv := make(map[string]string)
				for _, env := range container.Env {
					actualEnv[env.Name] = env.Value
				}

				for name, expectedValue := range tt.expectedEnv {
					if actualValue, ok := actualEnv[name]; !ok {
						t.Errorf("Expected env var %s not found", name)
					} else if actualValue != expectedValue {
						t.Errorf("Expected env var %s = %s, got %s",
							name, expectedValue, actualValue)
					}
				}

				if len(actualEnv) != len(tt.expectedEnv) {
					t.Errorf("Expected %d env vars, got %d",
						len(tt.expectedEnv), len(actualEnv))
				}
			}

			// Verify CPU
			if tt.expectedCPU > 0 {
				actualCPU := int64(0)
				if container.Resources.Requests != nil {
					if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
						actualCPU = cpu.MilliValue()
					}
				}

				if actualCPU != tt.expectedCPU {
					t.Errorf("Expected CPU %d millis, got %d millis",
						tt.expectedCPU, actualCPU)
				}
			}

			// Verify memory
			if tt.expectedMemory > 0 {
				actualMem := int64(0)
				if container.Resources.Requests != nil {
					if mem, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
						actualMem = mem.Value() / (1024 * 1024)
					}
				}

				if actualMem != tt.expectedMemory {
					t.Errorf("Expected memory %d MiB, got %d MiB",
						tt.expectedMemory, actualMem)
				}
			}
		})
	}
}

func TestApplyPlacementConstraints(t *testing.T) {
	converter := NewTaskConverter("us-east-1", "123456789012")

	tests := []struct {
		name                 string
		pod                  *corev1.Pod
		constraints          []types.PlacementConstraint
		expectedNodeSelector map[string]string
		expectedAffinity     bool
		expectedAntiAffinity bool
	}{
		{
			name: "memberOf with exact match",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kecs.dev/task-family": "my-task",
					},
				},
				Spec: corev1.PodSpec{},
			},
			constraints: []types.PlacementConstraint{
				{
					Type:       ptr.To("memberOf"),
					Expression: ptr.To("attribute:ecs.instance-type == t2.micro"),
				},
			},
			expectedNodeSelector: map[string]string{
				"node.kubernetes.io/instance-type": "t2.micro",
			},
		},
		{
			name: "memberOf with regex",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kecs.dev/task-family": "my-task",
					},
				},
				Spec: corev1.PodSpec{},
			},
			constraints: []types.PlacementConstraint{
				{
					Type:       ptr.To("memberOf"),
					Expression: ptr.To("attribute:ecs.instance-type =~ t2.*"),
				},
			},
			expectedAffinity: true,
		},
		{
			name: "memberOf with in list",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kecs.dev/task-family": "my-task",
					},
				},
				Spec: corev1.PodSpec{},
			},
			constraints: []types.PlacementConstraint{
				{
					Type:       ptr.To("memberOf"),
					Expression: ptr.To("attribute:ecs.availability-zone in [us-east-1a, us-east-1b]"),
				},
			},
			expectedAffinity: true,
		},
		{
			name: "distinctInstance constraint",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kecs.dev/task-family": "my-task",
					},
				},
				Spec: corev1.PodSpec{},
			},
			constraints: []types.PlacementConstraint{
				{
					Type: ptr.To("distinctInstance"),
				},
			},
			expectedAntiAffinity: true,
		},
		{
			name: "multiple constraints",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kecs.dev/task-family": "my-task",
					},
				},
				Spec: corev1.PodSpec{},
			},
			constraints: []types.PlacementConstraint{
				{
					Type:       ptr.To("memberOf"),
					Expression: ptr.To("attribute:ecs.instance-type == t2.small"),
				},
				{
					Type: ptr.To("distinctInstance"),
				},
			},
			expectedNodeSelector: map[string]string{
				"node.kubernetes.io/instance-type": "t2.small",
			},
			expectedAntiAffinity: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply placement constraints
			converter.applyPlacementConstraints(tt.pod, tt.constraints)

			// Verify node selector
			if tt.expectedNodeSelector != nil {
				if tt.pod.Spec.NodeSelector == nil {
					t.Error("Expected node selector to be set")
				} else {
					for key, expectedValue := range tt.expectedNodeSelector {
						if actualValue, ok := tt.pod.Spec.NodeSelector[key]; !ok {
							t.Errorf("Expected node selector %s not found", key)
						} else if actualValue != expectedValue {
							t.Errorf("Expected node selector %s = %s, got %s",
								key, expectedValue, actualValue)
						}
					}
				}
			}

			// Verify affinity
			if tt.expectedAffinity {
				if tt.pod.Spec.Affinity == nil || tt.pod.Spec.Affinity.NodeAffinity == nil {
					t.Error("Expected node affinity to be set")
				}
			}

			// Verify anti-affinity
			if tt.expectedAntiAffinity {
				if tt.pod.Spec.Affinity == nil || tt.pod.Spec.Affinity.PodAntiAffinity == nil {
					t.Error("Expected pod anti-affinity to be set")
				} else {
					// Verify it has preferred rules
					if len(tt.pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) == 0 {
						t.Error("Expected pod anti-affinity preferred rules")
					}
				}
			}
		})
	}
}

func TestParseValueList(t *testing.T) {
	converter := NewTaskConverter("us-east-1", "123456789012")

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple list",
			input:    "[us-east-1a, us-east-1b, us-east-1c]",
			expected: []string{"us-east-1a", "us-east-1b", "us-east-1c"},
		},
		{
			name:     "list with extra spaces",
			input:    "[ value1 ,  value2  , value3 ]",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "empty list",
			input:    "[]",
			expected: []string{},
		},
		{
			name:     "single value",
			input:    "[single-value]",
			expected: []string{"single-value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.parseValueList(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d values, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected value[%d] = %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestExpandRegexPattern(t *testing.T) {
	converter := NewTaskConverter("us-east-1", "123456789012")

	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:     "t2 instance types",
			pattern:  "t2.*",
			expected: []string{"t2.micro", "t2.small", "t2.medium", "t2.large", "t2.xlarge", "t2.2xlarge"},
		},
		{
			name:     "m5 instance types",
			pattern:  "m5.*",
			expected: []string{"m5.large", "m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m5.16xlarge", "m5.24xlarge"},
		},
		{
			name:     "unknown pattern",
			pattern:  "custom-pattern",
			expected: []string{"custom-pattern"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.expandRegexPattern(tt.pattern)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d values, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected value[%d] = %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestConvertECSAttributeToK8sLabel(t *testing.T) {
	converter := NewTaskConverter("us-east-1", "123456789012")

	tests := []struct {
		name      string
		attribute string
		expected  string
	}{
		{
			name:      "instance type",
			attribute: "ecs.instance-type",
			expected:  "node.kubernetes.io/instance-type",
		},
		{
			name:      "availability zone",
			attribute: "ecs.availability-zone",
			expected:  "topology.kubernetes.io/zone",
		},
		{
			name:      "OS type",
			attribute: "ecs.os-type",
			expected:  "kubernetes.io/os",
		},
		{
			name:      "CPU architecture",
			attribute: "ecs.cpu-architecture",
			expected:  "kubernetes.io/arch",
		},
		{
			name:      "custom attribute",
			attribute: "custom.attribute:name",
			expected:  "kecs.dev/custom-attribute-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertECSAttributeToK8sLabel(tt.attribute)
			assert.Equal(t, tt.expected, result)
		})
	}
}
