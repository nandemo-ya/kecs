package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// PodMutator handles pod mutation requests
type PodMutator struct {
	storage   storage.Storage
	decoder   runtime.Decoder
	region    string
	accountID string
}

// NewPodMutator creates a new pod mutator
func NewPodMutator(storage storage.Storage, region, accountID string) *PodMutator {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)

	return &PodMutator{
		storage:   storage,
		decoder:   serializer.NewCodecFactory(scheme).UniversalDeserializer(),
		region:    region,
		accountID: accountID,
	}
}

// Handle processes admission requests for pod mutations
func (m *PodMutator) Handle(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// Verify content type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		logging.Error("Invalid content type", "contentType", contentType)
		http.Error(w, "invalid Content-Type, expecting application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Parse admission review
	var admissionReview admissionv1.AdmissionReview
	if _, _, err := m.decoder.Decode(body, nil, &admissionReview); err != nil {
		logging.Error("Failed to decode admission review", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process the request
	response := m.mutate(admissionReview.Request)
	response.UID = admissionReview.Request.UID

	// Create admission review response
	admissionReviewResponse := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: response,
	}

	// Write response
	resp, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		logging.Error("Failed to marshal admission review response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(resp); err != nil {
		logging.Error("Failed to write response", "error", err)
	}
}

// mutate performs the actual pod mutation
func (m *PodMutator) mutate(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	// Debug log to track webhook invocations
	logging.Info("Webhook mutate called",
		"namespace", req.Namespace,
		"name", req.Name,
		"operation", req.Operation,
		"uid", req.UID)

	// Parse pod from request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		logging.Error("Failed to unmarshal pod", "error", err)
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	// Debug log pod labels
	logging.Debug("Pod labels before processing",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"labels", pod.Labels)

	// Only process KECS-managed pods
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	// Log pod labels for debugging
	logging.Info("Pod labels in webhook (before isKECSManaged check)",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"labels", pod.Labels)

	// Check if this is a KECS-managed pod
	// Either it has kecs.dev/managed-by label or it's a service pod with kecs.dev/service label
	isKECSManaged := pod.Labels["kecs.dev/managed-by"] == "kecs" || pod.Labels["kecs.dev/service"] != ""

	logging.Info("isKECSManaged check result",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"isKECSManaged", isKECSManaged,
		"managed-by", pod.Labels["kecs.dev/managed-by"],
		"service", pod.Labels["kecs.dev/service"])

	if !isKECSManaged {
		// Not a KECS pod, allow without mutation
		logging.Info("Pod not managed by KECS, skipping mutation",
			"pod", pod.Name,
			"namespace", pod.Namespace,
			"managed-by", pod.Labels["kecs.dev/managed-by"],
			"service", pod.Labels["kecs.dev/service"])
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	// Generate patches for mutation
	var patches []patchOperation

	// Log pod labels for debugging
	logging.Info("Pod labels in webhook (after isKECSManaged check)", "pod", pod.Name, "namespace", pod.Namespace, "labels", pod.Labels)

	// Inject AWS environment variables for ECS tasks
	// This must be done BEFORE the task ID check to ensure environment variables are injected
	if _, isECSTask := pod.Labels["ecs.task.definition.family"]; isECSTask {
		logging.Info("ECS task detected, adding AWS environment variables",
			"pod", pod.Name,
			"namespace", pod.Namespace,
			"family", pod.Labels["ecs.task.definition.family"])

		// AWS environment variables to inject
		awsEnvVars := []corev1.EnvVar{
			{Name: "AWS_ENDPOINT_URL", Value: "http://localstack.kecs-system.svc.cluster.local:4566"},
			{Name: "AWS_ACCESS_KEY_ID", Value: "test"},
			{Name: "AWS_SECRET_ACCESS_KEY", Value: "test"},
			{Name: "AWS_DEFAULT_REGION", Value: "us-east-1"},
			{Name: "AWS_REGION", Value: "us-east-1"},
		}

		// Add environment variables to all containers
		for containerIdx := range pod.Spec.Containers {
			containerPath := "/spec/containers/" + strconv.Itoa(containerIdx)

			// Check if env array exists
			if pod.Spec.Containers[containerIdx].Env == nil {
				// Create env array with all AWS variables
				patches = append(patches, patchOperation{
					Op:    "add",
					Path:  containerPath + "/env",
					Value: awsEnvVars,
				})
			} else {
				// Add to existing env array
				for _, envVar := range awsEnvVars {
					// Check if the env var already exists
					exists := false
					for _, existing := range pod.Spec.Containers[containerIdx].Env {
						if existing.Name == envVar.Name {
							exists = true
							break
						}
					}

					if !exists {
						patches = append(patches, patchOperation{
							Op:    "add",
							Path:  containerPath + "/env/-",
							Value: envVar,
						})
					}
				}
			}
		}

		logging.Info("Added AWS environment variable patches (AWS_ENDPOINT_URL + credentials)", "patchCount", len(patches))
	}

	// Check if task ID already exists (only skip for non-service pods)
	if _, exists := pod.Labels["kecs.dev/task-id"]; exists {
		_, isServicePod := pod.Labels["kecs.dev/service"]
		if !isServicePod {
			// Task ID already set for non-service pod, allow without further mutation
			logging.Info("Task ID already exists for non-service pod, skipping task ID generation",
				"pod", pod.Name,
				"namespace", pod.Namespace,
				"taskId", pod.Labels["kecs.dev/task-id"],
				"patchCount", len(patches))

			// Return with any patches we already created (e.g., env vars)
			if len(patches) == 0 {
				return &admissionv1.AdmissionResponse{
					Allowed: true,
				}
			}
			// Continue to apply patches
			goto applyPatches
		}
	}

	// For service-managed pods, generate and add task ID
	if serviceName, ok := pod.Labels["kecs.dev/service"]; ok {
		// Generate a unique task ID
		taskID := generateTaskID()

		logging.Info("Adding task ID to service pod",
			"service", serviceName,
			"taskId", taskID,
			"namespace", pod.Namespace)

		// Add task ID label
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/metadata/labels/kecs.dev~1task-id",
			Value: taskID,
		})

		// Don't create task here - let ServiceManager handle it to avoid duplicates
		// The webhook only adds the task ID label, and ServiceManager will create
		// the actual task record when it processes the pod
		logging.Debug("Added task ID label to pod, task will be created by ServiceManager",
			"taskId", taskID,
			"service", serviceName)
	}

applyPatches:
	// If no patches needed, allow without mutation
	if len(patches) == 0 {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	// Create patch response
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		logging.Error("Failed to marshal patches", "error", err)
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	patchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}
}

// patchOperation represents a JSON patch operation
type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	// Generate a UUID and remove hyphens to match ECS task ID format
	id := uuid.New().String()
	return strings.ReplaceAll(id, "-", "")
}
