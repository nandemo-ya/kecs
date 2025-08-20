package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// HandleTaskSetRequest handles TaskSet-related API requests
func (api *DefaultECSAPI) HandleTaskSetRequest(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logging.Error("Failed to read request body", "error", err)
		writeErrorResponse(w, http.StatusBadRequest, "InvalidParameterException", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	ctx := context.Background()

	switch target {
	case "AmazonEC2ContainerServiceV20141113.CreateTaskSet":
		var req generated.CreateTaskSetRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logging.Error("Failed to unmarshal CreateTaskSet request", "error", err)
			writeErrorResponse(w, http.StatusBadRequest, "InvalidParameterException", "Invalid request format")
			return
		}

		resp, err := api.CreateTaskSet(ctx, &req)
		if err != nil {
			logging.Error("CreateTaskSet failed", "error", err)
			writeErrorResponse(w, http.StatusInternalServerError, "ServerException", err.Error())
			return
		}

		writeJSONResponse(w, resp)

	case "AmazonEC2ContainerServiceV20141113.DeleteTaskSet":
		var req generated.DeleteTaskSetRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logging.Error("Failed to unmarshal DeleteTaskSet request", "error", err)
			writeErrorResponse(w, http.StatusBadRequest, "InvalidParameterException", "Invalid request format")
			return
		}

		resp, err := api.DeleteTaskSet(ctx, &req)
		if err != nil {
			logging.Error("DeleteTaskSet failed", "error", err)
			writeErrorResponse(w, http.StatusInternalServerError, "ServerException", err.Error())
			return
		}

		writeJSONResponse(w, resp)

	case "AmazonEC2ContainerServiceV20141113.DescribeTaskSets":
		var req generated.DescribeTaskSetsRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logging.Error("Failed to unmarshal DescribeTaskSets request", "error", err)
			writeErrorResponse(w, http.StatusBadRequest, "InvalidParameterException", "Invalid request format")
			return
		}

		resp, err := api.DescribeTaskSets(ctx, &req)
		if err != nil {
			logging.Error("DescribeTaskSets failed", "error", err)
			writeErrorResponse(w, http.StatusInternalServerError, "ServerException", err.Error())
			return
		}

		writeJSONResponse(w, resp)

	case "AmazonEC2ContainerServiceV20141113.UpdateTaskSet":
		var req generated.UpdateTaskSetRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logging.Error("Failed to unmarshal UpdateTaskSet request", "error", err)
			writeErrorResponse(w, http.StatusBadRequest, "InvalidParameterException", "Invalid request format")
			return
		}

		resp, err := api.UpdateTaskSet(ctx, &req)
		if err != nil {
			logging.Error("UpdateTaskSet failed", "error", err)
			writeErrorResponse(w, http.StatusInternalServerError, "ServerException", err.Error())
			return
		}

		writeJSONResponse(w, resp)

	case "AmazonEC2ContainerServiceV20141113.UpdateServicePrimaryTaskSet":
		var req generated.UpdateServicePrimaryTaskSetRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logging.Error("Failed to unmarshal UpdateServicePrimaryTaskSet request", "error", err)
			writeErrorResponse(w, http.StatusBadRequest, "InvalidParameterException", "Invalid request format")
			return
		}

		resp, err := api.UpdateServicePrimaryTaskSet(ctx, &req)
		if err != nil {
			logging.Error("UpdateServicePrimaryTaskSet failed", "error", err)
			writeErrorResponse(w, http.StatusInternalServerError, "ServerException", err.Error())
			return
		}

		writeJSONResponse(w, resp)

	default:
		writeErrorResponse(w, http.StatusBadRequest, "InvalidAction", fmt.Sprintf("Unknown TaskSet action: %s", target))
	}
}

// writeJSONResponse writes a JSON response
func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logging.Error("Failed to encode response", "error", err)
		// Response headers already sent, can't change status code
	}
}

// writeErrorResponse writes an error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, errorType, message string) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"__type":  errorType,
		"message": message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Error("Failed to encode error response", "error", err)
	}
}
