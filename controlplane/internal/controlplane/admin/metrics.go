package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// MetricsCollector collects and exposes metrics
type MetricsCollector struct {
	mu        sync.RWMutex
	startTime time.Time
	counters  map[string]int64
	gauges    map[string]float64
}

// Metrics represents the application metrics
type Metrics struct {
	Application ApplicationMetrics `json:"application"`
	Runtime     RuntimeMetrics     `json:"runtime"`
	API         APIMetrics         `json:"api"`
	Storage     StorageMetrics     `json:"storage"`
	WebSocket   WebSocketMetrics   `json:"websocket"`
}

// ApplicationMetrics contains application-level metrics
type ApplicationMetrics struct {
	Version        string    `json:"version"`
	StartTime      time.Time `json:"start_time"`
	Uptime         string    `json:"uptime"`
	UptimeSeconds  float64   `json:"uptime_seconds"`
}

// RuntimeMetrics contains Go runtime metrics
type RuntimeMetrics struct {
	GoVersion       string      `json:"go_version"`
	NumCPU          int         `json:"num_cpu"`
	NumGoroutine    int         `json:"num_goroutine"`
	MemoryStats     MemoryStats `json:"memory"`
}

// APIMetrics contains API-related metrics
type APIMetrics struct {
	TotalRequests   int64   `json:"total_requests"`
	ErrorCount      int64   `json:"error_count"`
	AverageLatency  float64 `json:"average_latency_ms"`
	ActiveRequests  int64   `json:"active_requests"`
}

// StorageMetrics contains storage-related metrics
type StorageMetrics struct {
	ConnectionStatus string `json:"connection_status"`
	QueryCount       int64  `json:"query_count"`
	ErrorCount       int64  `json:"error_count"`
}

// WebSocketMetrics contains WebSocket-related metrics
type WebSocketMetrics struct {
	ActiveConnections int64 `json:"active_connections"`
	TotalConnections  int64 `json:"total_connections"`
	MessagesSent      int64 `json:"messages_sent"`
	MessagesReceived  int64 `json:"messages_received"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
		counters:  make(map[string]int64),
		gauges:    make(map[string]float64),
	}
}

// IncrementCounter increments a counter metric
func (mc *MetricsCollector) IncrementCounter(name string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.counters[name]++
}

// SetGauge sets a gauge metric
func (mc *MetricsCollector) SetGauge(name string, value float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.gauges[name] = value
}

// GetMetrics returns current metrics
func (mc *MetricsCollector) GetMetrics() Metrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(mc.startTime)

	return Metrics{
		Application: ApplicationMetrics{
			Version:       getVersion(),
			StartTime:     mc.startTime,
			Uptime:        uptime.String(),
			UptimeSeconds: uptime.Seconds(),
		},
		Runtime: RuntimeMetrics{
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			MemoryStats: MemoryStats{
				Alloc:      m.Alloc,
				TotalAlloc: m.TotalAlloc,
				Sys:        m.Sys,
				NumGC:      m.NumGC,
			},
		},
		API: APIMetrics{
			TotalRequests:  mc.counters["api.requests.total"],
			ErrorCount:     mc.counters["api.errors.total"],
			AverageLatency: mc.gauges["api.latency.average"],
			ActiveRequests: mc.counters["api.requests.active"],
		},
		Storage: StorageMetrics{
			ConnectionStatus: "connected", // TODO: Get actual status
			QueryCount:       mc.counters["storage.queries.total"],
			ErrorCount:       mc.counters["storage.errors.total"],
		},
		WebSocket: WebSocketMetrics{
			ActiveConnections: mc.counters["websocket.connections.active"],
			TotalConnections:  mc.counters["websocket.connections.total"],
			MessagesSent:      mc.counters["websocket.messages.sent"],
			MessagesReceived:  mc.counters["websocket.messages.received"],
		},
	}
}

// handleMetrics handles the metrics endpoint
func (s *Server) handleMetrics(collector *MetricsCollector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := collector.GetMetrics()
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(metrics)
	}
}

// handlePrometheusMetrics handles Prometheus-format metrics endpoint
func (s *Server) handlePrometheusMetrics(collector *MetricsCollector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := collector.GetMetrics()
		
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		
		// Write metrics in Prometheus format
		// Application metrics
		fmt.Fprintf(w, "# HELP kecs_up Whether the KECS server is up\n")
		fmt.Fprintf(w, "# TYPE kecs_up gauge\n")
		fmt.Fprintf(w, "kecs_up 1\n")
		
		fmt.Fprintf(w, "# HELP kecs_uptime_seconds Time since KECS server started in seconds\n")
		fmt.Fprintf(w, "# TYPE kecs_uptime_seconds counter\n")
		fmt.Fprintf(w, "kecs_uptime_seconds %f\n", metrics.Application.UptimeSeconds)
		
		// Runtime metrics
		fmt.Fprintf(w, "# HELP kecs_goroutines Number of goroutines\n")
		fmt.Fprintf(w, "# TYPE kecs_goroutines gauge\n")
		fmt.Fprintf(w, "kecs_goroutines %d\n", metrics.Runtime.NumGoroutine)
		
		fmt.Fprintf(w, "# HELP kecs_memory_alloc_bytes Current memory allocation in bytes\n")
		fmt.Fprintf(w, "# TYPE kecs_memory_alloc_bytes gauge\n")
		fmt.Fprintf(w, "kecs_memory_alloc_bytes %d\n", metrics.Runtime.MemoryStats.Alloc)
		
		// API metrics
		fmt.Fprintf(w, "# HELP kecs_api_requests_total Total number of API requests\n")
		fmt.Fprintf(w, "# TYPE kecs_api_requests_total counter\n")
		fmt.Fprintf(w, "kecs_api_requests_total %d\n", metrics.API.TotalRequests)
		
		fmt.Fprintf(w, "# HELP kecs_api_errors_total Total number of API errors\n")
		fmt.Fprintf(w, "# TYPE kecs_api_errors_total counter\n")
		fmt.Fprintf(w, "kecs_api_errors_total %d\n", metrics.API.ErrorCount)
		
		// WebSocket metrics
		fmt.Fprintf(w, "# HELP kecs_websocket_connections_active Number of active WebSocket connections\n")
		fmt.Fprintf(w, "# TYPE kecs_websocket_connections_active gauge\n")
		fmt.Fprintf(w, "kecs_websocket_connections_active %d\n", metrics.WebSocket.ActiveConnections)
	}
}