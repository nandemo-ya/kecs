package utils

// ECSClient wraps CurlClient to provide backward compatibility
type ECSClient struct {
	*CurlClient
}

// NewECSClient creates a new ECS client (backward compatibility)
func NewECSClient(endpoint string) *ECSClient {
	return &ECSClient{
		CurlClient: NewCurlClient(endpoint),
	}
}