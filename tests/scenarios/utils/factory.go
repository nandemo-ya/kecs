package utils

// NewECSClientInterface creates a new ECS client interface with optional mode
// Use this for new tests that support multiple client modes
func NewECSClientInterface(endpoint string, mode ...ClientMode) ECSClientInterface {
	if len(mode) > 0 {
		switch mode[0] {
		case AWSCLIMode:
			return NewAWSCLIClient(endpoint)
		case CurlMode:
			return NewCurlClient(endpoint)
		}
	}
	// Default to curl mode for backward compatibility
	return NewCurlClient(endpoint)
}

