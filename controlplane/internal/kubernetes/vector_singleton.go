package kubernetes

import (
	"context"
	"sync"

	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

var (
	vectorOnce sync.Once
	vectorErr  error
)

// DeployVectorOnce ensures Vector is deployed only once per KECS instance
// This uses a singleton pattern to prevent multiple deployments
func DeployVectorOnce(ctx context.Context, clientset kubernetes.Interface, localstackEndpoint string, region string) error {
	vectorOnce.Do(func() {
		logging.Info("Deploying Vector DaemonSet (singleton)")
		vectorErr = EnsureVectorDaemonSet(ctx, clientset, localstackEndpoint, region)
		if vectorErr != nil {
			logging.Error("Failed to deploy Vector DaemonSet", "error", vectorErr)
		}
	})
	return vectorErr
}

// ResetVectorSingleton resets the singleton state (mainly for testing)
func ResetVectorSingleton() {
	vectorOnce = sync.Once{}
	vectorErr = nil
}
