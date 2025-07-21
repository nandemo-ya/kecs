package artifacts

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/s3"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

// Manager handles artifact downloads and management
type Manager struct {
	s3Integration s3.Integration
	httpClient    *http.Client
}

// NewManager creates a new artifact manager
func NewManager(s3Integration s3.Integration) *Manager {
	return &Manager{
		s3Integration: s3Integration,
		httpClient: &http.Client{
			Timeout: 0, // No timeout for large downloads
		},
	}
}

// DownloadArtifact downloads an artifact based on its type
func (m *Manager) DownloadArtifact(ctx context.Context, artifact *types.Artifact) (err error) {
	if artifact.ArtifactUrl == nil || artifact.TargetPath == nil {
		return fmt.Errorf("artifact URL and target path are required")
	}

	artifactURL := *artifact.ArtifactUrl
	targetPath := *artifact.TargetPath

	logging.Debug("Downloading artifact", "from", artifactURL, "to", targetPath)

	// Create target directory if it doesn't exist
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
	}

	// Download based on URL scheme
	if strings.HasPrefix(artifactURL, "s3://") {
		return m.downloadFromS3(ctx, artifact)
	} else if strings.HasPrefix(artifactURL, "http://") || strings.HasPrefix(artifactURL, "https://") {
		return m.downloadFromHTTP(ctx, artifact)
	}

	return fmt.Errorf("unsupported artifact URL scheme: %s", artifactURL)
}

// downloadFromS3 downloads an artifact from S3
func (m *Manager) downloadFromS3(ctx context.Context, artifact *types.Artifact) error {
	artifactURL := *artifact.ArtifactUrl
	targetPath := *artifact.TargetPath

	logging.Debug("Downloading artifact from S3", "url", artifactURL)

	// Parse S3 URL (s3://bucket/key)
	s3URL := strings.TrimPrefix(artifactURL, "s3://")
	parts := strings.SplitN(s3URL, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid S3 URL format: %s", artifactURL)
	}

	bucket := parts[0]
	key := parts[1]

	// Download from S3 using integration
	reader, err := m.s3Integration.DownloadFile(ctx, bucket, key)
	if err != nil {
		return fmt.Errorf("failed to download from S3: %w", err)
	}
	defer reader.Close()

	// Create target file
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", targetPath, err)
	}
	defer file.Close()

	// Copy S3 data to file
	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write artifact to %s: %w", targetPath, err)
	}

	// Set permissions if specified
	if artifact.Permissions != nil {
		if err := m.setFilePermissions(targetPath, *artifact.Permissions); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	// Validate checksum if specified
	if artifact.Checksum != nil {
		if err := m.validateChecksum(targetPath, *artifact.Checksum, artifact.ChecksumType); err != nil {
			return fmt.Errorf("checksum validation failed: %w", err)
		}
	}

	logging.Debug("Successfully downloaded S3 artifact", "targetPath", targetPath)
	return nil
}

// downloadFromHTTP downloads an artifact from HTTP/HTTPS
func (m *Manager) downloadFromHTTP(ctx context.Context, artifact *types.Artifact) error {
	artifactURL := *artifact.ArtifactUrl
	targetPath := *artifact.TargetPath

	logging.Debug("Downloading artifact from HTTP", "url", artifactURL)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", artifactURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Execute request
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download from %s: %w", artifactURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Create target file
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", targetPath, err)
	}
	defer file.Close()

	// Copy response body to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write artifact to %s: %w", targetPath, err)
	}

	// Set permissions if specified
	if artifact.Permissions != nil {
		if err := m.setFilePermissions(targetPath, *artifact.Permissions); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	// Validate checksum if specified
	if artifact.Checksum != nil {
		if err := m.validateChecksum(targetPath, *artifact.Checksum, artifact.ChecksumType); err != nil {
			return fmt.Errorf("checksum validation failed: %w", err)
		}
	}

	logging.Debug("Successfully downloaded HTTP artifact", "targetPath", targetPath)
	return nil
}

// setFilePermissions sets file permissions
func (m *Manager) setFilePermissions(filePath, permissions string) error {
	// Convert octal string to file mode
	var mode os.FileMode
	_, err := fmt.Sscanf(permissions, "%o", &mode)
	if err != nil {
		return fmt.Errorf("invalid permissions format %s: %w", permissions, err)
	}

	if err := os.Chmod(filePath, mode); err != nil {
		return fmt.Errorf("failed to set permissions %s on %s: %w", permissions, filePath, err)
	}

	logging.Debug("Set file permissions", "permissions", permissions, "filePath", filePath)
	return nil
}

// validateChecksum validates file checksum
func (m *Manager) validateChecksum(filePath, expectedChecksum string, checksumType *string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum validation: %w", err)
	}
	defer file.Close()

	var hash string
	checksumMethod := "sha256" // default
	if checksumType != nil {
		checksumMethod = strings.ToLower(*checksumType)
	}

	switch checksumMethod {
	case "sha256":
		hasher := sha256.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return fmt.Errorf("failed to compute SHA256 hash: %w", err)
		}
		hash = hex.EncodeToString(hasher.Sum(nil))
	case "md5":
		hasher := md5.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return fmt.Errorf("failed to compute MD5 hash: %w", err)
		}
		hash = hex.EncodeToString(hasher.Sum(nil))
	default:
		return fmt.Errorf("unsupported checksum type: %s", checksumMethod)
	}

	if hash != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, hash)
	}

	logging.Debug("Checksum validation passed", "filePath", filePath)
	return nil
}

// GetArtifactScript generates a shell script for downloading artifacts in init containers
func (m *Manager) GetArtifactScript(artifacts []types.Artifact) string {
	if len(artifacts) == 0 {
		return ""
	}

	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	script.WriteString("set -e\n\n")

	for _, artifact := range artifacts {
		if artifact.ArtifactUrl == nil || artifact.TargetPath == nil {
			continue
		}

		artifactURL := *artifact.ArtifactUrl
		targetPath := *artifact.TargetPath
		fullPath := "/artifacts/" + targetPath

		// Create directory
		script.WriteString(fmt.Sprintf("mkdir -p $(dirname %s)\n", fullPath))

		// Download based on URL scheme
		if strings.HasPrefix(artifactURL, "s3://") {
			// S3 download - placeholder for now
			script.WriteString(fmt.Sprintf("echo 'Downloading %s from S3 to %s'\n", artifactURL, fullPath))
			script.WriteString(fmt.Sprintf("echo 'S3 download placeholder' > %s\n", fullPath))
		} else if strings.HasPrefix(artifactURL, "http://") || strings.HasPrefix(artifactURL, "https://") {
			// HTTP/HTTPS download
			script.WriteString(fmt.Sprintf("wget -O %s %s\n", fullPath, artifactURL))
		}

		// Set permissions if specified
		if artifact.Permissions != nil {
			script.WriteString(fmt.Sprintf("chmod %s %s\n", *artifact.Permissions, fullPath))
		}

		script.WriteString("\n")
	}

	return script.String()
}
