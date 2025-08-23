package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// HealthChecker provides health check functionality
type HealthChecker struct {
	storage     storage.Storage
	startTime   time.Time
	mu          sync.RWMutex
	checks      map[string]CheckFunc
	lastResults map[string]CheckResult
}

// CheckFunc is a function that performs a health check
type CheckFunc func(ctx context.Context) error

// CheckResult represents the result of a health check
type CheckResult struct {
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// MemoryStats contains memory usage statistics (used by metrics.go as well)
type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(storage storage.Storage) *HealthChecker {
	hc := &HealthChecker{
		storage:     storage,
		startTime:   time.Now(),
		checks:      make(map[string]CheckFunc),
		lastResults: make(map[string]CheckResult),
	}

	// Register default checks
	hc.RegisterCheck("storage", hc.checkStorage)
	hc.RegisterCheck("kubernetes", hc.checkKubernetes)

	return hc
}

// RegisterCheck registers a new health check
func (hc *HealthChecker) RegisterCheck(name string, check CheckFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = check
}

// RunChecks runs all registered health checks
func (hc *HealthChecker) RunChecks(ctx context.Context) map[string]CheckResult {
	hc.mu.RLock()
	checks := make(map[string]CheckFunc)
	for name, check := range hc.checks {
		checks[name] = check
	}
	hc.mu.RUnlock()

	results := make(map[string]CheckResult)
	var wg sync.WaitGroup

	for name, check := range checks {
		wg.Add(1)
		go func(n string, c CheckFunc) {
			defer wg.Done()

			// Create a timeout context for each check
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			result := CheckResult{
				Status:    "healthy",
				Timestamp: time.Now(),
			}

			if err := c(checkCtx); err != nil {
				result.Status = "unhealthy"
				result.Message = err.Error()
			}

			hc.mu.Lock()
			hc.lastResults[n] = result
			hc.mu.Unlock()

			results[n] = result
		}(name, check)
	}

	wg.Wait()
	return results
}

// Health check implementations

func (hc *HealthChecker) checkStorage(ctx context.Context) error {
	// Simple ping to check if storage is accessible
	// This is a placeholder - implement actual storage health check
	if hc.storage == nil {
		return nil // Storage might be optional
	}
	return nil
}

func (hc *HealthChecker) checkKubernetes(ctx context.Context) error {
	// Check Kubernetes API connectivity
	// This is a placeholder - implement actual k8s health check
	return nil
}

// HTTP handlers

// handleHealth handles the simple health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handleReadiness handles the readiness probe endpoint
func (s *Server) handleReadiness(checker *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Run critical checks only
		checker.mu.RLock()
		storageCheck := checker.checks["storage"]
		checker.mu.RUnlock()

		if storageCheck != nil {
			if err := storageCheck(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{
					"status": "not ready",
					"reason": err.Error(),
				})
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		})
	}
}

// handleLiveness handles the liveness probe endpoint
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - if we can respond, we're alive
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
	})
}

// getVersion returns the application version
func getVersion() string {
	// This should be set during build time
	// For now, return a placeholder
	return "dev"
}
