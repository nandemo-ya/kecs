package kubernetes

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Client wraps the Kubernetes clientset for KECS operations
type Client struct {
	Clientset kubernetes.Interface
	Config    *rest.Config
}

// NewClient creates a new Kubernetes client from the given config
func NewClient(config *rest.Config) (*Client, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		Clientset: clientset,
		Config:    config,
	}, nil
}
