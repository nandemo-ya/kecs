package kubernetes

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes/resources"
)

// ResourceDeployer handles deployment of Kubernetes resources
type ResourceDeployer struct {
	client        kubernetes.Interface
	extClient     apiextensionsclient.Interface
	config        *rest.Config
}

// NewResourceDeployer creates a new resource deployer
func NewResourceDeployer(client kubernetes.Interface) *ResourceDeployer {
	return &ResourceDeployer{
		client: client,
	}
}

// NewResourceDeployerWithConfig creates a new resource deployer with full config
func NewResourceDeployerWithConfig(client kubernetes.Interface, config *rest.Config) (*ResourceDeployer, error) {
	extClient, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create apiextensions client: %w", err)
	}
	
	return &ResourceDeployer{
		client:    client,
		extClient: extClient,
		config:    config,
	}, nil
}

// DeployControlPlane deploys all control plane resources
func (d *ResourceDeployer) DeployControlPlane(ctx context.Context, config *resources.ControlPlaneConfig) error {
	klog.Info("Deploying KECS control plane resources")
	
	// Create resources
	res := resources.CreateControlPlaneResources(config)
	
	// Deploy namespace first
	if err := d.applyNamespace(ctx, res.Namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}
	
	// Deploy RBAC resources
	if err := d.applyServiceAccount(ctx, res.ServiceAccount); err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}
	
	if err := d.applyClusterRole(ctx, res.ClusterRole); err != nil {
		return fmt.Errorf("failed to create cluster role: %w", err)
	}
	
	if err := d.applyClusterRoleBinding(ctx, res.ClusterRoleBinding); err != nil {
		return fmt.Errorf("failed to create cluster role binding: %w", err)
	}
	
	// Deploy ConfigMap
	if err := d.applyConfigMap(ctx, res.ConfigMap); err != nil {
		return fmt.Errorf("failed to create config map: %w", err)
	}
	
	// Deploy PVC
	if err := d.applyPVC(ctx, res.PVC); err != nil {
		return fmt.Errorf("failed to create PVC: %w", err)
	}
	
	// Deploy Services
	for _, svc := range res.Services {
		if err := d.applyService(ctx, svc); err != nil {
			return fmt.Errorf("failed to create service %s: %w", svc.Name, err)
		}
	}
	
	// Deploy Deployment
	if err := d.applyDeployment(ctx, res.Deployment); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}
	
	klog.Info("Control plane resources deployed successfully")
	return nil
}

// DeployTraefik deploys all Traefik resources
func (d *ResourceDeployer) DeployTraefik(ctx context.Context, config *resources.TraefikConfig) error {
	klog.Info("Deploying Traefik gateway resources")
	
	// Deploy Traefik CRDs first if we have extClient
	if d.extClient != nil {
		if err := d.deployTraefikCRDs(ctx); err != nil {
			return fmt.Errorf("failed to deploy Traefik CRDs: %w", err)
		}
	}
	
	// Create resources
	res := resources.CreateTraefikResources(config)
	
	// Deploy RBAC resources
	if err := d.applyServiceAccount(ctx, res.ServiceAccount); err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}
	
	if err := d.applyClusterRole(ctx, res.ClusterRole); err != nil {
		return fmt.Errorf("failed to create cluster role: %w", err)
	}
	
	if err := d.applyClusterRoleBinding(ctx, res.ClusterRoleBinding); err != nil {
		return fmt.Errorf("failed to create cluster role binding: %w", err)
	}
	
	// Deploy ConfigMaps
	if err := d.applyConfigMap(ctx, res.ConfigMap); err != nil {
		return fmt.Errorf("failed to create config map: %w", err)
	}
	
	if err := d.applyConfigMap(ctx, res.DynamicConfigMap); err != nil {
		return fmt.Errorf("failed to create dynamic config map: %w", err)
	}
	
	// Deploy Services
	for _, svc := range res.Services {
		if err := d.applyService(ctx, svc); err != nil {
			return fmt.Errorf("failed to create service %s: %w", svc.Name, err)
		}
	}
	
	// Deploy Deployment
	if err := d.applyDeployment(ctx, res.Deployment); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}
	
	// Deploy IngressRoutes if we have the necessary clients
	if d.config != nil {
		if err := d.deployTraefikRoutes(ctx); err != nil {
			return fmt.Errorf("failed to deploy Traefik routes: %w", err)
		}
	}
	
	klog.Info("Traefik resources deployed successfully")
	return nil
}

