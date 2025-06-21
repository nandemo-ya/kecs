package awsclient

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSigner_SignRequest(t *testing.T) {
	// Create test credentials
	creds := Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	// Create signer
	signer := NewSigner(creds, SignerOptions{
		Service: "ecs",
		Region:  "us-east-1",
	})

	// Create test request
	req, err := http.NewRequest("POST", "https://ecs.us-east-1.amazonaws.com/", nil)
	require.NoError(t, err)

	// Sign the request
	err = signer.SignRequest(req)
	require.NoError(t, err)

	// Verify required headers are set
	assert.NotEmpty(t, req.Header.Get("Authorization"))
	assert.NotEmpty(t, req.Header.Get("X-Amz-Date"))
	assert.Contains(t, req.Header.Get("Authorization"), "AWS4-HMAC-SHA256")
	assert.Contains(t, req.Header.Get("Authorization"), "Credential=AKIAIOSFODNN7EXAMPLE")
}

func TestSigner_SignRequestWithSessionToken(t *testing.T) {
	// Create test credentials with session token
	creds := Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "AQoEXAMPLEH4aoAH0gNCAPyJxz4BlCFFxWNE1OPTgk5TthT+FvwqnKwRcOIfrRh3c/LTo6UDdyJwOOvEVPvLXCrrrUtdnniCEXAMPLE/IvU1dYUg2RVAJBanLiHb4IgRmpRV3zrkuWJOgQs8IZZaIv2BXIa2R4OlgkBN9bkUDNCJiBeb/AXlzBBko7b15fjrBs2+cTQtpZ3CYWFXG8C5zqx37wnOE49mRl/+OtkIKGO7fAE",
	}

	// Create signer
	signer := NewSigner(creds, SignerOptions{
		Service: "ecs",
		Region:  "us-west-2",
	})

	// Create test request
	req, err := http.NewRequest("POST", "https://ecs.us-west-2.amazonaws.com/", nil)
	require.NoError(t, err)

	// Sign the request
	err = signer.SignRequest(req)
	require.NoError(t, err)

	// Verify session token header is set
	assert.Equal(t, creds.SessionToken, req.Header.Get("X-Amz-Security-Token"))
}

func TestSigner_CanonicalURI(t *testing.T) {
	signer := &Signer{}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "empty path",
			path:     "",
			expected: "/",
		},
		{
			name:     "root path",
			path:     "/",
			expected: "/",
		},
		{
			name:     "simple path",
			path:     "/test",
			expected: "/test",
		},
		{
			name:     "path with spaces",
			path:     "/test path",
			expected: "/test%20path",
		},
		{
			name:     "path with special chars",
			path:     "/test/path/with/@special",
			expected: "/test/path/with/%40special",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com"+tt.path, nil)
			require.NoError(t, err)
			
			result := signer.canonicalURI(req.URL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSigner_CredentialScope(t *testing.T) {
	signer := NewSigner(Credentials{}, SignerOptions{
		Service: "ecs",
		Region:  "us-east-1",
	})

	// Test date: 2015-08-30T12:36:00Z
	testTime, err := time.Parse(time.RFC3339, "2015-08-30T12:36:00Z")
	require.NoError(t, err)

	scope := signer.credentialScope(testTime)
	assert.Equal(t, "20150830/us-east-1/ecs/aws4_request", scope)
}