package webhook

import (
	"context"
	"fmt"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// WebhookRegistrar handles webhook registration with Kubernetes
type WebhookRegistrar struct {
	clientset kubernetes.Interface
	namespace string
	service   string
	port      int32
}

// NewWebhookRegistrar creates a new webhook registrar
func NewWebhookRegistrar(clientset kubernetes.Interface, namespace, service string, port int32) *WebhookRegistrar {
	return &WebhookRegistrar{
		clientset: clientset,
		namespace: namespace,
		service:   service,
		port:      port,
	}
}

// Register registers the mutating webhook configuration
func (r *WebhookRegistrar) Register(ctx context.Context, caBundle []byte) error {
	webhookName := "kecs-pod-mutator"
	configName := "kecs-webhook-config"

	// Define failure policy
	failurePolicy := admissionv1.Ignore // Use Ignore to prevent blocking pod creation if webhook is down
	sideEffects := admissionv1.SideEffectClassNone
	admissionReviewVersions := []string{"v1", "v1beta1"}

	// Create webhook configuration
	webhookConfig := &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: configName,
		},
		Webhooks: []admissionv1.MutatingWebhook{
			{
				Name:                    webhookName + ".kecs.dev",
				AdmissionReviewVersions: admissionReviewVersions,
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{
						Name:      r.service,
						Namespace: r.namespace,
						Path:      ptr.To("/mutate/pods"),
						Port:      &r.port,
					},
					CABundle: caBundle,
				},
				Rules: []admissionv1.RuleWithOperations{
					{
						Operations: []admissionv1.OperationType{
							admissionv1.Create,
						},
						Rule: admissionv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
				FailurePolicy: &failurePolicy,
				SideEffects:   &sideEffects,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kecs.dev/managed": "true",
					},
				},
				// ObjectSelector removed - filtering is done within the webhook handler
				// to avoid blocking pods that don't have the label yet
			},
		},
	}

	// Check if webhook configuration already exists
	existing, err := r.clientset.AdmissionregistrationV1().
		MutatingWebhookConfigurations().
		Get(ctx, configName, metav1.GetOptions{})

	if err == nil && existing != nil {
		// Update existing configuration
		existing.Webhooks = webhookConfig.Webhooks
		_, err = r.clientset.AdmissionregistrationV1().
			MutatingWebhookConfigurations().
			Update(ctx, existing, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update webhook configuration: %w", err)
		}
		logging.Info("Updated webhook configuration", "name", configName)
	} else {
		// Create new configuration
		_, err = r.clientset.AdmissionregistrationV1().
			MutatingWebhookConfigurations().
			Create(ctx, webhookConfig, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create webhook configuration: %w", err)
		}
		logging.Info("Created webhook configuration", "name", configName)
	}

	return nil
}

// IsRegistered checks if the webhook configuration exists and is valid
func (r *WebhookRegistrar) IsRegistered(ctx context.Context) bool {
	configName := "kecs-webhook-config"

	config, err := r.clientset.AdmissionregistrationV1().
		MutatingWebhookConfigurations().
		Get(ctx, configName, metav1.GetOptions{})

	if err != nil || config == nil {
		return false
	}

	// Check if webhook has at least one webhook configured
	return len(config.Webhooks) > 0
}

// Unregister removes the webhook configuration
func (r *WebhookRegistrar) Unregister(ctx context.Context) error {
	configName := "kecs-webhook-config"

	err := r.clientset.AdmissionregistrationV1().
		MutatingWebhookConfigurations().
		Delete(ctx, configName, metav1.DeleteOptions{})

	if err != nil {
		logging.Warn("Failed to delete webhook configuration", "error", err)
		// Don't return error as it might already be deleted
	} else {
		logging.Info("Deleted webhook configuration", "name", configName)
	}

	return nil
}
