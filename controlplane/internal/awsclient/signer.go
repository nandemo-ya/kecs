package awsclient

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	// AWS4-HMAC-SHA256 is the signing algorithm
	algorithm = "AWS4-HMAC-SHA256"
	
	// DateFormat is the ISO8601 date format
	dateFormat = "20060102T150405Z"
	
	// ShortDateFormat is the short date format
	shortDateFormat = "20060102"
)

// Credentials holds AWS credentials
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// SignerOptions contains options for the signer
type SignerOptions struct {
	Service string
	Region  string
}

// Signer signs AWS requests using Signature Version 4
type Signer struct {
	credentials Credentials
	options     SignerOptions
}

// NewSigner creates a new Signer
func NewSigner(creds Credentials, opts SignerOptions) *Signer {
	return &Signer{
		credentials: creds,
		options:     opts,
	}
}

// SignRequest signs an HTTP request with AWS Signature Version 4
func (s *Signer) SignRequest(req *http.Request) error {
	// Ensure the request has a body that can be read multiple times
	body, err := s.getRequestBody(req)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Set the time
	now := time.Now().UTC()
	req.Header.Set("X-Amz-Date", now.Format(dateFormat))

	// Add session token if present
	if s.credentials.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", s.credentials.SessionToken)
	}

	// Calculate the signature
	signature, signedHeaders := s.calculateSignature(req, body, now)

	// Add the Authorization header
	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		s.credentials.AccessKeyID,
		s.credentialScope(now),
		signedHeaders,
		signature,
	)
	req.Header.Set("Authorization", authHeader)

	return nil
}

// getRequestBody reads the request body and returns it as a byte slice
func (s *Signer) getRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return []byte{}, nil
	}

	// Read the body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	// Reset the body so it can be read again
	req.Body = io.NopCloser(bytes.NewReader(body))

	return body, nil
}

// calculateSignature calculates the AWS Signature Version 4
func (s *Signer) calculateSignature(req *http.Request, body []byte, t time.Time) (string, string) {
	// Step 1: Create canonical request
	canonicalRequest, signedHeaders := s.createCanonicalRequest(req, body)

	// Step 2: Create string to sign
	stringToSign := s.createStringToSign(canonicalRequest, t)

	// Step 3: Calculate signing key
	signingKey := s.getSigningKey(t)

	// Step 4: Calculate signature
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	return signature, signedHeaders
}

// createCanonicalRequest creates the canonical request string
func (s *Signer) createCanonicalRequest(req *http.Request, body []byte) (string, string) {
	// Canonical URI
	canonicalURI := s.canonicalURI(req.URL)

	// Canonical query string
	canonicalQueryString := s.canonicalQueryString(req.URL)

	// Canonical headers and signed headers
	canonicalHeaders, signedHeaders := s.canonicalHeaders(req)

	// Hashed payload
	hashedPayload := hex.EncodeToString(sha256Hash(body))

	// Create canonical request
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedPayload,
	}, "\n")

	return canonicalRequest, signedHeaders
}

// canonicalURI returns the canonical URI
func (s *Signer) canonicalURI(u *url.URL) string {
	path := u.Path
	if path == "" {
		path = "/"
	}
	// URI encode the path - AWS requires %20 for spaces, not +
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = uriEncode(segment, false)
	}
	return strings.Join(segments, "/")
}

// uriEncode performs URI encoding according to AWS requirements
func uriEncode(s string, encodeSlash bool) string {
	var encoded strings.Builder
	for _, r := range []byte(s) {
		if shouldEscape(r, encodeSlash) {
			encoded.WriteString(fmt.Sprintf("%%%02X", r))
		} else {
			encoded.WriteByte(r)
		}
	}
	return encoded.String()
}

// shouldEscape determines if a byte should be percent-encoded
func shouldEscape(c byte, encodeSlash bool) bool {
	// AWS signature version 4 requires specific encoding rules
	if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
		return false
	}
	switch c {
	case '-', '_', '.', '~':
		return false
	case '/':
		return encodeSlash
	default:
		return true
	}
}

// canonicalQueryString returns the canonical query string
func (s *Signer) canonicalQueryString(u *url.URL) string {
	query := u.Query()
	if len(query) == 0 {
		return ""
	}

	// Sort query parameters by key
	var keys []string
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build canonical query string
	var parts []string
	for _, k := range keys {
		for _, v := range query[k] {
			parts = append(parts, fmt.Sprintf("%s=%s",
				url.QueryEscape(k),
				url.QueryEscape(v),
			))
		}
	}

	return strings.Join(parts, "&")
}

// canonicalHeaders returns the canonical headers and signed headers list
func (s *Signer) canonicalHeaders(req *http.Request) (string, string) {
	// Get all headers to sign
	headers := make(map[string]string)
	for k, v := range req.Header {
		// Convert to lowercase
		key := strings.ToLower(k)
		// Join multiple values with comma
		value := strings.Join(v, ",")
		// Trim spaces
		headers[key] = strings.TrimSpace(value)
	}

	// Always include host header
	headers["host"] = req.Host
	if headers["host"] == "" {
		headers["host"] = req.URL.Host
	}

	// Sort header names
	var headerNames []string
	for k := range headers {
		headerNames = append(headerNames, k)
	}
	sort.Strings(headerNames)

	// Build canonical headers
	var canonicalHeaders []string
	for _, k := range headerNames {
		canonicalHeaders = append(canonicalHeaders, fmt.Sprintf("%s:%s", k, headers[k]))
	}

	return strings.Join(canonicalHeaders, "\n") + "\n", strings.Join(headerNames, ";")
}

// createStringToSign creates the string to sign
func (s *Signer) createStringToSign(canonicalRequest string, t time.Time) string {
	return strings.Join([]string{
		algorithm,
		t.Format(dateFormat),
		s.credentialScope(t),
		hex.EncodeToString(sha256Hash([]byte(canonicalRequest))),
	}, "\n")
}

// credentialScope returns the credential scope
func (s *Signer) credentialScope(t time.Time) string {
	return strings.Join([]string{
		t.Format(shortDateFormat),
		s.options.Region,
		s.options.Service,
		"aws4_request",
	}, "/")
}

// getSigningKey derives the signing key
func (s *Signer) getSigningKey(t time.Time) []byte {
	kDate := hmacSHA256([]byte("AWS4"+s.credentials.SecretAccessKey), []byte(t.Format(shortDateFormat)))
	kRegion := hmacSHA256(kDate, []byte(s.options.Region))
	kService := hmacSHA256(kRegion, []byte(s.options.Service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

// sha256Hash returns the SHA256 hash of the data
func sha256Hash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// hmacSHA256 returns the HMAC-SHA256 of the data
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}