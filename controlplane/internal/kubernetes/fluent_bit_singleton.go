package kubernetes

import (
	"context"
	"sync"

	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

var (
	fluentBitOnce sync.Once
	fluentBitErr  error
)

// EnsureFluentBitDaemonSet ensures FluentBit DaemonSet is deployed in kecs-system namespace only once
func EnsureFluentBitDaemonSet(ctx context.Context, clientset *kubernetes.Clientset, localstackEndpoint, region string) error {
	fluentBitOnce.Do(func() {
		logging.Info("Ensuring FluentBit DaemonSet in kecs-system namespace")
		
		// Create FluentBit manager
		manager := NewFluentBitManager(clientset, localstackEndpoint, region)
		
		// Deploy FluentBit DaemonSet to kecs-system namespace
		fluentBitErr = manager.DeployFluentBitDaemonSet(ctx, "kecs-system")
		if fluentBitErr != nil {
			logging.Error("Failed to deploy FluentBit DaemonSet", "error", fluentBitErr)
		} else {
			logging.Info("Successfully deployed FluentBit DaemonSet to kecs-system")
		}
	})
	
	return fluentBitErr
}