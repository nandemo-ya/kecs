package admission

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// WebhookServer serves admission webhook requests
type WebhookServer struct {
	server       *http.Server
	sidecarInj   *SidecarInjector
	tlsConfig    *tls.Config
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(port int, sidecarInj *SidecarInjector, tlsConfig *tls.Config) *WebhookServer {
	ws := &WebhookServer{
		sidecarInj: sidecarInj,
		tlsConfig:  tlsConfig,
	}
	
	mux := http.NewServeMux()
	mux.HandleFunc("/inject", ws.handleInject)
	mux.HandleFunc("/health", ws.handleHealth)
	
	ws.server = &http.Server{
		Addr:      fmt.Sprintf(":%d", port),
		Handler:   mux,
		TLSConfig: tlsConfig,
	}
	
	return ws
}

// Start starts the webhook server
func (ws *WebhookServer) Start(ctx context.Context) error {
	klog.Infof("Starting webhook server on %s", ws.server.Addr)
	
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := ws.server.Shutdown(shutdownCtx); err != nil {
			klog.Errorf("Error shutting down webhook server: %v", err)
		}
	}()
	
	if err := ws.server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("webhook server error: %w", err)
	}
	
	return nil
}

// Stop stops the webhook server
func (ws *WebhookServer) Stop(ctx context.Context) error {
	return ws.server.Shutdown(ctx)
}

// handleInject handles sidecar injection requests
func (ws *WebhookServer) handleInject(w http.ResponseWriter, r *http.Request) {
	klog.V(2).Info("Handling injection request")
	
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("Failed to read request body: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	
	// Decode admission review
	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		klog.Errorf("Failed to decode admission review: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Handle the request
	admissionResponse := ws.sidecarInj.Handle(r.Context(), admissionReview.Request)
	admissionResponse.UID = admissionReview.Request.UID
	
	// Create response
	responseReview := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: admissionResponse,
	}
	
	// Encode response
	responseBytes, err := json.Marshal(responseReview)
	if err != nil {
		klog.Errorf("Failed to encode admission review: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Write response
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(responseBytes); err != nil {
		klog.Errorf("Failed to write response: %v", err)
	}
}

// handleHealth handles health check requests
func (ws *WebhookServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}