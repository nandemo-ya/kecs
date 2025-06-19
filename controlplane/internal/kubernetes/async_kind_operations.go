package kubernetes

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AsyncKindOperations provides asynchronous Kind cluster operations
type AsyncKindOperations struct {
	kindManager *CachedKindManager
	operations  map[string]*AsyncOperation
	mu          sync.RWMutex
	workers     int
	queue       chan *operationRequest
	wg          sync.WaitGroup
}

// AsyncOperation represents an async operation
type AsyncOperation struct {
	ID        string
	Type      OperationType
	Status    OperationStatus
	ClusterName string
	StartTime time.Time
	EndTime   *time.Time
	Error     error
	Result    interface{}
	mu        sync.RWMutex
}

// OperationType represents the type of operation
type OperationType string

const (
	OperationCreateCluster OperationType = "CreateCluster"
	OperationDeleteCluster OperationType = "DeleteCluster"
	OperationScaleCluster  OperationType = "ScaleCluster"
)

// OperationStatus represents the status of an operation
type OperationStatus string

const (
	StatusPending    OperationStatus = "Pending"
	StatusInProgress OperationStatus = "InProgress"
	StatusCompleted  OperationStatus = "Completed"
	StatusFailed     OperationStatus = "Failed"
)

type operationRequest struct {
	operation *AsyncOperation
	callback  func(error)
}

// NewAsyncKindOperations creates a new async operations manager
func NewAsyncKindOperations(kindManager *CachedKindManager, workers int) *AsyncKindOperations {
	if workers <= 0 {
		workers = 3 // Default worker count
	}
	
	ops := &AsyncKindOperations{
		kindManager: kindManager,
		operations:  make(map[string]*AsyncOperation),
		workers:     workers,
		queue:       make(chan *operationRequest, 100),
	}
	
	// Start workers
	for i := 0; i < workers; i++ {
		ops.wg.Add(1)
		go ops.worker(i)
	}
	
	// Start cleanup routine
	go ops.cleanupLoop()
	
	return ops
}

// CreateClusterAsync creates a cluster asynchronously
func (ops *AsyncKindOperations) CreateClusterAsync(ctx context.Context, clusterName string, callback func(error)) string {
	operation := &AsyncOperation{
		ID:          generateOperationID(),
		Type:        OperationCreateCluster,
		Status:      StatusPending,
		ClusterName: clusterName,
		StartTime:   time.Now(),
	}
	
	ops.mu.Lock()
	ops.operations[operation.ID] = operation
	ops.mu.Unlock()
	
	// Queue the operation
	select {
	case ops.queue <- &operationRequest{operation: operation, callback: callback}:
		// Queued successfully
	case <-ctx.Done():
		// Context cancelled
		operation.mu.Lock()
		operation.Status = StatusFailed
		operation.Error = ctx.Err()
		now := time.Now()
		operation.EndTime = &now
		operation.mu.Unlock()
		
		if callback != nil {
			callback(ctx.Err())
		}
	}
	
	return operation.ID
}

// DeleteClusterAsync deletes a cluster asynchronously
func (ops *AsyncKindOperations) DeleteClusterAsync(ctx context.Context, clusterName string, callback func(error)) string {
	operation := &AsyncOperation{
		ID:          generateOperationID(),
		Type:        OperationDeleteCluster,
		Status:      StatusPending,
		ClusterName: clusterName,
		StartTime:   time.Now(),
	}
	
	ops.mu.Lock()
	ops.operations[operation.ID] = operation
	ops.mu.Unlock()
	
	// Queue the operation
	select {
	case ops.queue <- &operationRequest{operation: operation, callback: callback}:
		// Queued successfully
	case <-ctx.Done():
		// Context cancelled
		operation.mu.Lock()
		operation.Status = StatusFailed
		operation.Error = ctx.Err()
		now := time.Now()
		operation.EndTime = &now
		operation.mu.Unlock()
		
		if callback != nil {
			callback(ctx.Err())
		}
	}
	
	return operation.ID
}

// GetOperation returns the status of an operation
func (ops *AsyncKindOperations) GetOperation(operationID string) (*AsyncOperation, bool) {
	ops.mu.RLock()
	operation, exists := ops.operations[operationID]
	ops.mu.RUnlock()
	
	if !exists {
		return nil, false
	}
	
	// Return a copy to prevent concurrent access issues
	operation.mu.RLock()
	defer operation.mu.RUnlock()
	
	copy := &AsyncOperation{
		ID:          operation.ID,
		Type:        operation.Type,
		Status:      operation.Status,
		ClusterName: operation.ClusterName,
		StartTime:   operation.StartTime,
		EndTime:     operation.EndTime,
		Error:       operation.Error,
		Result:      operation.Result,
	}
	
	return copy, true
}

