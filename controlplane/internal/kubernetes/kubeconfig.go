package kubernetes

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubeConfig returns the Kubernetes client configuration
// It tries to use the default kubeconfig paths and settings
func GetKubeConfig() (*rest.Config, error) {
	// Try to use the default kubeconfig loading rules
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	// Adjust config for better performance
	config.QPS = 100
	config.Burst = 200

	return config, nil
}

// GetInClusterClient returns a Kubernetes clientset using in-cluster configuration
// This should be used when the control plane is running inside a Kubernetes pod
func GetInClusterClient() (*kubernetes.Clientset, error) {
	// Get in-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Adjust config for better performance
	config.QPS = 100
	config.Burst = 200

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
