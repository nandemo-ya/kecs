package utils

// Global manager instance for native mode
var globalNativeManager *NativeKECSManager

// StartKECSForTest starts a KECS instance using native Docker host mode
func StartKECSForTest(t TestingT, testName string) KECSContainerInterface {
	// Always use native Docker host mode
	if globalNativeManager == nil {
		globalNativeManager = NewNativeKECSManager()
		// Register cleanup on first use
		t.Cleanup(func() {
			if err := globalNativeManager.StopAll(); err != nil {
				t.Logf("Warning: failed to stop all instances: %v", err)
			}
			// Also cleanup any orphaned resources
			if err := CleanupOrphanedResources(); err != nil {
				t.Logf("Warning: failed to cleanup orphaned resources: %v", err)
			}
		})
	}
	
	instance, err := globalNativeManager.StartKECS(testName)
	if err != nil {
		t.Fatalf("Failed to start KECS: %v", err)
	}
	
	// Return adapter that implements KECSContainerInterface
	return NewNativeKECSAdapter(instance, globalNativeManager)
}

// KECSContainerInterface defines the common interface for KECS test containers
type KECSContainerInterface interface {
	Endpoint() string
	AdminEndpoint() string
	GetLogs() (string, error)
	Cleanup() error
	APIEndpoint() string
	Stop() error
	RunCommand(command ...string) (string, error)
	ExecuteCommand(args ...string) (string, error)
}

// StartKECSWithOptions starts a KECS instance with custom options
type KECSStartOptions struct {
	TestName          string
	EnableLocalStack  bool
	EnableTraefik     bool
	DataDir           string // Optional: specify custom data directory
	AdditionalEnv     map[string]string
}

// StartKECSWithOptionsForTest starts KECS with custom options
func StartKECSWithOptionsForTest(t TestingT, opts KECSStartOptions) KECSContainerInterface {
	if globalNativeManager == nil {
		globalNativeManager = NewNativeKECSManager()
		t.Cleanup(func() {
			if err := globalNativeManager.StopAll(); err != nil {
				t.Logf("Warning: failed to stop all instances: %v", err)
			}
			// Also cleanup any orphaned resources
			if err := CleanupOrphanedResources(); err != nil {
				t.Logf("Warning: failed to cleanup orphaned resources: %v", err)
			}
		})
	}
	
	// For now, use standard StartKECS
	// TODO: Implement options support in native manager
	instance, err := globalNativeManager.StartKECS(opts.TestName)
	if err != nil {
		t.Fatalf("Failed to start KECS: %v", err)
	}
	
	return NewNativeKECSAdapter(instance, globalNativeManager)
}

// CleanupTestResources cleans up any orphaned test resources
func CleanupTestResources() error {
	return CleanupOrphanedResources()
}

// GetGlobalNativeManager returns the global native manager instance (for advanced use)
func GetGlobalNativeManager() *NativeKECSManager {
	if globalNativeManager == nil {
		globalNativeManager = NewNativeKECSManager()
	}
	return globalNativeManager
}

