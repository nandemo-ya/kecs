package kubernetes

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// ManifestApplier handles applying Kubernetes manifests
type ManifestApplier struct {
	client          kubernetes.Interface
	extClient       apiextensionsclient.Interface
	dynamicClient   dynamic.Interface
	config          *rest.Config
}

// NewManifestApplier creates a new manifest applier
func NewManifestApplier(client kubernetes.Interface) *ManifestApplier {
	return &ManifestApplier{
		client: client,
	}
}

// NewManifestApplierWithConfig creates a new manifest applier with full client support
func NewManifestApplierWithConfig(config *rest.Config) (*ManifestApplier, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	extClient, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create apiextensions client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &ManifestApplier{
		client:        client,
		extClient:     extClient,
		dynamicClient: dynamicClient,
		config:        config,
	}, nil
}

// ApplyManifestsFromDirectory applies all YAML manifests from a directory
func (ma *ManifestApplier) ApplyManifestsFromDirectory(ctx context.Context, dir string) error {
	// Read kustomization.yaml if it exists
	kustomizationPath := filepath.Join(dir, "kustomization.yaml")
	if _, err := os.Stat(kustomizationPath); err == nil {
		return ma.applyKustomization(ctx, dir)
	}

	// Otherwise, apply all YAML files in the directory
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		return ma.ApplyManifestFile(ctx, path)
	})
}

// ApplyManifestFile applies a single manifest file
func (ma *ManifestApplier) ApplyManifestFile(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read manifest file %s: %w", path, err)
	}

	return ma.ApplyManifest(ctx, data)
}

// ApplyManifest applies raw manifest data
func (ma *ManifestApplier) ApplyManifest(ctx context.Context, data []byte) error {
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(data)), 4096)

	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode manifest: %w", err)
		}

		// Skip empty objects
		if len(obj.Object) == 0 {
			continue
		}

		if err := ma.applyObject(ctx, obj); err != nil {
			return fmt.Errorf("failed to apply %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}

		logging.Info("Applied object", "kind", obj.GetKind(), "name", obj.GetName(), "namespace", obj.GetNamespace())
	}

	return nil
}

// applyObject applies a single Kubernetes object
func (ma *ManifestApplier) applyObject(ctx context.Context, obj *unstructured.Unstructured) error {
	// Convert to typed object if possible for better handling
	typedObj, err := ma.convertToTypedObject(obj)
	if err != nil {
		// If conversion fails, apply as unstructured
		return ma.applyUnstructuredObject(ctx, obj)
	}

	// Apply based on type
	switch o := typedObj.(type) {
	case *corev1.Namespace:
		return ma.applyNamespace(ctx, o)
	case *corev1.ServiceAccount:
		return ma.applyServiceAccount(ctx, o)
	case *rbacv1.ClusterRole:
		return ma.applyClusterRole(ctx, o)
	case *rbacv1.ClusterRoleBinding:
		return ma.applyClusterRoleBinding(ctx, o)
	case *corev1.ConfigMap:
		return ma.applyConfigMap(ctx, o)
	case *corev1.Service:
		return ma.applyService(ctx, o)
	case *corev1.PersistentVolumeClaim:
		return ma.applyPVC(ctx, o)
	case *appsv1.Deployment:
		return ma.applyDeployment(ctx, o)
	case *apiextensionsv1.CustomResourceDefinition:
		return ma.applyCRD(ctx, o)
	default:
		// Fall back to unstructured
		return ma.applyUnstructuredObject(ctx, obj)
	}
}

// convertToTypedObject converts unstructured to typed object
func (ma *ManifestApplier) convertToTypedObject(obj *unstructured.Unstructured) (runtime.Object, error) {
	gvk := obj.GroupVersionKind()
	typedObj, err := scheme.Scheme.New(gvk)
	if err != nil {
		return nil, err
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, typedObj); err != nil {
		return nil, err
	}

	return typedObj, nil
}

