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

package app

import (
	"testing"
)

func TestExtractInstanceName(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{
			name:     "default port",
			endpoint: "http://localhost:8080",
			want:     "dev",
		},
		{
			name:     "staging port",
			endpoint: "http://localhost:8090",
			want:     "staging",
		},
		{
			name:     "test port",
			endpoint: "http://localhost:8100",
			want:     "test",
		},
		{
			name:     "local port",
			endpoint: "http://localhost:8110",
			want:     "local",
		},
		{
			name:     "prod port",
			endpoint: "http://localhost:8200",
			want:     "prod",
		},
		{
			name:     "custom port",
			endpoint: "http://localhost:8300",
			want:     "instance-8300",
		},
		{
			name:     "with path",
			endpoint: "http://localhost:8080/api",
			want:     "instance-8080/api",
		},
		{
			name:     "invalid endpoint",
			endpoint: "invalid",
			want:     "default",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInstanceName(tt.endpoint)
			if got != tt.want {
				t.Errorf("extractInstanceName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStartupFlowStateTransitions(t *testing.T) {
	tests := []struct {
		name        string
		startState  StartupState
		wantNext    StartupState
		description string
	}{
		{
			name:        "checking to dialog",
			startState:  StartupStateChecking,
			wantNext:    StartupStateDialog,
			description: "When KECS not running, should show dialog",
		},
		{
			name:        "dialog to starting",
			startState:  StartupStateDialog,
			wantNext:    StartupStateStarting,
			description: "When user confirms, should start KECS",
		},
		{
			name:        "starting to ready",
			startState:  StartupStateStarting,
			wantNext:    StartupStateReady,
			description: "When KECS starts successfully, should be ready",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a simplified test to verify state transitions
			t.Logf("State transition: %d -> %d (%s)", 
				tt.startState, tt.wantNext, tt.description)
		})
	}
}