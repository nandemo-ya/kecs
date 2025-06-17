package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// Annotation keys
	InjectSidecarAnnotation = "kecs.io/inject-aws-proxy"
	LocalStackEndpointAnnotation = "kecs.io/localstack-endpoint"
	ProxyServicesAnnotation = "kecs.io/proxy-services"
	
	// Default values
	DefaultLocalStackEndpoint = "http://localstack.localstack.svc.cluster.local:4566"
	DefaultProxyServices = "s3,dynamodb,sqs,sns,ssm,secretsmanager,cloudwatch"
	DefaultProxyPort = 8080
	
	// Container and volume names
	ProxySidecarName = "aws-sdk-proxy"
	ProxyVolumeName = "aws-proxy-config"
)

// SidecarInjector handles automatic injection of AWS SDK proxy sidecar
type SidecarInjector struct {
	decoder       runtime.Decoder
	proxyImage    string
	localStackMgr LocalStackManager
}

// LocalStackManager interface for checking LocalStack status
type LocalStackManager interface {
	IsEnabled() bool
	GetEndpoint() string
	GetEnabledServices() []string
}

// NewSidecarInjector creates a new sidecar injector
func NewSidecarInjector(proxyImage string, localStackMgr LocalStackManager) *SidecarInjector {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	
	return &SidecarInjector{
		decoder:       decoder,
		proxyImage:    proxyImage,
		localStackMgr: localStackMgr,
	}
}

// Handle processes admission webhook requests
func (si *SidecarInjector) Handle(ctx context.Context, req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	klog.V(2).Infof("Handling admission request for %s/%s", req.Namespace, req.Name)
	
	// Decode the pod
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		klog.Errorf("Failed to decode pod: %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}
	
	// Check if sidecar injection is requested
	if !si.shouldInjectSidecar(&pod) {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}
	
	// Check if LocalStack is enabled
	if !si.localStackMgr.IsEnabled() {
		klog.Warning("Sidecar injection requested but LocalStack is not enabled")
		return &admissionv1.AdmissionResponse{
			Allowed: true,
			Result: &metav1.Status{
				Message: "LocalStack is not enabled, skipping sidecar injection",
			},
		}
	}
	
	// Create patches for sidecar injection
	patches := si.createSidecarPatches(&pod)
	
	// Convert patches to JSON
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		klog.Errorf("Failed to marshal patches: %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}
	
	// Return admission response with patches
	patchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: &patchType,
	}
}

// shouldInjectSidecar checks if sidecar should be injected
func (si *SidecarInjector) shouldInjectSidecar(pod *corev1.Pod) bool {
	// Check annotation
	if val, ok := pod.Annotations[InjectSidecarAnnotation]; ok {
		inject, err := strconv.ParseBool(val)
		if err != nil {
			klog.Warningf("Invalid value for %s annotation: %s", InjectSidecarAnnotation, val)
			return false
		}
		return inject
	}
	
	// Check if any container has AWS SDK environment variables
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if strings.HasPrefix(env.Name, "AWS_") {
				return true
			}
		}
	}
	
	return false
}

// createSidecarPatches creates JSON patches for sidecar injection
func (si *SidecarInjector) createSidecarPatches(pod *corev1.Pod) []map[string]interface{} {
	var patches []map[string]interface{}
	
	// Get configuration from annotations or defaults
	localstackEndpoint := si.getLocalStackEndpoint(pod)
	proxyServices := si.getProxyServices(pod)
	
	// Create sidecar container
	sidecar := si.createSidecarContainer(localstackEndpoint, proxyServices)
	
	// Add sidecar container
	patches = append(patches, map[string]interface{}{
		"op":    "add",
		"path":  "/spec/containers/-",
		"value": sidecar,
	})
	
	// Update main containers to use proxy
	for i := range pod.Spec.Containers {
		// Add AWS_ENDPOINT_URL environment variable
		patches = append(patches, map[string]interface{}{
			"op":   "add",
			"path": fmt.Sprintf("/spec/containers/%d/env/-", i),
			"value": map[string]string{
				"name":  "AWS_ENDPOINT_URL",
				"value": fmt.Sprintf("http://localhost:%d", DefaultProxyPort),
			},
		})
		
		// Add HTTP_PROXY for AWS SDK v1 compatibility
		patches = append(patches, map[string]interface{}{
			"op":   "add",
			"path": fmt.Sprintf("/spec/containers/%d/env/-", i),
			"value": map[string]string{
				"name":  "HTTPS_PROXY",
				"value": fmt.Sprintf("http://localhost:%d", DefaultProxyPort),
			},
		})
		
		// Add NO_PROXY to exclude non-AWS traffic
		patches = append(patches, map[string]interface{}{
			"op":   "add",
			"path": fmt.Sprintf("/spec/containers/%d/env/-", i),
			"value": map[string]string{
				"name":  "NO_PROXY",
				"value": "localhost,127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,.svc,.local",
			},
		})
	}
	
	// Add annotation to mark pod as injected
	patches = append(patches, map[string]interface{}{
		"op":   "add",
		"path": "/metadata/annotations/kecs.io~1sidecar-injected",
		"value": "true",
	})
	
	return patches
}

// createSidecarContainer creates the sidecar container spec
func (si *SidecarInjector) createSidecarContainer(localstackEndpoint, services string) corev1.Container {
	return corev1.Container{
		Name:  ProxySidecarName,
		Image: si.proxyImage,
		Ports: []corev1.ContainerPort{
			{
				Name:          "proxy",
				ContainerPort: int32(DefaultProxyPort),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "LOCALSTACK_ENDPOINT",
				Value: localstackEndpoint,
			},
			{
				Name:  "PROXY_SERVICES",
				Value: services,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/health",
					Port:   intstr.FromInt(DefaultProxyPort),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/health",
					Port:   intstr.FromInt(DefaultProxyPort),
				},
			},
			InitialDelaySeconds: 3,
			PeriodSeconds:       5,
		},
	}
}

// getLocalStackEndpoint gets the LocalStack endpoint from annotations or defaults
func (si *SidecarInjector) getLocalStackEndpoint(pod *corev1.Pod) string {
	if endpoint, ok := pod.Annotations[LocalStackEndpointAnnotation]; ok {
		return endpoint
	}
	
	if si.localStackMgr != nil {
		return si.localStackMgr.GetEndpoint()
	}
	
	return DefaultLocalStackEndpoint
}

// getProxyServices gets the services to proxy from annotations or defaults
func (si *SidecarInjector) getProxyServices(pod *corev1.Pod) string {
	if services, ok := pod.Annotations[ProxyServicesAnnotation]; ok {
		return services
	}
	
	if si.localStackMgr != nil {
		enabledServices := si.localStackMgr.GetEnabledServices()
		if len(enabledServices) > 0 {
			return strings.Join(enabledServices, ",")
		}
	}
	
	return DefaultProxyServices
}