package utils

import (
	"encoding/json"
	"fmt"
)

// Legacy service methods for backward compatibility

// CreateService (legacy) creates a service using map format
func (c *ECSClient) CreateService(config map[string]interface{}) (map[string]interface{}, error) {
	// Convert config to JSON and use executeCurl directly to pass all parameters
	jsonData, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	output, err := c.CurlClient.executeCurl("CreateService", string(jsonData))
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateService (legacy) updates a service using map format
func (c *ECSClient) UpdateService(config map[string]interface{}) (map[string]interface{}, error) {
	// Convert config to JSON and use executeCurl directly to pass all parameters
	jsonData, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	output, err := c.CurlClient.executeCurl("UpdateService", string(jsonData))
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteServiceForce (legacy) deletes a service forcefully and returns result
func (c *ECSClient) DeleteServiceForce(cluster, service string) (map[string]interface{}, error) {
	err := c.CurlClient.DeleteService(cluster, service)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"service": map[string]interface{}{
			"serviceName": service,
			"status":      "DRAINING",
		},
	}, nil
}

// DeleteService (legacy) deletes a service and returns result
func (c *ECSClient) DeleteService(cluster, service string) (map[string]interface{}, error) {
	err := c.CurlClient.DeleteService(cluster, service)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"service": map[string]interface{}{
			"serviceName": service,
			"status":      "DRAINING",
		},
	}, nil
}

// Legacy task methods

// RunTask (legacy) runs a task using map format
func (c *ECSClient) RunTask(config map[string]interface{}) (map[string]interface{}, error) {
	cluster, _ := config["cluster"].(string)
	taskDef, _ := config["taskDefinition"].(string)
	count := 1
	if cnt, ok := config["count"].(float64); ok {
		count = int(cnt)
	} else if cnt, ok := config["count"].(int); ok {
		count = cnt
	}

	result, err := c.CurlClient.RunTask(cluster, taskDef, count)
	if err != nil {
		return nil, err
	}

	// Convert to map
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(resultJSON, &resultMap); err != nil {
		return nil, err
	}

	return resultMap, nil
}

// DescribeService (legacy) describes a service using map format
func (c *ECSClient) DescribeService(cluster, service string) (map[string]interface{}, error) {
	// Use executeCurl directly to get full response
	payload := fmt.Sprintf(`{
		"cluster": "%s",
		"services": ["%s"]
	}`, cluster, service)
	
	output, err := c.CurlClient.executeCurl("DescribeServices", payload)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	// Extract the first service from the services array
	if services, ok := result["services"].([]interface{}); ok && len(services) > 0 {
		if serviceMap, ok := services[0].(map[string]interface{}); ok {
			return map[string]interface{}{
				"service": serviceMap,
			}, nil
		}
	}

	return nil, fmt.Errorf("service not found")
}

// DescribeTasks (legacy) with map interface support
func (c *ECSClient) DescribeTasks(cluster string, taskArns []string) (map[string]interface{}, error) {
	tasks, err := c.CurlClient.DescribeTasks(cluster, taskArns)
	if err != nil {
		return nil, err
	}

	// Convert tasks to map array
	var tasksArray []map[string]interface{}
	for _, task := range tasks {
		taskJSON, err := json.Marshal(task)
		if err != nil {
			return nil, err
		}
		
		var taskMap map[string]interface{}
		if err := json.Unmarshal(taskJSON, &taskMap); err != nil {
			return nil, err
		}
		tasksArray = append(tasksArray, taskMap)
	}

	return map[string]interface{}{
		"tasks": tasksArray,
	}, nil
}

// ListTasks (legacy) with map interface support
func (c *ECSClient) ListTasks(cluster string, params map[string]interface{}) (map[string]interface{}, error) {
	serviceName, _ := params["serviceName"].(string)
	desiredStatus, _ := params["desiredStatus"].(string)
	
	// Use executeCurl directly to pass all parameters
	payload := fmt.Sprintf(`{"cluster": "%s"`, cluster)
	if serviceName != "" {
		payload += fmt.Sprintf(`, "serviceName": "%s"`, serviceName)
	}
	if desiredStatus != "" {
		payload += fmt.Sprintf(`, "desiredStatus": "%s"`, desiredStatus)
	}
	payload += "}"
	
	output, err := c.CurlClient.executeCurl("ListTasks", payload)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	// Convert taskArns from []string to []interface{} for backward compatibility
	if _, ok := result["taskArns"].([]interface{}); ok {
		// Already in the expected format
		return result, nil
	} else if taskArnsStr, ok := result["taskArns"].([]string); ok {
		// Convert []string to []interface{}
		taskArnsInterface := make([]interface{}, len(taskArnsStr))
		for i, arn := range taskArnsStr {
			taskArnsInterface[i] = arn
		}
		result["taskArns"] = taskArnsInterface
	}

	return result, nil
}

// StopTask (legacy) stops a task
func (c *ECSClient) StopTask(cluster, task, reason string) (map[string]interface{}, error) {
	err := c.CurlClient.StopTask(cluster, task, reason)
	if err != nil {
		return nil, err
	}

	// Get task details after stopping
	tasks, err := c.CurlClient.DescribeTasks(cluster, []string{task})
	if err != nil {
		return nil, err
	}

	if len(tasks) > 0 {
		taskJSON, err := json.Marshal(tasks[0])
		if err != nil {
			return nil, err
		}

		var taskMap map[string]interface{}
		if err := json.Unmarshal(taskJSON, &taskMap); err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"task": taskMap,
		}, nil
	}

	return map[string]interface{}{}, nil
}

// DescribeServices (legacy) describes multiple services
func (c *ECSClient) DescribeServices(cluster string, services []string) (map[string]interface{}, error) {
	var serviceResults []map[string]interface{}
	
	for _, serviceName := range services {
		svc, err := c.CurlClient.DescribeService(cluster, serviceName)
		if err != nil {
			// Add failure instead of erroring out
			continue
		}
		
		svcJSON, _ := json.Marshal(svc)
		var svcMap map[string]interface{}
		json.Unmarshal(svcJSON, &svcMap)
		serviceResults = append(serviceResults, svcMap)
	}
	
	return map[string]interface{}{
		"services": serviceResults,
	}, nil
}

// ListServices (legacy) lists services with map result
func (c *ECSClient) ListServices(cluster string) (map[string]interface{}, error) {
	arns, err := c.CurlClient.ListServices(cluster)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"serviceArns": arns,
	}, nil
}