// WaitForDeploymentReady waits for a deployment to become ready
func (d *ResourceDeployer) WaitForDeploymentReady(ctx context.Context, namespace, name string, timeout time.Duration) error {
	klog.Infof("Waiting for deployment %s/%s to be ready", namespace, name)
	
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		deployment, err := d.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		
		// Check if deployment is ready
		if deployment.Status.ReadyReplicas > 0 && deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
			return true, nil
		}
		
		return false, nil
	})
}

// applyNamespace creates or updates a namespace
func (d *ResourceDeployer) applyNamespace(ctx context.Context, ns *corev1.Namespace) error {
	existing, err := d.client.CoreV1().Namespaces().Get(ctx, ns.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = d.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			klog.Infof("Created namespace %s", ns.Name)
			return nil
		}
		return err
	}
	
	// Update labels if needed
	existing.Labels = ns.Labels
	_, err = d.client.CoreV1().Namespaces().Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	klog.Infof("Updated namespace %s", ns.Name)
	return nil
}

// applyServiceAccount creates or updates a service account
func (d *ResourceDeployer) applyServiceAccount(ctx context.Context, sa *corev1.ServiceAccount) error {
	existing, err := d.client.CoreV1().ServiceAccounts(sa.Namespace).Get(ctx, sa.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = d.client.CoreV1().ServiceAccounts(sa.Namespace).Create(ctx, sa, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			klog.Infof("Created service account %s/%s", sa.Namespace, sa.Name)
			return nil
		}
		return err
	}
	
	// Update if exists
	existing.Labels = sa.Labels
	_, err = d.client.CoreV1().ServiceAccounts(sa.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	klog.Infof("Updated service account %s/%s", sa.Namespace, sa.Name)
	return nil
}

// applyClusterRole creates or updates a cluster role
func (d *ResourceDeployer) applyClusterRole(ctx context.Context, cr *rbacv1.ClusterRole) error {
	existing, err := d.client.RbacV1().ClusterRoles().Get(ctx, cr.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = d.client.RbacV1().ClusterRoles().Create(ctx, cr, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			klog.Infof("Created cluster role %s", cr.Name)
			return nil
		}
		return err
	}
	
	// Update if exists
	existing.Labels = cr.Labels
	existing.Rules = cr.Rules
	_, err = d.client.RbacV1().ClusterRoles().Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	klog.Infof("Updated cluster role %s", cr.Name)
	return nil
}

// applyClusterRoleBinding creates or updates a cluster role binding
func (d *ResourceDeployer) applyClusterRoleBinding(ctx context.Context, crb *rbacv1.ClusterRoleBinding) error {
	existing, err := d.client.RbacV1().ClusterRoleBindings().Get(ctx, crb.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = d.client.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			klog.Infof("Created cluster role binding %s", crb.Name)
			return nil
		}
		return err
	}
	
	// Update if exists
	existing.Labels = crb.Labels
	existing.RoleRef = crb.RoleRef
	existing.Subjects = crb.Subjects
	_, err = d.client.RbacV1().ClusterRoleBindings().Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	klog.Infof("Updated cluster role binding %s", crb.Name)
	return nil
}

// applyConfigMap creates or updates a config map
func (d *ResourceDeployer) applyConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	existing, err := d.client.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = d.client.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, cm, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			klog.Infof("Created config map %s/%s", cm.Namespace, cm.Name)
			return nil
		}
		return err
	}
	
	// Update if exists
	existing.Labels = cm.Labels
	existing.Data = cm.Data
	_, err = d.client.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	klog.Infof("Updated config map %s/%s", cm.Namespace, cm.Name)
	return nil
}

