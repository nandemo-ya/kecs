package proxy

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSidecarProxy_ShouldInjectSidecar(t *testing.T) {
	sp := NewSidecarProxy("http://localstack:4566")

	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected bool
	}{
		{
			name: "ECS task without annotations",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kecs.dev/task-id": "task-123",
					},
				},
			},
			expected: true, // Default to sidecar for ECS tasks
		},
		{
			name: "Explicitly enabled via annotation",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kecs.io/inject-aws-proxy": "true",
					},
				},
			},
			expected: true,
		},
		{
			name: "Explicitly disabled",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kecs.io/aws-proxy-enabled": "false",
					},
				},
			},
			expected: false,
		},
		{
			name: "Sidecar mode specified",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kecs.io/aws-proxy-mode": "sidecar",
					},
				},
			},
			expected: true,
		},
		{
			name: "Environment mode specified",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kecs.io/aws-proxy-mode": "environment",
					},
				},
			},
			expected: false,
		},
		{
			name: "Non-ECS pod without annotations",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "my-app",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sp.ShouldInjectSidecar(tt.pod)
			if result != tt.expected {
				t.Errorf("ShouldInjectSidecar() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSidecarProxy_CreateProxySidecar(t *testing.T) {
	sp := NewSidecarProxy("http://localstack:4566")
	sp.SetProxyImage("custom/aws-proxy:v1.0")

	tests := []struct {
		name            string
		pod             *corev1.Pod
		expectedImage   string
		expectedDebug   string
		expectedEnpoint string
	}{
		{
			name: "Basic sidecar",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			},
			expectedImage:   "custom/aws-proxy:v1.0",
			expectedDebug:   "false",
			expectedEnpoint: "http://localstack:4566",
		},
		{
			name: "Custom endpoint",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
					Annotations: map[string]string{
						"kecs.io/localstack-endpoint": "http://custom-localstack:9999",
					},
				},
			},
			expectedImage:   "custom/aws-proxy:v1.0",
			expectedDebug:   "false",
			expectedEnpoint: "http://custom-localstack:9999",
		},
		{
			name: "Debug enabled",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
					Annotations: map[string]string{
						"kecs.io/aws-proxy-debug": "true",
					},
				},
			},
			expectedImage:   "custom/aws-proxy:v1.0",
			expectedDebug:   "true",
			expectedEnpoint: "http://localstack:4566",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sidecar := sp.CreateProxySidecar(tt.pod)

			if sidecar.Name != "aws-proxy-sidecar" {
				t.Errorf("Expected sidecar name 'aws-proxy-sidecar', got %s", sidecar.Name)
			}

			if sidecar.Image != tt.expectedImage {
				t.Errorf("Expected image %s, got %s", tt.expectedImage, sidecar.Image)
			}

			// Check environment variables
			var foundEndpoint, foundDebug bool
			for _, env := range sidecar.Env {
				if env.Name == "LOCALSTACK_ENDPOINT" {
					foundEndpoint = true
					if env.Value != tt.expectedEnpoint {
						t.Errorf("Expected endpoint %s, got %s", tt.expectedEnpoint, env.Value)
					}
				}
				if env.Name == "DEBUG" {
					foundDebug = true
					if env.Value != tt.expectedDebug {
						t.Errorf("Expected debug %s, got %s", tt.expectedDebug, env.Value)
					}
				}
			}

			if !foundEndpoint {
				t.Error("LOCALSTACK_ENDPOINT environment variable not found")
			}
			if !foundDebug {
				t.Error("DEBUG environment variable not found")
			}

			// Check ports
			if len(sidecar.Ports) != 1 || sidecar.Ports[0].ContainerPort != 4566 {
				t.Error("Expected single port 4566")
			}

			// Check probes
			if sidecar.ReadinessProbe == nil || sidecar.LivenessProbe == nil {
				t.Error("Expected both readiness and liveness probes")
			}
		})
	}
}

func TestSidecarProxy_InjectSidecar(t *testing.T) {
	sp := NewSidecarProxy("http://localstack:4566")

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			Annotations: map[string]string{
				"kecs.io/inject-aws-proxy": "true",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "myapp:latest",
					Env: []corev1.EnvVar{
						{Name: "MY_ENV", Value: "value"},
					},
				},
			},
		},
	}

	err := sp.InjectSidecar(pod)
	if err != nil {
		t.Fatalf("InjectSidecar() error = %v", err)
	}

	// Check that sidecar was added
	if len(pod.Spec.Containers) != 2 {
		t.Errorf("Expected 2 containers, got %d", len(pod.Spec.Containers))
	}

	// Check that the sidecar is named correctly
	foundSidecar := false
	for _, container := range pod.Spec.Containers {
		if container.Name == "aws-proxy-sidecar" {
			foundSidecar = true
			break
		}
	}
	if !foundSidecar {
		t.Error("Sidecar container not found")
	}

	// Check that AWS environment variables were added to app container
	appContainer := pod.Spec.Containers[0]
	expectedEnvVars := map[string]string{
		"AWS_ENDPOINT_URL":                "http://localhost:4566",
		"AWS_ENDPOINT_URL_S3":             "http://localhost:4566",
		"AWS_ENDPOINT_URL_IAM":            "http://localhost:4566",
		"AWS_ENDPOINT_URL_LOGS":           "http://localhost:4566",
		"AWS_ENDPOINT_URL_SSM":            "http://localhost:4566",
		"AWS_ENDPOINT_URL_SECRETSMANAGER": "http://localhost:4566",
		"AWS_ENDPOINT_URL_ELB":            "http://localhost:4566",
		"MY_ENV":                          "value", // Original env var should be preserved
	}

	envMap := make(map[string]string)
	for _, env := range appContainer.Env {
		envMap[env.Name] = env.Value
	}

	for name, value := range expectedEnvVars {
		if envMap[name] != value {
			t.Errorf("Expected env var %s=%s, got %s", name, value, envMap[name])
		}
	}

	// Check annotation was added
	if pod.Annotations["kecs.io/aws-proxy-sidecar-injected"] != "true" {
		t.Error("Injection annotation not set")
	}
}

func TestSidecarProxy_InjectSidecar_ShouldNotInject(t *testing.T) {
	sp := NewSidecarProxy("http://localstack:4566")

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			Annotations: map[string]string{
				"kecs.io/aws-proxy-enabled": "false",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "myapp:latest",
				},
			},
		},
	}

	err := sp.InjectSidecar(pod)
	if err != nil {
		t.Fatalf("InjectSidecar() error = %v", err)
	}

	// Check that no sidecar was added
	if len(pod.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(pod.Spec.Containers))
	}

	// Check that no injection annotation was added
	if _, exists := pod.Annotations["kecs.io/aws-proxy-sidecar-injected"]; exists {
		t.Error("Injection annotation should not be set")
	}
}
