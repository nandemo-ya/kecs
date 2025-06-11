package utils

// NewECSClient creates a new ECS client with optional mode
// Default is curl mode for backward compatibility
func NewECSClient(endpoint string, mode ...ClientMode) ECSClientInterface {
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

// ECSClient is a type alias for backward compatibility
// Deprecated: Use NewECSClient with ClientMode instead
type ECSClient = CurlClient