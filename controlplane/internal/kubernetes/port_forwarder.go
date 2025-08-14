package kubernetes

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// PortForwarder manages port forwarding to Kubernetes services
type PortForwarder struct {
	kubeClient kubernetes.Interface
	restConfig *rest.Config
	namespace  string
	service    string
	localPort  int
	remotePort int
	stopCh     chan struct{}
	readyCh    chan struct{}
	errorCh    chan error
	mu         sync.Mutex
	forwarder  *portforward.PortForwarder
}

// NewPortForwarder creates a new port forwarder
func NewPortForwarder(kubeClient kubernetes.Interface, restConfig *rest.Config, namespace, service string, localPort, remotePort int) *PortForwarder {
	return &PortForwarder{
		kubeClient: kubeClient,
		restConfig: restConfig,
		namespace:  namespace,
		service:    service,
		localPort:  localPort,
		remotePort: remotePort,
		stopCh:     make(chan struct{}),
		readyCh:    make(chan struct{}),
		errorCh:    make(chan error, 1),
	}
}

// Start begins port forwarding
func (pf *PortForwarder) Start(ctx context.Context) error {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	// Get service
	svc, err := pf.kubeClient.CoreV1().Services(pf.namespace).Get(ctx, pf.service, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	// Get pods for the service
	labelSelector := ""
	for k, v := range svc.Spec.Selector {
		if labelSelector != "" {
			labelSelector += ","
		}
		labelSelector += fmt.Sprintf("%s=%s", k, v)
	}

	pods, err := pf.kubeClient.CoreV1().Pods(pf.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found for service %s/%s", pf.namespace, pf.service)
	}

	// Use the first pod
	podName := pods.Items[0].Name

	// Create the URL for the pod
	req := pf.kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(pf.namespace).
		Name(podName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(pf.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	// Prepare ports
	ports := []string{fmt.Sprintf("%d:%d", pf.localPort, pf.remotePort)}

	// Create port forwarder
	pf.forwarder, err = portforward.New(dialer, ports, pf.stopCh, pf.readyCh, nil, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create port forwarder: %w", err)
	}

	// Start forwarding in background
	go func() {
		if err := pf.forwarder.ForwardPorts(); err != nil {
			select {
			case pf.errorCh <- err:
			default:
			}
		}
	}()

	// Wait for ready or error
	select {
	case <-pf.readyCh:
		logging.Info("Port forwarding established",
			"localPort", pf.localPort, "namespace", pf.namespace, "service", pf.service, "remotePort", pf.remotePort)
		return nil
	case err := <-pf.errorCh:
		return fmt.Errorf("port forwarding failed: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop terminates port forwarding
func (pf *PortForwarder) Stop() {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	close(pf.stopCh)
	if pf.forwarder != nil {
		pf.forwarder.Close()
	}
}

// GetLocalEndpoint returns the local endpoint for accessing the forwarded service
func (pf *PortForwarder) GetLocalEndpoint() string {
	return fmt.Sprintf("localhost:%d", pf.localPort)
}

// IsPortAvailable checks if a port is available
func IsPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
