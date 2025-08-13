package kubernetes

import (
	"io"

	"github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/sirupsen/logrus"
)

func init() {
	// Always suppress k3d logs for cleaner output
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

	// Note: klog suppression is no longer needed as we don't use klog anymore
}
