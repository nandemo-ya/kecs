package kubernetes

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

//go:embed manifests/traefik.yaml
var traefikManifest string

// TraefikManager manages Traefik reverse proxy deployment
type TraefikManager struct {
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
	restConfig    *rest.Config
}

// NewTraefikManager creates a new Traefik manager
func NewTraefikManager(kubeClient kubernetes.Interface, restConfig *rest.Config) (*TraefikManager, error) {
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &TraefikManager{
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
		restConfig:    restConfig,
	}, nil
}

// Deploy deploys Traefik to the cluster
func (tm *TraefikManager) Deploy(ctx context.Context) error {
	klog.Info("Deploying Traefik reverse proxy...")

	// First pass: collect resources by type
	var ingressRoutes []*unstructured.Unstructured
	var otherResources []*unstructured.Unstructured

	klog.Info("Parsing Traefik manifests...")
	// Parse the manifest into individual resources
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(traefikManifest), 4096)
	
	for {
		var rawObj unstructured.Unstructured
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode manifest: %w", err)
		}

		// Skip empty objects
		if len(rawObj.Object) == 0 {
			continue
		}

		// Separate IngressRoute resources to apply after Traefik is ready
		if rawObj.GetKind() == "IngressRoute" {
			ingressRoutes = append(ingressRoutes, rawObj.DeepCopy())
		} else {
			otherResources = append(otherResources, rawObj.DeepCopy())
		}
	}

	// Apply non-IngressRoute resources first
	klog.Infof("Applying %d Traefik resources...", len(otherResources))
	for i, obj := range otherResources {
		klog.V(2).Infof("Applying resource %d/%d: %s %s", i+1, len(otherResources), obj.GetKind(), obj.GetName())
		if err := tm.applyResource(ctx, obj); err != nil {
			// Log error but continue with other resources
			klog.Warningf("Failed to apply resource %s/%s: %v", 
				obj.GetKind(), obj.GetName(), err)
		}
	}

	// Wait for Traefik to be ready
	klog.Info("Waiting for Traefik deployment to be ready...")
	if err := tm.WaitForReady(ctx, 2*time.Minute); err != nil {
		return err
	}

	// Install Traefik CRDs if needed
	if err := tm.ensureTraefikCRDs(ctx); err != nil {
		klog.Warningf("Failed to ensure Traefik CRDs: %v", err)
	}

	// Now apply IngressRoute resources
	if len(ingressRoutes) > 0 {
		klog.Infof("Applying %d IngressRoute resources...", len(ingressRoutes))
		for i, obj := range ingressRoutes {
			klog.V(2).Infof("Applying IngressRoute %d/%d: %s", i+1, len(ingressRoutes), obj.GetName())
			if err := tm.applyResource(ctx, obj); err != nil {
				klog.Warningf("Failed to apply IngressRoute %s: %v", obj.GetName(), err)
			}
		}
	}

	klog.Info("Traefik deployment completed successfully")
	return nil
}

// applyResource applies a single resource to the cluster
func (tm *TraefikManager) applyResource(ctx context.Context, obj *unstructured.Unstructured) error {
	gvk := obj.GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: tm.pluralize(gvk.Kind),
	}

	namespace := obj.GetNamespace()
	var resourceClient dynamic.ResourceInterface
	if namespace != "" {
		resourceClient = tm.dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resourceClient = tm.dynamicClient.Resource(gvr)
	}

	// Try to create the resource
	_, err := resourceClient.Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Resource already exists, try to update it
			_, err = resourceClient.Update(ctx, obj, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update resource: %w", err)
			}
			klog.V(2).Infof("Updated %s %s/%s", gvk.Kind, namespace, obj.GetName())
		} else {
			return fmt.Errorf("failed to create resource: %w", err)
		}
	} else {
		klog.V(2).Infof("Created %s %s/%s", gvk.Kind, namespace, obj.GetName())
	}

	return nil
}

// WaitForReady waits for Traefik to be ready
func (tm *TraefikManager) WaitForReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for Traefik to be ready")
			}

			// Check if Traefik deployment is ready
			deployment, err := tm.kubeClient.AppsV1().Deployments("kecs-system").
				Get(ctx, "traefik", metav1.GetOptions{})
			if err != nil {
				klog.V(4).Infof("Failed to get Traefik deployment: %v", err)
				continue
			}

			if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
				klog.Info("Traefik deployment is ready")
				return nil
			}

			klog.Infof("Waiting for Traefik deployment: %d/%d replicas ready",
				deployment.Status.ReadyReplicas, *deployment.Spec.Replicas)
		}
	}
}

// GetProxyEndpoint returns the endpoint for accessing services through Traefik
func (tm *TraefikManager) GetProxyEndpoint() string {
	// This will be set up with port forwarding or NodePort
	// For now, return the expected endpoint
	return "http://localhost:8090"
}

// ensureTraefikCRDs waits for Traefik CRDs to be available
func (tm *TraefikManager) ensureTraefikCRDs(ctx context.Context) error {
	klog.Info("Waiting for Traefik CRDs to be available...")
	
	// Wait up to 30 seconds for CRDs to be available
	deadline := time.Now().Add(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for Traefik CRDs")
			}

			// Check if IngressRoute CRD exists
			gvr := schema.GroupVersionResource{
				Group:    "traefik.io",
				Version:  "v1alpha1",
				Resource: "ingressroutes",
			}

			// Try to list IngressRoutes to check if CRD is available
			_, err := tm.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{Limit: 1})
			if err == nil {
				klog.Info("Traefik CRDs are available")
				return nil
			}

			if !errors.IsNotFound(err) {
				klog.V(4).Infof("Waiting for Traefik CRDs: %v", err)
			}
		}
	}
}

// pluralize converts a Kind to its plural form for API resources
func (tm *TraefikManager) pluralize(kind string) string {
	// Simple pluralization rules
	switch strings.ToLower(kind) {
	case "ingress":
		return "ingresses"
	case "ingressroute":
		return "ingressroutes"
	case "service":
		return "services"
	case "deployment":
		return "deployments"
	case "configmap":
		return "configmaps"
	case "namespace":
		return "namespaces"
	case "serviceaccount":
		return "serviceaccounts"
	case "clusterrole":
		return "clusterroles"
	case "clusterrolebinding":
		return "clusterrolebindings"
	default:
		// Default: lowercase and add 's'
		return strings.ToLower(kind) + "s"
	}
}