// applyNamespace applies a namespace
func (ma *ManifestApplier) applyNamespace(ctx context.Context, ns *corev1.Namespace) error {
	existing, err := ma.client.CoreV1().Namespaces().Get(ctx, ns.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = ma.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update if exists
	existing.Labels = ns.Labels
	existing.Annotations = ns.Annotations
	_, err = ma.client.CoreV1().Namespaces().Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

// applyServiceAccount applies a service account
func (ma *ManifestApplier) applyServiceAccount(ctx context.Context, sa *corev1.ServiceAccount) error {
	existing, err := ma.client.CoreV1().ServiceAccounts(sa.Namespace).Get(ctx, sa.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = ma.client.CoreV1().ServiceAccounts(sa.Namespace).Create(ctx, sa, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update if exists
	existing.Labels = sa.Labels
	existing.Annotations = sa.Annotations
	_, err = ma.client.CoreV1().ServiceAccounts(sa.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

// applyClusterRole applies a cluster role
func (ma *ManifestApplier) applyClusterRole(ctx context.Context, cr *rbacv1.ClusterRole) error {
	existing, err := ma.client.RbacV1().ClusterRoles().Get(ctx, cr.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = ma.client.RbacV1().ClusterRoles().Create(ctx, cr, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update if exists
	existing.Labels = cr.Labels
	existing.Annotations = cr.Annotations
	existing.Rules = cr.Rules
	_, err = ma.client.RbacV1().ClusterRoles().Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

// applyClusterRoleBinding applies a cluster role binding
func (ma *ManifestApplier) applyClusterRoleBinding(ctx context.Context, crb *rbacv1.ClusterRoleBinding) error {
	existing, err := ma.client.RbacV1().ClusterRoleBindings().Get(ctx, crb.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = ma.client.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update if exists
	existing.Labels = crb.Labels
	existing.Annotations = crb.Annotations
	existing.RoleRef = crb.RoleRef
	existing.Subjects = crb.Subjects
	_, err = ma.client.RbacV1().ClusterRoleBindings().Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

// applyConfigMap applies a config map
func (ma *ManifestApplier) applyConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	existing, err := ma.client.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = ma.client.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, cm, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update if exists
	existing.Labels = cm.Labels
	existing.Annotations = cm.Annotations
	existing.Data = cm.Data
	existing.BinaryData = cm.BinaryData
	_, err = ma.client.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

// applyService applies a service
func (ma *ManifestApplier) applyService(ctx context.Context, svc *corev1.Service) error {
	existing, err := ma.client.CoreV1().Services(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = ma.client.CoreV1().Services(svc.Namespace).Create(ctx, svc, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update if exists (preserve ClusterIP and NodePorts)
	svc.Spec.ClusterIP = existing.Spec.ClusterIP
	svc.Spec.ClusterIPs = existing.Spec.ClusterIPs
	
	// Preserve NodePorts if not specified
	for i, port := range svc.Spec.Ports {
		for _, existingPort := range existing.Spec.Ports {
			if port.Name == existingPort.Name && port.NodePort == 0 {
				svc.Spec.Ports[i].NodePort = existingPort.NodePort
			}
		}
	}

	existing.Labels = svc.Labels
	existing.Annotations = svc.Annotations
	existing.Spec = svc.Spec
	_, err = ma.client.CoreV1().Services(svc.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

// applyPVC applies a persistent volume claim
func (ma *ManifestApplier) applyPVC(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error {
	_, err := ma.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(ctx, pvc.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = ma.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// PVCs are mostly immutable, so we don't update
	logging.Debug("PVC already exists, skipping update", "namespace", pvc.Namespace, "name", pvc.Name)
	return nil
}

// applyDeployment applies a deployment
func (ma *ManifestApplier) applyDeployment(ctx context.Context, deploy *appsv1.Deployment) error {
	existing, err := ma.client.AppsV1().Deployments(deploy.Namespace).Get(ctx, deploy.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = ma.client.AppsV1().Deployments(deploy.Namespace).Create(ctx, deploy, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update if exists
	existing.Labels = deploy.Labels
	existing.Annotations = deploy.Annotations
	existing.Spec = deploy.Spec
	_, err = ma.client.AppsV1().Deployments(deploy.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

// applyCRD applies a CustomResourceDefinition
func (ma *ManifestApplier) applyCRD(ctx context.Context, crd *apiextensionsv1.CustomResourceDefinition) error {
	if ma.extClient == nil {
		return fmt.Errorf("apiextensions client not available")
	}

	existing, err := ma.extClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crd.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = ma.extClient.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crd, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update if exists
	existing.Labels = crd.Labels
	existing.Annotations = crd.Annotations
	existing.Spec = crd.Spec
	_, err = ma.extClient.ApiextensionsV1().CustomResourceDefinitions().Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

// applyUnstructuredObject applies an unstructured object using dynamic client
func (ma *ManifestApplier) applyUnstructuredObject(ctx context.Context, obj *unstructured.Unstructured) error {
	// If dynamic client is not available, skip
	if ma.dynamicClient == nil {
		logging.Debug("Skipping unstructured object (dynamic client not available)", "kind", obj.GetKind(), "name", obj.GetName())
		return nil
	}

	// Get the resource schema
	gvk := obj.GroupVersionKind()
	gvr, err := ma.getGVRFromGVK(gvk)
	if err != nil {
		logging.Warn("Failed to get GVR for object", "kind", obj.GetKind(), "name", obj.GetName(), "error", err)
		return nil
	}

	// Get the dynamic client for this resource
	var client dynamic.ResourceInterface
	if obj.GetNamespace() != "" {
		client = ma.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
	} else {
		client = ma.dynamicClient.Resource(gvr)
	}

	// Try to get existing object
	existing, err := client.Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new object
			_, err = client.Create(ctx, obj, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update existing object (preserve resourceVersion)
	obj.SetResourceVersion(existing.GetResourceVersion())
	_, err = client.Update(ctx, obj, metav1.UpdateOptions{})
	return err
}

// applyKustomization applies manifests using kustomization.yaml
func (ma *ManifestApplier) applyKustomization(ctx context.Context, dir string) error {
	// Read kustomization.yaml
	kustomizationPath := filepath.Join(dir, "kustomization.yaml")
	data, err := os.ReadFile(kustomizationPath)
	if err != nil {
		return fmt.Errorf("failed to read kustomization.yaml: %w", err)
	}

	// Parse kustomization
	var kustomization struct {
		Resources []string `yaml:"resources"`
		Namespace string   `yaml:"namespace"`
	}
	if err := yaml.Unmarshal(data, &kustomization); err != nil {
		return fmt.Errorf("failed to parse kustomization.yaml: %w", err)
	}

	// Apply each resource
	for _, resource := range kustomization.Resources {
		resourcePath := filepath.Join(dir, resource)
		if err := ma.ApplyManifestFile(ctx, resourcePath); err != nil {
			return fmt.Errorf("failed to apply resource %s: %w", resource, err)
		}
	}

	return nil
}

// getGVRFromGVK converts GroupVersionKind to GroupVersionResource
func (ma *ManifestApplier) getGVRFromGVK(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	// For CRDs, we need to discover the resource name
	// This is a simplified version - in production, you'd use discovery client
	
	// Handle known Traefik CRDs
	if gvk.Group == "traefik.io" {
		var resource string
		switch gvk.Kind {
		case "IngressRoute":
			resource = "ingressroutes"
		case "Middleware":
			resource = "middlewares"
		case "IngressRouteTCP":
			resource = "ingressroutetcps"
		case "IngressRouteUDP":
			resource = "ingressrouteudps"
		case "TLSOption":
			resource = "tlsoptions"
		case "TLSStore":
			resource = "tlsstores"
		case "TraefikService":
			resource = "traefikservices"
		case "ServersTransport":
			resource = "serverstransports"
		case "ServersTransportTCP":
			resource = "serverstransporttcps"
		case "MiddlewareTCP":
			resource = "middlewaretcps"
		default:
			// Try to pluralize the kind
			resource = strings.ToLower(gvk.Kind) + "s"
		}
		
		return schema.GroupVersionResource{
			Group:    gvk.Group,
			Version:  gvk.Version,
			Resource: resource,
		}, nil
	}
	
	// For other resources, try simple pluralization
	// In production, use discovery client for accurate resource names
	resource := strings.ToLower(gvk.Kind) + "s"
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}, nil
}