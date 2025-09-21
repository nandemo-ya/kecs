package api

import (
	"fmt"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
)

// toECSError converts internal errors to appropriate ECS API errors
func toECSError(err error, operation string) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Handle DuckDB constraint violations
	if strings.Contains(errStr, "Constraint Error: Duplicate key") || strings.Contains(errStr, "violates unique constraint") {
		// Extract the resource name from the error message if possible
		resourceName := extractResourceName(errStr)
		message := fmt.Sprintf("The service %s already exists", resourceName)
		if resourceName == "" {
			message = "A service with the same name already exists in the specified cluster"
		}

		return &generated.InvalidParameterException{
			Message: ptr.String(message),
		}
	}

	// Handle resource not found errors
	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "does not exist") {
		return &generated.ResourceNotFoundException{
			Message: ptr.String(err.Error()),
		}
	}

	// Handle invalid request errors
	if strings.Contains(errStr, "invalid") || strings.Contains(errStr, "required") {
		return &generated.InvalidParameterException{
			Message: ptr.String(err.Error()),
		}
	}

	// Default to generic server error
	return &generated.ServerException{
		Message: ptr.String(fmt.Sprintf("An internal error occurred while processing the %s operation", operation)),
	}
}

// extractResourceName attempts to extract the resource name from error messages
func extractResourceName(errStr string) string {
	// Try to extract from ARN
	if idx := strings.Index(errStr, "arn:aws:ecs:"); idx != -1 {
		arn := errStr[idx:]
		// Find the end of the ARN (usually ends with a quote or space)
		endIdx := strings.IndexAny(arn, "\" \n")
		if endIdx != -1 {
			arn = arn[:endIdx]
		}
		// Extract service name from ARN
		// Format: arn:aws:ecs:region:account:service/cluster-name/service-name
		parts := strings.Split(arn, "/")
		if len(parts) >= 3 {
			return parts[len(parts)-1]
		}
	}

	// Try to extract from "service: serviceName" pattern
	if idx := strings.Index(errStr, "service: "); idx != -1 {
		name := errStr[idx+9:]
		if endIdx := strings.IndexAny(name, " ,\n"); endIdx != -1 {
			return name[:endIdx]
		}
	}

	return ""
}