// applyService creates or updates a service
func (d *ResourceDeployer) applyService(ctx context.Context, svc *corev1.Service) error {
	existing, err := d.client.CoreV1().Services(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = d.client.CoreV1().Services(svc.Namespace).Create(ctx, svc, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			klog.Infof("Created service %s/%s", svc.Namespace, svc.Name)
			return nil
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
	existing.Spec = svc.Spec
	_, err = d.client.CoreV1().Services(svc.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	klog.Infof("Updated service %s/%s", svc.Namespace, svc.Name)
	return nil
}

// applyPVC creates a PVC (only if it doesn't exist)
func (d *ResourceDeployer) applyPVC(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error {
	_, err := d.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(ctx, pvc.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = d.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			klog.Infof("Created PVC %s/%s", pvc.Namespace, pvc.Name)
			return nil
		}
		return err
	}
	
	// PVCs are mostly immutable, so we don't update
	klog.V(2).Infof("PVC %s/%s already exists, skipping update", pvc.Namespace, pvc.Name)
	return nil
}

// applyDeployment creates or updates a deployment
func (d *ResourceDeployer) applyDeployment(ctx context.Context, deploy *appsv1.Deployment) error {
	existing, err := d.client.AppsV1().Deployments(deploy.Namespace).Get(ctx, deploy.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = d.client.AppsV1().Deployments(deploy.Namespace).Create(ctx, deploy, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			klog.Infof("Created deployment %s/%s", deploy.Namespace, deploy.Name)
			return nil
		}
		return err
	}
	
	// Update if exists
	existing.Labels = deploy.Labels
	existing.Spec = deploy.Spec
	_, err = d.client.AppsV1().Deployments(deploy.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	klog.Infof("Updated deployment %s/%s", deploy.Namespace, deploy.Name)
	return nil
}

// deployTraefikCRDs deploys Traefik CRDs
func (d *ResourceDeployer) deployTraefikCRDs(ctx context.Context) error {
	klog.Info("Deploying Traefik CRDs")
	
	// Create IngressRoute CRD
	ingressRouteCRD := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ingressroutes.traefik.io",
			Labels: map[string]string{
				resources.LabelManagedBy: "true",
				resources.LabelComponent: "gateway",
			},
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "traefik.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:     "IngressRoute",
				Plural:   "ingressroutes",
				Singular: "ingressroute",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"spec": {
									Type: "object",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"entryPoints": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "string",
												},
											},
										},
										"routes": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "object",
													Properties: map[string]apiextensionsv1.JSONSchemaProps{
														"match": {
															Type: "string",
														},
														"kind": {
															Type: "string",
														},
														"priority": {
															Type: "integer",
														},
														"services": {
															Type: "array",
															Items: &apiextensionsv1.JSONSchemaPropsOrArray{
																Schema: &apiextensionsv1.JSONSchemaProps{
																	Type: "object",
																	Properties: map[string]apiextensionsv1.JSONSchemaProps{
																		"name": {
																			Type: "string",
																		},
																		"port": {
																			XIntOrString: true,
																		},
																		"namespace": {
																			Type: "string",
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	if err := d.applyCRD(ctx, ingressRouteCRD); err != nil {
		return fmt.Errorf("failed to create IngressRoute CRD: %w", err)
	}
	
	klog.Info("Traefik CRDs deployed successfully")
	return nil
}

// applyCRD creates or updates a CRD
func (d *ResourceDeployer) applyCRD(ctx context.Context, crd *apiextensionsv1.CustomResourceDefinition) error {
	existing, err := d.extClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crd.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = d.extClient.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crd, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			klog.Infof("Created CRD %s", crd.Name)
			return nil
		}
		return err
	}
	
	// Update if exists
	existing.Labels = crd.Labels
	existing.Spec = crd.Spec
	_, err = d.extClient.ApiextensionsV1().CustomResourceDefinitions().Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	klog.Infof("Updated CRD %s", crd.Name)
	return nil
}

// deployTraefikRoutes deploys Traefik IngressRoute resources
func (d *ResourceDeployer) deployTraefikRoutes(ctx context.Context) error {
	klog.Info("Deploying Traefik routes")
	
	// For now, we'll skip the actual IngressRoute deployment as it requires dynamic client
	// This can be implemented later with unstructured resources
	klog.Info("Traefik routes deployment skipped (requires dynamic client)")
	
	return nil
}