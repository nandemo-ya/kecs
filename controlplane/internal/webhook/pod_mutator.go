package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

	// Only process KECS-managed pods
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	if pod.Labels["kecs.dev/managed-by"] != "kecs" {
		// Not a KECS pod, allow without mutation
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	// Check if task ID already exists
	if _, exists := pod.Labels["kecs.dev/task-id"]; exists {
		// Task ID already set, allow without mutation
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

		// Get cluster name from label or namespace
		clusterName := "default"
		if cluster, ok := pod.Labels["kecs.dev/cluster"]; ok {
			clusterName = cluster
		}

		// Create task record in storage
		if m.storage != nil && m.storage.TaskStore() != nil {
			task := &storage.Task{
				ID:                generateTaskIDForStorage(taskID),
				ARN:               fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", m.region, m.accountID, clusterName, taskID),
				ClusterARN:        fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", m.region, m.accountID, clusterName),
				TaskDefinitionARN: "", // Will be updated by task sync
				LaunchType:        "FARGATE",
				LastStatus:        "PENDING",
				DesiredStatus:     "RUNNING",
				StartedBy:         fmt.Sprintf("ecs-svc/%s", serviceName),
				CreatedAt:         time.Now(),
				Version:           1,
				Namespace:         pod.Namespace,
				// PodName will be set after pod is created
			}

			ctx := context.Background()
			if err := m.storage.TaskStore().Create(ctx, task); err != nil {
				logging.Warn("Failed to create task record for service pod",
					"taskId", taskID,
					"service", serviceName,
					"error", err)
				// Don't fail the pod creation, just log the error
			} else {
				logging.Debug("Created task record for service pod",
					"taskId", taskID,
					"service", serviceName)
			}
		}
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

// generateTaskIDForStorage generates a storage-friendly task ID
func generateTaskIDForStorage(taskID string) string {
	// For storage, we can use the same ID
	return taskID
}
