package main

import (
	"os"
	
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/cmd"
)

func init() {
	// Set environment variable to suppress k3d logs
	// This will be checked by our kubernetes package
	os.Setenv("K3D_LOG_LEVEL", "panic")
}

func main() {
	cmd.Execute()
}
