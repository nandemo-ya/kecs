package awsclient

import (
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries     int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	BackoffFactor  float64
	JitterEnabled  bool
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      20 * time.Second,
		BackoffFactor: 2.0,
		JitterEnabled: true,
	}
}

// Retryer handles retry logic with exponential backoff
type Retryer struct {
	config RetryConfig
	rng    *rand.Rand
}

// NewRetryer creates a new retryer
func NewRetryer(config RetryConfig) *Retryer {
	return &Retryer{
		config: config,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RetryDelay calculates the delay for the given attempt number
func (r *Retryer) RetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate exponential backoff
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.BackoffFactor, float64(attempt-1))
	
	// Cap at max delay
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	// Add jitter if enabled
	if r.config.JitterEnabled {
		// Add random jitter between 0% and 25% of the delay
		jitter := r.rng.Float64() * 0.25 * delay
		delay += jitter
	}

	return time.Duration(delay)
}

// ShouldRetry determines if a request should be retried
func (r *Retryer) ShouldRetry(attempt int) bool {
	return attempt < r.config.MaxRetries
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// TODO: Add more sophisticated error checking
	// For now, we'll consider network errors as retryable
	return true
}

// IsThrottleError determines if an error is a throttling error
func IsThrottleError(statusCode int) bool {
	return statusCode == 429 || statusCode == 503
}