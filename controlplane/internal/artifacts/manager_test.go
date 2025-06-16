package artifacts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/nandemo-ya/kecs/controlplane/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtifactManager_DownloadHTTPArtifact(t *testing.T) {
	// Create test server
	testContent := "test artifact content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "artifact-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create artifact manager
	am, err := NewArtifactManager(filepath.Join(tempDir, "cache"))
	require.NoError(t, err)

	// Test artifact
	artifact := types.Artifact{
		Name:        stringPtr("test-artifact"),
		ArtifactUrl: stringPtr(server.URL + "/test.txt"),
		TargetPath:  stringPtr("artifacts/test.txt"),
		Type:        stringPtr("http"),
	}

	// Download artifact
	ctx := context.Background()
	workDir := filepath.Join(tempDir, "work")
	err = am.DownloadArtifacts(ctx, []types.Artifact{artifact}, workDir)
	require.NoError(t, err)

	// Verify file exists
	targetPath := filepath.Join(workDir, "artifacts/test.txt")
	assert.FileExists(t, targetPath)

	// Verify content
	content, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))

	// Cleanup
	err = am.CleanupArtifacts(workDir)
	require.NoError(t, err)
	assert.NoDirExists(t, workDir)
}

func TestArtifactManager_ValidateChecksum(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "artifact-checksum-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "test content"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Create artifact manager
	am, err := NewArtifactManager(filepath.Join(tempDir, "cache"))
	require.NoError(t, err)

	tests := []struct {
		name         string
		checksumType string
		checksum     string
		expectError  bool
	}{
		{
			name:         "valid sha256",
			checksumType: "sha256",
			checksum:     "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72",
			expectError:  false,
		},
		{
			name:         "invalid sha256",
			checksumType: "sha256",
			checksum:     "invalid",
			expectError:  true,
		},
		{
			name:         "valid md5",
			checksumType: "md5",
			checksum:     "9473fdd0d880a43c21b7778d34872157",
			expectError:  false,
		},
		{
			name:         "unsupported type",
			checksumType: "sha1",
			checksum:     "any",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := am.validateChecksum(testFile, tt.checksum, tt.checksumType)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestArtifactManager_DetermineArtifactType(t *testing.T) {
	am := &ArtifactManager{}

	tests := []struct {
		name         string
		url          string
		explicitType *string
		expected     string
	}{
		{
			name:     "s3 URL",
			url:      "s3://my-bucket/path/to/file.txt",
			expected: "s3",
		},
		{
			name:     "http URL",
			url:      "http://example.com/file.txt",
			expected: "http",
		},
		{
			name:     "https URL",
			url:      "https://example.com/file.txt",
			expected: "https",
		},
		{
			name:         "explicit type overrides",
			url:          "https://example.com/file.txt",
			explicitType: stringPtr("s3"),
			expected:     "s3",
		},
		{
			name:     "unknown type",
			url:      "ftp://example.com/file.txt",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := am.determineArtifactType(tt.url, tt.explicitType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFileMode(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		expected    os.FileMode
		expectError bool
	}{
		{
			name:     "valid mode with leading zero",
			mode:     "0644",
			expected: 0644,
		},
		{
			name:     "valid mode without leading zero",
			mode:     "755",
			expected: 0755,
		},
		{
			name:        "invalid mode",
			mode:        "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFileMode(tt.mode)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestArtifactManager_SetS3Endpoint(t *testing.T) {
	// Create artifact manager
	am, err := NewArtifactManager("/tmp/cache")
	require.NoError(t, err)

	// Set S3 endpoint
	endpoint := "http://localhost:4566"
	am.SetS3Endpoint(endpoint)

	// Verify endpoint is set
	assert.Equal(t, endpoint, am.s3Endpoint)
	assert.Nil(t, am.s3Client) // Client should be nil, will be initialized on first use
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}