package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
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

	// Check if this is a KECS-managed pod
	// Either it has kecs.dev/managed-by label or it's a service pod with kecs.dev/service label
	isKECSManaged := pod.Labels["kecs.dev/managed-by"] == "kecs" || pod.Labels["kecs.dev/service"] != ""

	if !isKECSManaged {
		// Not a KECS pod, allow without mutation
		logging.Debug("Pod not managed by KECS, skipping mutation",
			"pod", pod.Name,
			"namespace", pod.Namespace,
			"managed-by", pod.Labels["kecs.dev/managed-by"],
			"service", pod.Labels["kecs.dev/service"])
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	// Check if task ID already exists
	if _, exists := pod.Labels["kecs.dev/task-id"]; exists {
		// Task ID already set, allow without mutation
		logging.Debug("Task ID already exists, skipping mutation",
			"pod", pod.Name,
			"namespace", pod.Namespace,
			"taskId", pod.Labels["kecs.dev/task-id"])
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	// Generate patches for mutation
	var patches []patchOperation

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
