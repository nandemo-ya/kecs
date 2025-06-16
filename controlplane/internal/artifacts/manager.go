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
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
	"k8s.io/klog/v2"
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

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Download based on type
	var reader io.ReadCloser

	artifactType := m.detectArtifactType(artifactURL, artifact.Type)
	
	switch artifactType {
	case "s3":
		reader, err = m.downloadFromS3(ctx, artifactURL)
	case "http", "https":
		reader, err = m.downloadFromHTTP(ctx, artifactURL)
	default:
		return fmt.Errorf("unsupported artifact type: %s", artifactType)
	}

	if err != nil {
		return fmt.Errorf("failed to download artifact: %w", err)
	}
	defer reader.Close()

	// Create temporary file
	tempFile, tempErr := os.CreateTemp(targetDir, ".download-*")
	if tempErr != nil {
		return fmt.Errorf("failed to create temp file: %w", tempErr)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // Clean up temp file on error

	// Download to temp file with optional checksum validation
	var hasher io.Writer
	if artifact.Checksum != nil && artifact.ChecksumType != nil {
		switch strings.ToLower(*artifact.ChecksumType) {
		case "sha256":
			h := sha256.New()
			hasher = h
			defer func() {
				if err == nil {
					checksum := hex.EncodeToString(h.Sum(nil))
					if checksum != *artifact.Checksum {
						err = fmt.Errorf("checksum mismatch: expected %s, got %s", *artifact.Checksum, checksum)
					}
				}
			}()
		case "md5":
			h := md5.New()
			hasher = h
			defer func() {
				if err == nil {
					checksum := hex.EncodeToString(h.Sum(nil))
					if checksum != *artifact.Checksum {
						err = fmt.Errorf("checksum mismatch: expected %s, got %s", *artifact.Checksum, checksum)
					}
				}
			}()
		default:
			return fmt.Errorf("unsupported checksum type: %s", *artifact.ChecksumType)
		}
	}

	// Copy data
	var writer io.Writer = tempFile
	if hasher != nil {
		writer = io.MultiWriter(tempFile, hasher)
	}

	if _, copyErr := io.Copy(writer, reader); copyErr != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write artifact: %w", copyErr)
	}
	tempFile.Close()

	// Set permissions if specified
	if artifact.Permissions != nil {
		mode, parseErr := parseFileMode(*artifact.Permissions)
		if parseErr != nil {
			return fmt.Errorf("invalid permissions: %w", parseErr)
		}
		if chmodErr := os.Chmod(tempPath, mode); chmodErr != nil {
			return fmt.Errorf("failed to set permissions: %w", chmodErr)
		}
	}

	// Move temp file to final location
	if renameErr := os.Rename(tempPath, targetPath); renameErr != nil {
		// If rename fails (e.g., across filesystems), try copy
		if copyErr := copyFile(tempPath, targetPath); copyErr != nil {
			return fmt.Errorf("failed to move artifact to target: %w", copyErr)
		}
		os.Remove(tempPath)
	}

	klog.Infof("Successfully downloaded artifact to: %s", targetPath)
	return nil
}

// detectArtifactType detects the artifact type from URL or explicit type
func (m *Manager) detectArtifactType(artifactURL string, explicitType *string) string {
	if explicitType != nil {
		return strings.ToLower(*explicitType)
	}

	// Auto-detect from URL
	if strings.HasPrefix(artifactURL, "s3://") {
		return "s3"
	} else if strings.HasPrefix(artifactURL, "http://") {
		return "http"
	} else if strings.HasPrefix(artifactURL, "https://") {
		return "https"
	}

	// Default to https for URLs without scheme
	return "https"
}

// downloadFromS3 downloads an artifact from S3
func (m *Manager) downloadFromS3(ctx context.Context, s3URL string) (io.ReadCloser, error) {
	// Parse S3 URL: s3://bucket/key
	if !strings.HasPrefix(s3URL, "s3://") {
		return nil, fmt.Errorf("invalid S3 URL format: %s", s3URL)
	}

	s3Path := strings.TrimPrefix(s3URL, "s3://")
	parts := strings.SplitN(s3Path, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid S3 URL format: %s", s3URL)
	}

	bucket := parts[0]
	key := parts[1]

	return m.s3Integration.DownloadFile(ctx, bucket, key)
}

// downloadFromHTTP downloads an artifact from HTTP/HTTPS
func (m *Manager) downloadFromHTTP(ctx context.Context, artifactURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", artifactURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	return resp.Body, nil
}

// parseFileMode parses file permissions string (e.g., "0644") to os.FileMode
func parseFileMode(permissions string) (os.FileMode, error) {
	// Parse as octal
	mode, err := parseOctal(permissions)
	if err != nil {
		return 0, err
	}
	return os.FileMode(mode), nil
}

// parseOctal parses an octal string to uint32
func parseOctal(s string) (uint32, error) {
	// Remove leading zeros
	s = strings.TrimLeft(s, "0")
	if s == "" {
		return 0, nil
	}

	var result uint32
	for _, c := range s {
		if c < '0' || c > '7' {
			return 0, fmt.Errorf("invalid octal digit: %c", c)
		}
		result = result*8 + uint32(c-'0')
	}
	return result, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Sync()
}

// CleanupArtifacts removes downloaded artifacts
func (m *Manager) CleanupArtifacts(artifacts []types.Artifact) {
	for _, artifact := range artifacts {
		if artifact.TargetPath != nil {
			if err := os.Remove(*artifact.TargetPath); err != nil && !os.IsNotExist(err) {
				klog.Warningf("Failed to cleanup artifact %s: %v", *artifact.TargetPath, err)
			}
		}
	}
}