// ListOperations returns all operations
func (ops *AsyncKindOperations) ListOperations() []*AsyncOperation {
	ops.mu.RLock()
	defer ops.mu.RUnlock()
	
	operations := make([]*AsyncOperation, 0, len(ops.operations))
	for _, op := range ops.operations {
		op.mu.RLock()
		copy := &AsyncOperation{
			ID:          op.ID,
			Type:        op.Type,
			Status:      op.Status,
			ClusterName: op.ClusterName,
			StartTime:   op.StartTime,
			EndTime:     op.EndTime,
			Error:       op.Error,
			Result:      op.Result,
		}
		op.mu.RUnlock()
		operations = append(operations, copy)
	}
	
	return operations
}

// worker processes operations from the queue
func (ops *AsyncKindOperations) worker(id int) {
	defer ops.wg.Done()
	
	for req := range ops.queue {
		ops.processOperation(req)
	}
}

// processOperation executes an operation
func (ops *AsyncKindOperations) processOperation(req *operationRequest) {
	operation := req.operation
	
	// Update status to in progress
	operation.mu.Lock()
	operation.Status = StatusInProgress
	operation.mu.Unlock()
	
	// Execute the operation
	var err error
	ctx := context.Background()
	
	switch operation.Type {
	case OperationCreateCluster:
		err = ops.kindManager.CreateCluster(ctx, operation.ClusterName)
		
	case OperationDeleteCluster:
		err = ops.kindManager.DeleteCluster(ctx, operation.ClusterName)
		
	default:
		err = fmt.Errorf("unknown operation type: %s", operation.Type)
	}
	
	// Update operation status
	now := time.Now()
	operation.mu.Lock()
	operation.EndTime = &now
	if err != nil {
		operation.Status = StatusFailed
		operation.Error = err
	} else {
		operation.Status = StatusCompleted
	}
	operation.mu.Unlock()
	
	// Call callback if provided
	if req.callback != nil {
		req.callback(err)
	}
}

// cleanupLoop periodically removes old completed operations
func (ops *AsyncKindOperations) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		ops.cleanup()
	}
}

// cleanup removes operations older than 1 hour
func (ops *AsyncKindOperations) cleanup() {
	ops.mu.Lock()
	defer ops.mu.Unlock()
	
	now := time.Now()
	toRemove := []string{}
	
	for id, op := range ops.operations {
		op.mu.RLock()
		if op.EndTime != nil && now.Sub(*op.EndTime) > 1*time.Hour {
			toRemove = append(toRemove, id)
		}
		op.mu.RUnlock()
	}
	
	for _, id := range toRemove {
		delete(ops.operations, id)
	}
}

// Stats returns statistics about async operations
func (ops *AsyncKindOperations) Stats() AsyncOperationStats {
	ops.mu.RLock()
	defer ops.mu.RUnlock()
	
	stats := AsyncOperationStats{
		TotalOperations: len(ops.operations),
		QueuedOperations: len(ops.queue),
		WorkerCount:     ops.workers,
	}
	
	for _, op := range ops.operations {
		op.mu.RLock()
		switch op.Status {
		case StatusPending:
			stats.PendingOperations++
		case StatusInProgress:
			stats.InProgressOperations++
		case StatusCompleted:
			stats.CompletedOperations++
		case StatusFailed:
			stats.FailedOperations++
		}
		op.mu.RUnlock()
	}
	
	return stats
}

// Close stops all workers and waits for them to finish
func (ops *AsyncKindOperations) Close() {
	close(ops.queue)
	ops.wg.Wait()
}

// AsyncOperationStats contains statistics about async operations
type AsyncOperationStats struct {
	TotalOperations      int
	PendingOperations    int
	InProgressOperations int
	CompletedOperations  int
	FailedOperations     int
	QueuedOperations     int
	WorkerCount          int
}

// generateOperationID generates a unique operation ID
func generateOperationID() string {
	return fmt.Sprintf("op-%d-%d", time.Now().Unix(), time.Now().Nanosecond())
}