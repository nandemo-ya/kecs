package artifacts

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
	"k8s.io/klog/v2"
)

// Manager handles artifact downloads for containers
type Manager interface {
	// DownloadArtifacts downloads all artifacts for a container
	DownloadArtifacts(ctx context.Context, artifacts []types.Artifact, workDir string) error
	// CleanupArtifacts removes downloaded artifacts
	CleanupArtifacts(workDir string) error
	// SetS3Endpoint sets custom S3 endpoint for LocalStack integration
	SetS3Endpoint(endpoint string)
}

// ArtifactManager implements the Manager interface
type ArtifactManager struct {
	httpClient      *http.Client
	s3Client        *s3.Client
	s3Endpoint      string
	cacheDir        string
	downloadTimeout time.Duration
}

// NewArtifactManager creates a new artifact manager
func NewArtifactManager(cacheDir string) (*ArtifactManager, error) {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 5 * time.Minute,
	}

	return &ArtifactManager{
		httpClient:      httpClient,
		cacheDir:        cacheDir,
		downloadTimeout: 5 * time.Minute,
	}, nil
}

// SetS3Endpoint sets custom S3 endpoint for LocalStack integration
func (am *ArtifactManager) SetS3Endpoint(endpoint string) {
	am.s3Endpoint = endpoint
	// Reinitialize S3 client with new endpoint
	am.s3Client = nil
}

// initS3Client initializes the S3 client with custom endpoint if set
func (am *ArtifactManager) initS3Client(ctx context.Context) error {
	if am.s3Client != nil {
		return nil
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Override endpoint if set (for LocalStack)
	if am.s3Endpoint != "" {
		cfg.BaseEndpoint = aws.String(am.s3Endpoint)
		klog.V(3).Infof("Using custom S3 endpoint: %s", am.s3Endpoint)
	}

	// Create S3 client
	am.s3Client = s3.NewFromConfig(cfg)
	return nil
}

// DownloadArtifacts downloads all artifacts for a container
func (am *ArtifactManager) DownloadArtifacts(ctx context.Context, artifacts []types.Artifact, workDir string) error {
	if len(artifacts) == 0 {
		return nil
	}

	klog.V(2).Infof("Downloading %d artifacts to %s", len(artifacts), workDir)

	// Create work directory
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	// Download each artifact
	for _, artifact := range artifacts {
		if err := am.downloadArtifact(ctx, artifact, workDir); err != nil {
			return fmt.Errorf("failed to download artifact %s: %w", *artifact.Name, err)
		}
	}

	return nil
}

// downloadArtifact downloads a single artifact
func (am *ArtifactManager) downloadArtifact(ctx context.Context, artifact types.Artifact, workDir string) error {
	if artifact.ArtifactUrl == nil || artifact.TargetPath == nil {
		return fmt.Errorf("artifact URL and target path are required")
	}

	artifactURL := *artifact.ArtifactUrl
	targetPath := filepath.Join(workDir, *artifact.TargetPath)

	klog.V(3).Infof("Downloading artifact from %s to %s", artifactURL, targetPath)

	// Create target directory
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Determine artifact type
	artifactType := am.determineArtifactType(artifactURL, artifact.Type)

	// Download based on type
	var err error
	switch artifactType {
	case "s3":
		err = am.downloadS3Artifact(ctx, artifactURL, targetPath)
	case "http", "https":
		err = am.downloadHTTPArtifact(ctx, artifactURL, targetPath)
	default:
		return fmt.Errorf("unsupported artifact type: %s", artifactType)
	}

	if err != nil {
		return err
	}

	// Validate checksum if provided
	if artifact.Checksum != nil && artifact.ChecksumType != nil {
		if err := am.validateChecksum(targetPath, *artifact.Checksum, *artifact.ChecksumType); err != nil {
			// Remove invalid file
			os.Remove(targetPath)
			return fmt.Errorf("checksum validation failed: %w", err)
		}
	}

	// Set permissions if specified
	if artifact.Permissions != nil {
		mode, err := parseFileMode(*artifact.Permissions)
		if err != nil {
			return fmt.Errorf("invalid permissions: %w", err)
		}
		if err := os.Chmod(targetPath, mode); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	klog.V(3).Infof("Successfully downloaded artifact to %s", targetPath)
	return nil
}

// determineArtifactType determines the type of artifact from URL
func (am *ArtifactManager) determineArtifactType(artifactURL string, explicitType *string) string {
	if explicitType != nil {
		return *explicitType
	}

	// Parse URL to determine type
	if strings.HasPrefix(artifactURL, "s3://") {
		return "s3"
	} else if strings.HasPrefix(artifactURL, "http://") {
		return "http"
	} else if strings.HasPrefix(artifactURL, "https://") {
		return "https"
	}

	return "unknown"
}

// downloadS3Artifact downloads an artifact from S3
func (am *ArtifactManager) downloadS3Artifact(ctx context.Context, s3URL, targetPath string) error {
	// Initialize S3 client if needed
	if err := am.initS3Client(ctx); err != nil {
		return err
	}

	// Parse S3 URL
	u, err := url.Parse(s3URL)
	if err != nil {
		return fmt.Errorf("invalid S3 URL: %w", err)
	}

	bucket := u.Host
	key := strings.TrimPrefix(u.Path, "/")

	klog.V(4).Infof("Downloading from S3 bucket=%s key=%s", bucket, key)

	// Download from S3
	result, err := am.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download from S3: %w", err)
	}
	defer result.Body.Close()

	// Create target file
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer file.Close()

	// Copy content
	_, err = io.Copy(file, result.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// downloadHTTPArtifact downloads an artifact via HTTP/HTTPS
func (am *ArtifactManager) downloadHTTPArtifact(ctx context.Context, artifactURL, targetPath string) error {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", artifactURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Download file
	resp, err := am.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create target file
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer file.Close()

	// Copy content
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// validateChecksum validates file checksum
func (am *ArtifactManager) validateChecksum(filePath, expectedChecksum, checksumType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var actualChecksum string

	switch strings.ToLower(checksumType) {
	case "sha256":
		h := sha256.New()
		if _, err := io.Copy(h, file); err != nil {
			return fmt.Errorf("failed to calculate checksum: %w", err)
		}
		actualChecksum = hex.EncodeToString(h.Sum(nil))

	case "md5":
		h := md5.New()
		if _, err := io.Copy(h, file); err != nil {
			return fmt.Errorf("failed to calculate checksum: %w", err)
		}
		actualChecksum = hex.EncodeToString(h.Sum(nil))

	default:
		return fmt.Errorf("unsupported checksum type: %s", checksumType)
	}

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// CleanupArtifacts removes downloaded artifacts
func (am *ArtifactManager) CleanupArtifacts(workDir string) error {
	if workDir == "" || workDir == "/" {
		return fmt.Errorf("invalid work directory")
	}

	klog.V(3).Infof("Cleaning up artifacts in %s", workDir)

	// Remove work directory
	if err := os.RemoveAll(workDir); err != nil {
		return fmt.Errorf("failed to cleanup artifacts: %w", err)
	}

	return nil
}

// parseFileMode parses file mode string (e.g., "0644") to os.FileMode
func parseFileMode(mode string) (os.FileMode, error) {
	// Remove leading zero if present
	mode = strings.TrimPrefix(mode, "0")

	// Parse as octal
	parsed, err := fmt.Sscanf(mode, "%o", new(int))
	if err != nil || parsed != 1 {
		return 0, fmt.Errorf("invalid file mode: %s", mode)
	}

	var fileMode os.FileMode
	fmt.Sscanf(mode, "%o", &fileMode)
	return fileMode, nil
}