package kubernetes

import (
	"io"
	"os"
	
	"github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/sirupsen/logrus"
	klog "k8s.io/klog/v2"
)

func init() {
	// Check if we should suppress k3d logs
	if os.Getenv("K3D_LOG_LEVEL") == "panic" || os.Getenv("LOGRUS_LEVEL") == "panic" {
		// Suppress k3d's logger
		if logger.Logger != nil {
			logger.Logger.SetLevel(logrus.PanicLevel)
			logger.Logger.SetOutput(io.Discard)
		}
		
		// Suppress logrus completely
		logrus.SetLevel(logrus.PanicLevel)
		logrus.SetOutput(io.Discard)
		logrus.StandardLogger().SetLevel(logrus.PanicLevel)
		logrus.StandardLogger().SetOutput(io.Discard)
		
		// Disable all hooks
		logrus.StandardLogger().Hooks = make(logrus.LevelHooks)
		
		// Also suppress klog which k3d might use
		klog.SetOutput(io.Discard)
	}
}