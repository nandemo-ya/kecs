package proxy

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// WebhookConfig contains configuration for the webhook server
type WebhookConfig struct {
	Port        int
	CertDir     string
	ServiceName string
	Namespace   string
}

// WebhookServer handles admission webhook requests
type WebhookServer struct {
	server      *http.Server
	kubeClient  kubernetes.Interface
	config      *WebhookConfig
	envProxy    *EnvironmentVariableProxy
	deserializer runtime.Decoder
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(kubeClient kubernetes.Interface, config *WebhookConfig, envProxy *EnvironmentVariableProxy) (*WebhookServer, error) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
	
	return &WebhookServer{
		kubeClient:   kubeClient,
		config:       config,
		envProxy:     envProxy,
		deserializer: serializer.NewCodecFactory(scheme).UniversalDeserializer(),
	}, nil
}

// Start starts the webhook server
func (ws *WebhookServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", ws.handleMutate)
	mux.HandleFunc("/health", ws.handleHealth)

	// Load TLS certificates
	certPath := fmt.Sprintf("%s/tls.crt", ws.config.CertDir)
	keyPath := fmt.Sprintf("%s/tls.key", ws.config.CertDir)

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	ws.server = &http.Server{
		Addr:      fmt.Sprintf(":%d", ws.config.Port),
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
		Handler:   mux,
	}

	go func() {
		klog.Infof("Starting webhook server on port %d", ws.config.Port)
		if err := ws.server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			klog.Errorf("Failed to start webhook server: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return ws.Stop()
}

// Stop stops the webhook server
func (ws *WebhookServer) Stop() error {
	if ws.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return ws.server.Shutdown(ctx)
	}
	return nil
}

// handleMutate handles the mutate webhook requests
func (ws *WebhookServer) handleMutate(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		klog.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Decode the admission review request
	var admissionReview admissionv1.AdmissionReview
	if _, _, err := ws.deserializer.Decode(body, nil, &admissionReview); err != nil {
		klog.Errorf("Failed to decode admission review: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract the pod from the request
	raw := admissionReview.Request.Object.Raw
	pod := corev1.Pod{}
	if _, _, err := ws.deserializer.Decode(raw, nil, &pod); err != nil {
		klog.Errorf("Failed to decode pod: %v", err)
		sendAdmissionResponse(w, &admissionReview, false, err.Error())
		return
	}

	// Inject environment variables
	patches, err := ws.envProxy.InjectEnvironmentVariables(&pod)
	if err != nil {
		klog.Errorf("Failed to inject environment variables: %v", err)
		sendAdmissionResponse(w, &admissionReview, false, err.Error())
		return
	}

	// Create the admission response
	var patchBytes []byte
	var patchType *admissionv1.PatchType
	if len(patches) > 0 {
		patchBytes, err = CreatePatch(patches)
		if err != nil {
			klog.Errorf("Failed to create patch: %v", err)
			sendAdmissionResponse(w, &admissionReview, false, err.Error())
			return
		}
		pt := admissionv1.PatchTypeJSONPatch
		patchType = &pt
	}

	// Send the admission response
	sendAdmissionResponseWithPatch(w, &admissionReview, true, "", patchBytes, patchType)
}

// handleHealth handles health check requests
func (ws *WebhookServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// sendAdmissionResponse sends an admission response
func sendAdmissionResponse(w http.ResponseWriter, ar *admissionv1.AdmissionReview, allowed bool, message string) {
	sendAdmissionResponseWithPatch(w, ar, allowed, message, nil, nil)
}

// sendAdmissionResponseWithPatch sends an admission response with optional patch
func sendAdmissionResponseWithPatch(w http.ResponseWriter, ar *admissionv1.AdmissionReview, allowed bool, message string, patch []byte, patchType *admissionv1.PatchType) {
	response := &admissionv1.AdmissionResponse{
		UID:     ar.Request.UID,
		Allowed: allowed,
	}

	if message != "" {
		response.Result = &metav1.Status{
			Message: message,
		}
	}

	if len(patch) > 0 {
		response.Patch = patch
		response.PatchType = patchType
	}

	ar.Response = response
	ar.Request = nil

	responseBytes, err := json.Marshal(ar)
	if err != nil {
		klog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)
}