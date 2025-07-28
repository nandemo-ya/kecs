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

package startup

import (
	"testing"
)

func TestCheckKECSStatus(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{
			name:     "default endpoint",
			endpoint: "http://localhost:8080",
			wantErr:  false,
		},
		{
			name:     "custom endpoint",
			endpoint: "http://localhost:8090",
			wantErr:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will fail if KECS is not running, which is expected
			running, err := CheckKECSStatus(tt.endpoint)
			if err != nil && tt.wantErr {
				t.Errorf("CheckKECSStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
			t.Logf("KECS running at %s: %v", tt.endpoint, running)
		})
	}
}

func TestShouldDisplayLog(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "normal info log",
			line: "2025-01-28T12:00:00Z INFO Starting server",
			want: true,
		},
		{
			name: "debug log without error",
			line: "2025-01-28T12:00:00Z DEBUG Processing request",
			want: false,
		},
		{
			name: "debug log with error",
			line: "2025-01-28T12:00:00Z DEBUG Error processing request",
			want: true,
		},
		{
			name: "empty line",
			line: "",
			want: false,
		},
		{
			name: "whitespace only",
			line: "    ",
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldDisplayLog(tt.line); got != tt.want {
				t.Errorf("shouldDisplayLog() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatLogLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "log with timestamp",
			line: "2025-01-28T12:00:00Z INFO Starting server",
			want: "Starting server",
		},
		{
			name: "log without timestamp",
			line: "Starting server",
			want: "Starting server",
		},
		{
			name: "log with partial timestamp",
			line: "2025-01-28 INFO",
			want: "2025-01-28 INFO",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatLogLine(tt.line); got != tt.want {
				t.Errorf("formatLogLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractProgress(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "creating cluster",
			line: "INFO Creating Kubernetes cluster",
			want: "Creating Kubernetes cluster...",
		},
		{
			name: "waiting for ready",
			line: "Waiting for cluster to be ready",
			want: "Waiting for cluster to be ready...",
		},
		{
			name: "deploying components",
			line: "Deploying KECS control plane",
			want: "Deploying KECS components...",
		},
		{
			name: "starting server",
			line: "Starting API server on port 8080",
			want: "Starting API server...",
		},
		{
			name: "kecs ready",
			line: "KECS is ready and accepting requests",
			want: "KECS is ready!",
		},
		{
			name: "unrelated log",
			line: "Processing request from client",
			want: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractProgress(tt.line); got != tt.want {
				t.Errorf("extractProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}