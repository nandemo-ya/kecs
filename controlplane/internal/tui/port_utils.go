// Copyright 2025 The KECS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tui

import (
	"fmt"
	"net"
	"sort"
)

// FindAvailablePort finds an available port starting from the given port
func FindAvailablePort(startPort int) int {
	for port := startPort; port < 65535; port++ {
		if IsPortAvailable(port) {
			return port
		}
	}
	return 0
}

// IsPortAvailable checks if a port is available for binding
func IsPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// SuggestAvailablePorts suggests available API and Admin ports based on existing instances
func SuggestAvailablePorts(instances []Instance, preferredAPIPort, preferredAdminPort int) (apiPort, adminPort int) {
	// Collect all used ports
	usedPorts := make(map[int]bool)
	for _, inst := range instances {
		usedPorts[inst.APIPort] = true
		usedPorts[inst.APIPort+1] = true // Reserve adjacent port for admin
	}

	// Find available API port
	apiPort = preferredAPIPort
	if usedPorts[apiPort] || !IsPortAvailable(apiPort) {
		// Try common alternatives
		alternatives := []int{8080, 8090, 8100, 8110, 8120, 8130, 8140, 8150}
		for _, port := range alternatives {
			if !usedPorts[port] && IsPortAvailable(port) {
				apiPort = port
				break
			}
		}

		// If still not found, scan from 8080
		if usedPorts[apiPort] || !IsPortAvailable(apiPort) {
			apiPort = FindAvailablePort(8080)
		}
	}

	// Admin port is typically API port + 1
	adminPort = apiPort + 1
	if usedPorts[adminPort] || !IsPortAvailable(adminPort) {
		adminPort = FindAvailablePort(apiPort + 1)
	}

	return apiPort, adminPort
}

// GetNextInstanceNumber gets the next available instance number for naming
func GetNextInstanceNumber(instances []Instance, prefix string) int {
	numbers := make([]int, 0)

	// Extract numbers from instance names with the given prefix
	for _, inst := range instances {
		if len(inst.Name) > len(prefix) && inst.Name[:len(prefix)] == prefix {
			// Try to parse the number suffix
			var num int
			suffix := inst.Name[len(prefix):]
			if suffix[0] == '-' && len(suffix) > 1 {
				if _, err := fmt.Sscanf(suffix, "-%d", &num); err == nil {
					numbers = append(numbers, num)
				}
			}
		}
	}

	if len(numbers) == 0 {
		return 1
	}

	// Sort numbers and find the first gap
	sort.Ints(numbers)
	for i := 0; i < len(numbers); i++ {
		if i+1 != numbers[i] {
			return i + 1
		}
	}

	return numbers[len(numbers)-1] + 1
}
